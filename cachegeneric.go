package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"golang.org/x/exp/constraints"

	"github.com/mgtv-tech/jetcache-go/logger"
	"github.com/mgtv-tech/jetcache-go/util"
)

// T wrap Cache to support golang's generics
type T[K constraints.Ordered, V any] struct {
	Cache
}

// NewT new a T
func NewT[K constraints.Ordered, V any](cache Cache) *T[K, V] {
	return &T[K, V]{cache}
}

// Set sets the value `v` associated with the given `key` and `id` in the cache.
// The expiration time of the cached value is determined by the cache configuration.
func (w *T[K, V]) Set(ctx context.Context, key string, id K, v V) error {
	c := w.Cache.(*jetCache)

	combKey := fmt.Sprintf("%s%s%v", key, c.separator, id)
	return w.Cache.Set(ctx, combKey, Value(v))
}

// Get retrieves the value associated with the given `key` and `id`.
//
// It first attempts to fetch the value from the cache. If a cache miss occurs, it calls the provided
// `fn` function to fetch the value and stores it in the cache with an expiration time
// determined by the cache configuration.
//
// A `Once` mechanism is employed to ensure only one fetch is performed for a given `key` and `id`
// combination, even under concurrent access.
func (w *T[K, V]) Get(ctx context.Context, key string, id K, fn func(context.Context, K) (V, error)) (V, error) {
	c := w.Cache.(*jetCache)

	var varT V
	combKey := fmt.Sprintf("%s%s%v", key, c.separator, id)
	err := w.Once(ctx, combKey, Value(&varT), Do(func(ctx context.Context) (any, error) {
		return fn(ctx, id)
	}))

	return varT, err
}

// MGet efficiently retrieves multiple values associated with the given `key` and `ids`.
// It is a wrapper around MGetWithErr that logs any errors and returns only the results.
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V) {
	var err error
	if result, err = w.MGetWithErr(ctx, key, ids, fn); err != nil {
		logger.Warn("MGet error(%v)", err)
	}

	return
}

// MGetWithErr efficiently retrieves multiple values associated with the given `key` and `ids`,
// returning both the results and any errors encountered during the process.
//
// It first attempts to retrieve values from the local cache (if enabled), then from the remote cache (if enabled).
// For any values not found in the caches, it calls the provided `fn` function to fetch them from the
// underlying data source. The fetched values are then stored in both the local and remote caches for
// future use.
//
// The results are returned as a map where the key is the `id` and the value is the corresponding data.
// Any errors encountered during the cache retrieval or data fetching process are returned as a non-nil error.
func (w *T[K, V]) MGetWithErr(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V, errs error) {
	c := w.Cache.(*jetCache)

	miss := make(map[string]K, len(ids))
	for _, missId := range ids {
		missKey := fmt.Sprintf("%s%s%v", key, c.separator, missId)
		miss[missKey] = missId
	}

	if c.local != nil {
		result, errs = w.mGetLocal(miss, true)
		if len(miss) == 0 {
			return
		}
	}

	if c.remote == nil && fn == nil {
		return
	}

	missIds := make([]K, 0, len(miss))
	for _, missId := range miss {
		missIds = append(missIds, missId)
	}

	sort.Slice(missIds, func(i, j int) bool {
		return missIds[i] < missIds[j]
	})

	combKey := fmt.Sprintf("%s%s%v", key, c.separator, missIds)
	v, err, _ := c.group.Do(combKey, func() (interface{}, error) {
		var ret map[K]V

		process := func(r map[K]V, e error) {
			errs = errors.Join(errs, e)
			ret = util.MergeMap(ret, r)
		}

		if c.local != nil {
			process(w.mGetLocal(miss, false))
			if len(miss) == 0 {
				return ret, nil
			}
		}

		if c.remote != nil {
			process(w.mGetRemote(ctx, miss))
			if len(miss) == 0 {
				return ret, nil
			}
		}

		if fn != nil {
			process(w.mQueryAndSetCache(ctx, miss, fn))
		}

		return ret, nil
	})

	if err != nil {
		errs = errors.Join(errs, err)
		return
	}

	return util.MergeMap(result, v.(map[K]V)), errs
}

