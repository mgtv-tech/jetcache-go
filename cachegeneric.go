package cache

import (
	"bytes"
	"context"

	"github.com/daoshenzzg/jetcache-go/logger"
	"github.com/daoshenzzg/jetcache-go/util"
)

// T wrap Cache to support golang's generics
type T[K comparable, V any] struct {
	Cache
}

// NewT new a T
func NewT[K comparable, V any](cache Cache) *T[K, V] {
	return &T[K, V]{cache}
}

// MGet efficiently fetches multiple values associated with a specific key prefix and a list of IDs in a single call.
// It prioritizes retrieving data from the cache and falls back to a user-provided function (`fn`) if values are missing.
// Asynchronous refresh is not supported.
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) map[K]V {
	c := w.Cache.(*jetCache)
	values, missIds := w.mGetCache(ctx, key, ids)

	if len(missIds) > 0 && fn != nil {
		c.statsHandler.IncrQuery()

		fnValues, err := fn(ctx, missIds)
		if err != nil {
			c.statsHandler.IncrQueryFail(err)
			logger.Error("MGet#fn(%s) error(%v)", util.JoinAny(",", ids), err)
		} else {
			placeholderValues := make(map[string]any, len(ids))
			cacheValues := make(map[string]any, len(ids))
			for rk, rv := range fnValues {
				values[rk] = rv
				cacheKey := util.JoinAny(":", key, rk)
				if b, err := c.Marshal(rv); err != nil {
					placeholderValues[cacheKey] = notFoundPlaceholder
					logger.Error("MGet#w.Marshal(%v) error(%v)", rv, err)
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
						logger.Error("MGet#MSet error(%v)", err)
					}
				}
				if len(cacheValues) > 0 {
					if err = c.remote.MSet(ctx, cacheValues, c.remoteExpiry); err != nil {
						logger.Error("MGet#MSet error(%v)", err)
					}
				}
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
					logger.Error("mGetCache#c.Unmarshal(%s) error(%v)", cacheKey, err)
				} else {
					v[id] = varT
				}
			} else {
				miss[cacheKey] = id
				c.statsHandler.IncrLocalMiss()
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
						logger.Error("mGetCache#c.Unmarshal(%s) error(%v)", mk, err)
					} else {
						v[mv] = varT
						if c.local != nil {
							c.local.Set(mk, b)
						}
					}
				} else {
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