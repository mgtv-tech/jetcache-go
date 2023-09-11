package local

import (
	"math/rand"
	"sync"
	"time"

	"github.com/coocood/freecache"

	"github.com/jetcache-go/logger"
	"github.com/jetcache-go/util"
)

var _ Local = (*FreeCache)(nil)

var (
	innerCache *freecache.Cache
	once       sync.Once
)

type (
	FreeCache struct {
		rand   *rand.Rand
		ttl    time.Duration
		offset time.Duration
	}
)

// NewFreeCache Create a new cache instance, but the internal cache instances are shared,
// and they will only be initialized once.
func NewFreeCache(size Size, ttl time.Duration) *FreeCache {
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
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		ttl:    ttl,
		offset: offset,
	}
}

func (c *FreeCache) UseRandomizedTTL(offset time.Duration) {
	c.offset = offset
}

func (c *FreeCache) Set(key string, b []byte) {
	ttl := c.ttl
	if c.offset > 0 {
		ttl += time.Duration(c.rand.Int63n(int64(c.offset)))
	}

	if err := innerCache.Set(util.Bytes(key), b, int(ttl.Seconds())); err != nil {
		logger.Error("freeCache set(%s) error(%v)", key, err)
	}
}

func (c *FreeCache) Get(key string) ([]byte, bool) {
	b, err := innerCache.Get(util.Bytes(key))
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
	innerCache.Del(util.Bytes(key))
}