func (w *T[K, V]) mGetLocal(miss map[string]K, skipMissStats bool) (result map[K]V, errs error) {
	c := w.Cache.(*jetCache)

	result = make(map[K]V, len(miss))
	for missKey, missId := range miss {
		if b, ok := c.local.Get(missKey); ok {
			delete(miss, missKey)
			c.statsHandler.IncrHit()
			c.statsHandler.IncrLocalHit()
			if bytes.Compare(b, notFoundPlaceholder) == 0 {
				continue
			}
			var varT V
			if err := c.Unmarshal(b, &varT); err != nil {
				errs = errors.Join(errs, fmt.Errorf("mGetLocal#c.Unmarshal(%s) error(%v)", missKey, err))
			} else {
				result[missId] = varT
			}
		} else if !skipMissStats {
			c.statsHandler.IncrLocalMiss()
			if c.remote == nil {
				c.statsHandler.IncrMiss()
			}
		}
	}

	return
}

func (w *T[K, V]) mGetRemote(ctx context.Context, miss map[string]K) (result map[K]V, errs error) {
	c := w.Cache.(*jetCache)

	missKeys := make([]string, 0, len(miss))
	for missKey := range miss {
		missKeys = append(missKeys, missKey)
	}

	cacheValues, err := c.remote.MGet(ctx, missKeys...)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("mGetRemote#c.Remote.MGet error(%v)", err))
		return
	}

	result = make(map[K]V, len(cacheValues))
	for missKey, missId := range miss {
		if val, ok := cacheValues[missKey]; ok {
			delete(miss, missKey)
			c.statsHandler.IncrHit()
			c.statsHandler.IncrRemoteHit()
			b := util.Bytes(val.(string))
			if bytes.Compare(b, notFoundPlaceholder) == 0 {
				continue
			}
			var varT V
			if err = c.Unmarshal(b, &varT); err != nil {
				errs = errors.Join(errs, fmt.Errorf("mGetRemote#c.Unmarshal(%s) error(%v)", missKey, err))
			} else {
				result[missId] = varT
				if c.local != nil {
					c.local.Set(missKey, b)
				}
			}
		} else {
			c.statsHandler.IncrMiss()
			c.statsHandler.IncrRemoteMiss()
		}
	}

	return
}

func (w *T[K, V]) mQueryAndSetCache(ctx context.Context, miss map[string]K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V, errs error) {
	c := w.Cache.(*jetCache)

	missIds := make([]K, 0, len(miss))
	for _, missId := range miss {
		missIds = append(missIds, missId)
	}

	c.statsHandler.IncrQuery()
	fnValues, err := fn(ctx, missIds)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("mQueryAndSetCache#fn(%v) error(%v)", missIds, err))
		c.statsHandler.IncrQueryFail(err)
		return
	}

	result = make(map[K]V, len(fnValues))
	cacheValues := make(map[string]any, len(miss))
	placeholderValues := make(map[string]any, len(miss))
	for missKey, missId := range miss {
		if val, ok := fnValues[missId]; ok {
			result[missId] = val
			if b, err := c.Marshal(val); err != nil {
				placeholderValues[missKey] = notFoundPlaceholder
				errs = errors.Join(errs, fmt.Errorf("mQueryAndSetCache#c.Marshal error(%v)", err))
			} else {
				cacheValues[missKey] = b
			}
		} else {
			placeholderValues[missKey] = notFoundPlaceholder
		}
	}

	if c.local != nil {
		if len(cacheValues) > 0 {
			for key, value := range cacheValues {
				c.local.Set(key, value.([]byte))
			}
		}
		if len(placeholderValues) > 0 {
			for key, value := range placeholderValues {
				c.local.Set(key, value.([]byte))
			}
		}
	}

	if c.remote != nil {
		if len(cacheValues) > 0 {
			if err = c.remote.MSet(ctx, cacheValues, c.remoteExpiry); err != nil {
				errs = errors.Join(errs, fmt.Errorf("mQueryAndSetCache#c.Remote.MSet error(%v)", err))
			}
		}
		if len(placeholderValues) > 0 {
			if err = c.remote.MSet(ctx, placeholderValues, c.notFoundExpiry); err != nil {
				errs = errors.Join(errs, fmt.Errorf("mQueryAndSetCache#c.Remote.MSet error(%v)", err))
			}
		}
		if c.isSyncLocal() {
			cacheKeys := make([]string, 0, len(miss))
			for missKey := range miss {
				cacheKeys = append(cacheKeys, missKey)
			}
			c.send(EventTypeSetByMGet, cacheKeys...)
		}
	}

	return
}
