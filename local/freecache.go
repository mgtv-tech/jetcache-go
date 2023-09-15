package local

import (
	"math/rand"
	"sync"
	"time"

	"github.com/coocood/freecache"

	"github.com/daoshenzzg/jetcache-go/logger"
	"github.com/daoshenzzg/jetcache-go/util"
)

var _ Local = (*FreeCache)(nil)

var (
	innerCache *freecache.Cache
	once       sync.Once
)

type (
	FreeCache struct {
		mu             sync.Mutex
		rand           *rand.Rand
		ttl            time.Duration
		offset         time.Duration
		innerKeyPrefix string
	}
	// Option defines the method to customize an Options.
	Option func(o *FreeCache)
)

// NewFreeCache Create a new cache instance, but the internal cache instances are shared,
// and they will only be initialized once.
func NewFreeCache(size Size, ttl time.Duration, innerKeyPrefix ...string) *FreeCache {
	prefix := ""
	if len(innerKeyPrefix) > 0 {
		prefix = innerKeyPrefix[0]
	}

	// avoid "expireSeconds <= 0 means no expire"
	if ttl > 0 && ttl < time.Second {
		ttl = time.Second
	}

	const maxOffset = 10 * time.Second
	offset := ttl / 10
	if offset > maxOffset {
		offset = maxOffset
	}

	once.Do(func() {
		if size < 512*KB || size > 8*GB {
			size = 256 * MB
		}
		innerCache = freecache.NewCache(int(size))
	})

	return &FreeCache{
		innerKeyPrefix: prefix,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
		ttl:            ttl,
		offset:         offset,
	}
}

func (c *FreeCache) UseRandomizedTTL(offset time.Duration) {
	c.offset = offset
}

func (c *FreeCache) Set(key string, b []byte) {
	ttl := c.ttl
	if c.offset > 0 {
		c.mu.Lock()
		ttl += time.Duration(c.rand.Int63n(int64(c.offset)))
		c.mu.Unlock()
	}

	if err := innerCache.Set(util.Bytes(c.Key(key)), b, int(ttl.Seconds())); err != nil {
		logger.Error("freeCache set(%s) error(%v)", key, err)
	}
}

func (c *FreeCache) Get(key string) ([]byte, bool) {
	b, err := innerCache.Get(util.Bytes(c.Key(key)))
	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, false
		}
		logger.Error("freeCache get(%s) error(%v)", key, err)
		return nil, false
	}

	return b, true
}

func (c *FreeCache) Del(key string) {
	innerCache.Del(util.Bytes(c.Key(key)))
}

func (c *FreeCache) Key(key string) string {
	if c.innerKeyPrefix == "" {
		return key
	}

	return util.JoinAny(":", c.innerKeyPrefix, key)
}
