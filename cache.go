package cache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"

	"github.com/daoshenzzg/jetcache-go/encoding"
	"github.com/daoshenzzg/jetcache-go/logger"
	"github.com/daoshenzzg/jetcache-go/util"
)

const (
	TypeLocal  = "local"
	TypeRemote = "remote"
	TypeBoth   = "both"

	lockKeySuffix = "_#RL#"
)

var (
	notFoundPlaceholder   = []byte("*")
	ErrCacheMiss          = errors.New("cache: key is missing")
	ErrRemoteLocalBothNil = errors.New("cache: both remote and local are nil")
)

type (
	// Cache interface is used to define the cache implementation.
	Cache[K comparable, V any] interface {
		// Set sets the cache with ItemOption
		Set(ctx context.Context, key string, opts ...ItemOption) error
		// Once gets the opts.value for the given key from the cache or
		// executes, caches, and returns the results of the given opts.do,
		// making sure that only one execution is in-flight for a given key
		// at a time. If a duplicate comes in, the duplicate caller waits for the
		// original to complete and receives the same results.
		Once(ctx context.Context, key string, opts ...ItemOption) error
		// Delete deletes cached val with key.
		Delete(ctx context.Context, key string) error
		// DeleteFromLocalCache deletes local cached val with key.
		DeleteFromLocalCache(key string)
		// Exists reports whether val for the given key exists.
		Exists(ctx context.Context, key string) bool
		// Get gets the val for the given key and fills into val.
		Get(ctx context.Context, key string, val any) error
		// GetSkippingLocal gets the val for the given key skipping local cache.
		GetSkippingLocal(ctx context.Context, key string, val any) error
		// MGet fetches multiple values with a given key prefix and IDs in one call.
		// Does not support asynchronous refresh.
		MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) map[K]V
		// TaskSize returns Refresh task size.
		TaskSize() int
		// CacheType returns cache type
		CacheType() string
		// Close closes the cache. This should be called when cache refreshing is
		// enabled and no longer needed, or when it may lead to resource leaks.
		Close()
	}

	jetCache[K comparable, V any] struct {
		sync.Mutex
		Options
		group          singleflight.Group
		safeRand       *util.SafeRand
		refreshTaskMap sync.Map
		stopChan       chan struct{}
	}
)

func New[K comparable, V any](opts ...Option) Cache[K, V] {
	o := newOptions(opts...)
	cache := &jetCache[K, V]{
		Options:  o,
		safeRand: util.NewSafeRand(),
		stopChan: make(chan struct{}),
	}

	if cache.refreshDuration > 0 {
		go util.WithRecover(func() {
			cache.tick()
		})
	}

	return cache
}

func (c *jetCache[K, V]) Set(ctx context.Context, key string, opts ...ItemOption) error {
	_, _, err := c.set(newItemOptions(ctx, key, opts...))
	return err
}

func (c *jetCache[K, V]) set(item *item) ([]byte, bool, error) {
	val, err := item.getValue()
	if item.do != nil {
		c.statsHandler.IncrQuery()
	}

	if c.IsNotFound(err) {
		if e := c.setNotFound(item.Context(), item.key, item.skipLocal); e != nil {
			logger.Error("setNotFound(%s) error(%v)", item.key, err)
		}
		return notFoundPlaceholder, true, nil
	} else if err != nil {
		c.statsHandler.IncrQueryFail(err)
		return nil, false, err
	}

	b, err := c.Marshal(val)
	if err != nil {
		return nil, false, err
	}

	if c.local != nil && !item.skipLocal {
		c.local.Set(item.key, b)
	}

	if c.remote == nil {
		if c.local == nil {
			return b, true, ErrRemoteLocalBothNil
		}
		return b, true, nil
	}

	ttl := item.getTtl(c.remoteExpiry)
	if ttl == 0 {
		return b, true, nil
	}

	if item.setXX {
		_, err := c.remote.SetXX(item.Context(), item.key, b, ttl)
		return b, true, err
	}
	if item.setNX {
		_, err := c.remote.SetNX(item.Context(), item.key, b, ttl)
		return b, true, err
	}
	return b, true, c.remote.SetEX(item.Context(), item.key, b, ttl)
}

func (c *jetCache[K, V]) Exists(ctx context.Context, key string) bool {
	_, err := c.getBytes(ctx, key, false)
	return err == nil
}

func (c *jetCache[K, V]) Get(ctx context.Context, key string, val any) error {
	return c.get(ctx, key, val, false)
}

func (c *jetCache[K, V]) GetSkippingLocal(ctx context.Context, key string, val any) error {
	return c.get(ctx, key, val, true)
}

func (c *jetCache[K, V]) get(ctx context.Context, key string, val any, skipLocal bool) error {
	b, err := c.getBytes(ctx, key, skipLocal)
	if err != nil {
		return err
	}

	return c.Unmarshal(b, val)
}

