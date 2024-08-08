package cache

import (
	"bytes"
	"context"
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

	combKey := fmt.Sprintf("%s:%v", key, id)
	return c.Set(ctx, combKey, Value(v))
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
	combKey := fmt.Sprintf("%s:%v", key, id)
	err := c.Once(ctx, combKey, Value(&varT), Do(func(ctx context.Context) (any, error) {
		return fn(ctx, id)
	}))

	return varT, err
}

// MGet efficiently retrieves multiple values associated with the given `key` and `ids`.
//
// It attempts to fetch all values from the cache. For missing values, it calls the provided
// `fn` function to fetch the remaining values and updates the cache with an expiration time
// determined by the cache configuration.
//
// The results are returned as a map where the key is the `id` and the value is the corresponding data.
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V) {
	c := w.Cache.(*jetCache)

	miss := make(map[string]K, len(ids))
	for _, missId := range ids {
		missKey := fmt.Sprintf("%s:%v", key, missId)
		miss[missKey] = missId
	}

	if c.local != nil {
		result = w.mGetLocal(miss, true)
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

	combKey := fmt.Sprintf("%s:%v", key, missIds)
	v, err, _ := c.group.Do(combKey, func() (interface{}, error) {
		var ret map[K]V
		if c.local != nil {
			ret = w.mGetLocal(miss, false)
			if len(miss) == 0 {
				return ret, nil
			}
		}

		if c.remote != nil {
			ret = util.MergeMap(ret, w.mGetRemote(ctx, miss))
			if len(miss) == 0 {
				return ret, nil
			}
		}

		if fn != nil {
			ret = util.MergeMap(ret, w.mQueryAndSetCache(ctx, miss, fn))
		}

		return ret, nil
	})

	if err != nil {
		return result
	}

	return util.MergeMap(result, v.(map[K]V))
}

func (w *T[K, V]) mGetLocal(miss map[string]K, skipMissStats bool) map[K]V {
	c := w.Cache.(*jetCache)

	result := make(map[K]V, len(miss))
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
				logger.Warn("mGetLocal#c.Unmarshal(%s) error(%v)", missKey, err)
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

	return result
}

func (w *T[K, V]) mGetRemote(ctx context.Context, miss map[string]K) map[K]V {
	c := w.Cache.(*jetCache)

	missKeys := make([]string, 0, len(miss))
	for missKey := range miss {
		missKeys = append(missKeys, missKey)
	}

	cacheValues, err := c.remote.MGet(ctx, missKeys...)
	if err != nil {
		logger.Warn("mGetRemote#c.Remote.MGet error(%v)", err)
		return nil
	}

	result := make(map[K]V, len(cacheValues))
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
				logger.Warn("mGetRemote#c.Unmarshal(%s) error(%v)", missKey, err)
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

	return result
}

func (w *T[K, V]) mQueryAndSetCache(ctx context.Context, miss map[string]K, fn func(context.Context, []K) (map[K]V, error)) map[K]V {
	c := w.Cache.(*jetCache)

	missIds := make([]K, 0, len(miss))
	for _, missId := range miss {
		missIds = append(missIds, missId)
	}

	c.statsHandler.IncrQuery()
	fnValues, err := fn(ctx, missIds)
	if err != nil {
		c.statsHandler.IncrQueryFail(err)
		return nil
	}

	result := make(map[K]V, len(fnValues))
	cacheValues := make(map[string]any, len(miss))
	placeholderValues := make(map[string]any, len(miss))
	for missKey, missId := range miss {
		if val, ok := fnValues[missId]; ok {
			result[missId] = val
			if b, err := c.Marshal(val); err != nil {
				placeholderValues[missKey] = notFoundPlaceholder
				logger.Warn("mQueryAndSetCache#c.Marshal error(%v)", err)
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
				logger.Warn("mQueryAndSetCache#remote.MSet error(%v)", err)
			}
		}
		if len(placeholderValues) > 0 {
			if err = c.remote.MSet(ctx, placeholderValues, c.notFoundExpiry); err != nil {
				logger.Warn("mQueryAndSetCache#remote.MSet error(%v)", err)
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

	return result
}
