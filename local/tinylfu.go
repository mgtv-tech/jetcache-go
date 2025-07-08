package local

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/mgtv-tech/jetcache-go/util"
)

const (
	numCounters = 1e7 // number of keys to track frequency of (10M).
	bufferItems = 64  // number of keys per Get buffer.
)

var _ Local = (*TinyLFU)(nil)

type TinyLFU struct {
	rand   *util.SafeRand
	cache  *ristretto.Cache[string, []byte]
	ttl    time.Duration
	offset time.Duration
}

func NewTinyLFU(size int, ttl time.Duration) *TinyLFU {
	const maxOffset = 10 * time.Second

	offset := ttl / 10
	if offset > maxOffset {
		offset = maxOffset
	}

	cache, err := ristretto.NewCache[string, []byte](&ristretto.Config[string, []byte]{
		NumCounters: numCounters,
		MaxCost:     int64(size),
		BufferItems: bufferItems,
	})
	if err != nil {
		panic(err)
	}

	return &TinyLFU{
		rand:   util.NewSafeRand(),
		cache:  cache,
		ttl:    ttl,
		offset: offset,
	}
}

func (c *TinyLFU) UseRandomizedTTL(offset time.Duration) {
	c.offset = offset
}

func (c *TinyLFU) Set(key string, b []byte) {
	ttl := c.ttl
	if c.offset > 0 {
		ttl += time.Duration(c.rand.Int63n(int64(c.offset)))
	}

	c.cache.SetWithTTL(key, b, 1, ttl)

	// wait for value to pass through buffers
	c.cache.Wait()
}

func (c *TinyLFU) Get(key string) ([]byte, bool) {
	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}

	return val, true
}

func (c *TinyLFU) Del(key string) {
	c.cache.Del(key)
}