func (c *jetCache[K, V]) getBytes(ctx context.Context, key string, skipLocal bool) ([]byte, error) {
	if !skipLocal && c.local != nil {
		b, ok := c.local.Get(key)
		if ok {
			c.statsHandler.IncrHit()
			c.statsHandler.IncrLocalHit()
			if bytes.Compare(b, notFoundPlaceholder) == 0 {
				return nil, c.errNotFound
			}
			return b, nil
		}
		c.statsHandler.IncrLocalMiss()
	}

	if c.remote == nil {
		if c.local == nil {
			return nil, ErrRemoteLocalBothNil
		}
		c.statsHandler.IncrMiss()
		return nil, ErrCacheMiss
	}

	s, err := c.remote.Get(ctx, key)
	if err != nil {
		c.statsHandler.IncrMiss()
		c.statsHandler.IncrRemoteMiss()
		if errors.Is(err, c.remote.Nil()) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	c.statsHandler.IncrHit()
	c.statsHandler.IncrRemoteHit()

	b := util.Bytes(s)
	if bytes.Compare(b, notFoundPlaceholder) == 0 {
		return nil, c.errNotFound
	}

	if !skipLocal && c.local != nil {
		c.local.Set(key, b)
	}

	return b, nil
}

func (c *jetCache[K, V]) Once(ctx context.Context, key string, opts ...ItemOption) error {
	item := newItemOptions(ctx, key, opts...)

	c.addOrUpdateRefreshTask(item)

	b, cached, err := c.getSetItemBytesOnce(item)
	if err != nil {
		return err
	}

	if bytes.Compare(b, notFoundPlaceholder) == 0 {
		return c.errNotFound
	}

	if item.value == nil || len(b) == 0 {
		return nil
	}

	if err := c.Unmarshal(b, item.value); err != nil {
		if cached {
			_ = c.Delete(ctx, item.key)
			return c.Once(ctx, key, opts...)
		}
		return err
	}

	return nil
}

func (c *jetCache[K, V]) getSetItemBytesOnce(item *item) (b []byte, cached bool, err error) {
	if c.local != nil {
		b, ok := c.local.Get(item.key)
		if ok {
			c.statsHandler.IncrHit()
			c.statsHandler.IncrLocalHit()
			if bytes.Compare(b, notFoundPlaceholder) == 0 {
				return nil, true, c.errNotFound
			}
			return b, true, nil
		}
	}

	v, err, _ := c.group.Do(item.key, func() (any, error) {
		b, err := c.getBytes(item.Context(), item.key, item.skipLocal)
		if err == nil {
			cached = true
			return b, nil
		} else if errors.Is(err, c.errNotFound) {
			cached = true
			return nil, c.errNotFound
		}

		b, ok, err := c.set(item)
		if ok {
			return b, nil
		}
		return nil, err
	})

	if err != nil {
		return nil, false, err
	}

	return v.([]byte), cached, nil
}

func (c *jetCache[K, V]) Delete(ctx context.Context, key string) error {
	if c.local != nil {
		c.local.Del(key)
	}

	if c.remote == nil {
		if c.local == nil {
			return ErrRemoteLocalBothNil
		}
		return nil
	}

	_, err := c.remote.Del(ctx, key)

	return err
}

func (c *jetCache[K, V]) DeleteFromLocalCache(key string) {
	if c.local != nil {
		c.local.Del(key)
	}
}

func (c *jetCache[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) map[K]V {
	values, missIds := c.mGetCache(ctx, key, ids)

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
					logger.Error("MGet#c.Marshal(%v) error(%v)", rv, err)
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

func (c *jetCache[K, V]) mGetCache(ctx context.Context, key string, ids []K) (v map[K]V, missIds []K) {
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

func (c *jetCache[K, V]) IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, c.errNotFound)
}

func (c *jetCache[K, V]) setNotFound(ctx context.Context, key string, skipLocal bool) error {
	if c.local != nil && !skipLocal {
		c.local.Set(key, notFoundPlaceholder)
	}

	if c.remote == nil {
		if c.local == nil {
			return ErrRemoteLocalBothNil
		}
		return nil
	}

	ttl := c.notFoundExpiry + time.Duration(c.safeRand.Int63n(int64(c.offset)))

	return c.remote.SetEX(ctx, key, notFoundPlaceholder, ttl)
}

func (c *jetCache[K, V]) Marshal(val any) ([]byte, error) {
	switch val := val.(type) {
	case nil:
		return nil, nil
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	}

	return encoding.GetCodec(c.codec).Marshal(val)
}

func (c *jetCache[K, V]) Unmarshal(b []byte, val any) error {
	if len(b) == 0 {
		return nil
	}

	switch val := val.(type) {
	case nil:
		return nil
	case *[]byte:
		clone := make([]byte, len(b))
		copy(clone, b)
		*val = clone
		return nil
	case *string:
		*val = string(b)
		return nil
	}

	return encoding.GetCodec(c.codec).Unmarshal(b, val)
}

func (c *jetCache[K, V]) Close() {
	c.stopRefresh()
	close(c.stopChan)
}

func (c *jetCache[K, V]) TaskSize() (size int) {
	c.refreshTaskMap.Range(func(key, val any) bool {
		size++
		return true
	})
	return
}

func (c *jetCache[K, V]) CacheType() string {
	if c.local != nil && c.remote != nil {
		return TypeBoth
	} else if c.remote != nil {
		return TypeRemote
	}
	return TypeLocal
}

func (c *jetCache[K, V]) addOrUpdateRefreshTask(item *item) {
	if c.refreshDuration <= 0 || !item.refresh {
		return
	}

	if ins, ok := c.refreshTaskMap.Load(item.key); ok {
		ins.(*refreshTask).lastAccessTime = time.Now()
	} else if ins, loaded := c.refreshTaskMap.LoadOrStore(item.key, item.toRefreshTask()); loaded {
		ins.(*refreshTask).lastAccessTime = time.Now()
	}
}

func (c *jetCache[K, V]) cancel(key any) {
	c.refreshTaskMap.Delete(key)
}

func (c *jetCache[K, V]) stopRefresh() {
	c.refreshTaskMap.Range(func(key, val any) bool {
		c.cancel(key)
		return true
	})
}

func (c *jetCache[K, V]) tick() {
	var (
		ticker = time.NewTicker(c.refreshDuration)
		sem    = semaphore.NewWeighted(int64(c.refreshConcurrency))
	)
	for {
		select {
		case <-ticker.C:
			c.Lock()
			// now is placed outside the Range to ensure that stopRefreshAfterLastAccess
			// does not time out under concurrent queuing.
			var now = time.Now()
			c.refreshTaskMap.Range(func(key, val any) bool {
				task := val.(*refreshTask)
				if c.stopRefreshAfterLastAccess > 0 {
					if task.lastAccessTime.Add(c.stopRefreshAfterLastAccess).Before(now) {
						logger.Debug("cancel refresh key: %s", key)
						c.cancel(key)
					} else {
						if err := sem.Acquire(context.Background(), 1); err != nil {
							logger.Error("tick#sem.Acquire error(%v)", err)
							return true
						}

						go util.WithRecover(func() {
							defer sem.Release(1)

							logger.Debug("start refresh key: %s", key)
							if c.remote != nil {
								c.externalLoad(context.Background(), task, now)
								return
							}
							c.load(context.Background(), task)
						})
					}
				}
				return true
			})
			c.Unlock()
		case <-c.stopChan:
			return
		}
	}
}

func (c *jetCache[K, V]) externalLoad(ctx context.Context, task *refreshTask, now time.Time) {
	var (
		lockKey    = fmt.Sprintf("%s%s", task.key, lockKeySuffix)
		shouldLoad bool
	)
	_, err := c.remote.Get(ctx, lockKey)
	if errors.Is(err, c.remote.Nil()) {
		shouldLoad = true
	} else if err != nil {
		logger.Error("externalLoad#c.remote.Get(%s) error(%v)", lockKey, err)
		return
	}

	if !shouldLoad {
		if c.local != nil {
			c.refreshLocal(ctx, task)
		}
		return
	}

	ok, err := c.remote.SetNX(ctx, lockKey, strconv.FormatInt(now.Unix(), 10), c.refreshDuration)
	if err != nil {
		logger.Error("externalLoad#c.remote.setNX(%s) error(%v)", lockKey, err)
		return
	}
	if ok {
		if err = c.Set(ctx, task.key, TTL(task.ttl), Do(task.do), SetXX(task.setXX),
			SetNX(task.setNX), SkipLocal(task.skipLocal)); err != nil {
			logger.Error("externalLoad#c.Set(%s) error(%v)", task.key, err)
			return
		}
	} else if c.local != nil {
		// If this goroutine fails to acquire the concurrent lock, it needs to wait briefly (delay) to trigger a refresh.
		// This way, it can directly fetch the origin result from Redis and refresh it locally.
		// The maximum concurrency here refers to the number of web machine instances, and the probability of
		// concurrent processing is actually not high. time.AfterFunc can be understood as a fallback mechanism to
		// reduce cache inconsistency time.
		time.AfterFunc(c.refreshDuration/5, func() {
			go util.WithRecover(func() {
				c.refreshLocal(context.Background(), task)
			})
		})
	}
}

func (c *jetCache[K, V]) load(ctx context.Context, task *refreshTask) {
	if err := c.Set(ctx, task.key, TTL(task.ttl), Do(task.do), SetXX(task.setXX),
		SetNX(task.setNX), SkipLocal(task.skipLocal)); err != nil {
		logger.Error("load#c.Set(%s) error(%v)", task.key, err)
	}
}

func (c *jetCache[K, V]) refreshLocal(ctx context.Context, task *refreshTask) {
	val, err := c.remote.Get(ctx, task.key)
	if err != nil {
		logger.Error("refreshLocal#c.remote.Get(%s) error(%v)", task.key, err)
		return
	}
	c.local.Set(task.key, util.Bytes(val))
}
