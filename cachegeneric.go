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

// MGet efficiently fetches multiple values associated with a specific key prefix and a list of IDs in a single call.
// It prioritizes retrieving data from the cache and falls back to a user-provided function (`fn`) if values are missing.
// Asynchronous refresh is not supported.
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) map[K]V {
	c := w.Cache.(*jetCache)
	values, missIds := w.mGetCache(ctx, key, ids)

	if len(missIds) > 0 && fn != nil {
		sort.Slice(missIds, func(i, j int) bool {
			return missIds[i] < missIds[j]
		})
		v, err, _ := c.group.Do(fmt.Sprintf("%s:%v", key, missIds), func() (interface{}, error) {
			c.statsHandler.IncrQuery()
			v, err := fn(ctx, missIds)
			if err != nil {
				c.statsHandler.IncrQueryFail(err)
			}
			return v, err
		})

		if err != nil {
			return values
		}

		fnValues := v.(map[K]V)

		placeholderValues := make(map[string]any, len(ids))
		cacheValues := make(map[string]any, len(ids))
		for rk, rv := range fnValues {
			values[rk] = rv
			cacheKey := util.JoinAny(":", key, rk)
			if b, err := c.Marshal(rv); err != nil {
				placeholderValues[cacheKey] = notFoundPlaceholder
				logger.Warn("MGet#w.Marshal(%v) error(%v)", rv, err)
			} else {
				cacheValues[cacheKey] = b
			}
		}

		for _, missId := range missIds {
			if _, ok := values[missId]; !ok {
				cacheKey := util.JoinAny(":", key, missId)
				placeholderValues[cacheKey] = notFoundPlaceholder
			}
		}

		if c.local != nil {
			if len(placeholderValues) > 0 {
				for key, value := range placeholderValues {
					c.local.Set(key, value.([]byte))
				}
			}
			if len(cacheValues) > 0 {
				for key, value := range cacheValues {
					c.local.Set(key, value.([]byte))
				}
			}
		}

		if c.remote != nil {
			if len(placeholderValues) > 0 {
				if err = c.remote.MSet(ctx, placeholderValues, c.notFoundExpiry); err != nil {
					logger.Warn("MGet#remote.MSet error(%v)", err)
				}
			}
			if len(cacheValues) > 0 {
				if err = c.remote.MSet(ctx, cacheValues, c.remoteExpiry); err != nil {
					logger.Warn("MGet#remote.MSet error(%v)", err)
				}
			}
			if c.isSyncLocal() {
				cacheKeys := make([]string, 0, len(missIds))
				for _, missId := range missIds {
					cacheKey := util.JoinAny(":", key, missId)
					cacheKeys = append(cacheKeys, cacheKey)
				}
				c.send(EventTypeSetByMGet, cacheKeys...)
			}
		}
	}

	return values
}

func (w *T[K, V]) mGetCache(ctx context.Context, key string, ids []K) (v map[K]V, missIds []K) {
	c := w.Cache.(*jetCache)
	v = make(map[K]V, len(ids))
	miss := make(map[string]K, len(ids))

	for _, id := range ids {
		cacheKey := util.JoinAny(":", key, id)
		if c.local != nil {
			if b, ok := c.local.Get(cacheKey); ok {
				c.statsHandler.IncrHit()
				c.statsHandler.IncrLocalHit()
				if bytes.Compare(b, notFoundPlaceholder) == 0 {
					continue
				}
				var varT V
				if err := c.Unmarshal(b, &varT); err != nil {
					logger.Warn("mGetCache#c.Unmarshal(%s) error(%v)", cacheKey, err)
				} else {
					v[id] = varT
				}
			} else {
				miss[cacheKey] = id
				c.statsHandler.IncrLocalMiss()
				if c.remote != nil {
					c.statsHandler.IncrMiss()
				}
			}
		} else {
			miss[cacheKey] = id
		}
	}

	if len(miss) > 0 && c.remote != nil {
		missKeys := make([]string, 0, len(miss))
		for k := range miss {
			missKeys = append(missKeys, k)
		}
		if values, err := c.remote.MGet(ctx, missKeys...); err == nil {
			for mk, mv := range miss {
				if val, ok := values[mk]; ok {
					c.statsHandler.IncrHit()
					c.statsHandler.IncrRemoteHit()
					delete(miss, mk)
					b := util.Bytes(val.(string))
					if bytes.Compare(b, notFoundPlaceholder) == 0 {
						continue
					}
					var varT V
					if err = c.Unmarshal(b, &varT); err != nil {
						logger.Warn("mGetCache#c.Unmarshal(%s) error(%v)", mk, err)
					} else {
						v[mv] = varT
						if c.local != nil {
							c.local.Set(mk, b)
						}
					}
				} else {
					c.statsHandler.IncrMiss()
					c.statsHandler.IncrRemoteMiss()
				}
			}
		}
	}

	for _, mv := range miss {
		missIds = append(missIds, mv)
		c.statsHandler.IncrMiss()
	}

	return
}
