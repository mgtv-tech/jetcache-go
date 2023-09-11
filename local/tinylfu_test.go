package local

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTinyLFU(t *testing.T) {
	cache := NewTinyLFU(1000, time.Second)
	assert.Equal(t, time.Second/10, cache.offset)
	cache.UseRandomizedTTL(time.Millisecond)
	assert.Equal(t, time.Millisecond, cache.offset)

	key1 := "key1"
	val, exists := cache.Get(key1)
	assert.False(t, exists)
	assert.Equal(t, []byte(nil), val)

	cache.Set(key1, []byte("value1"))
	val, exists = cache.Get(key1)
	assert.True(t, exists)
	assert.Equal(t, []byte("value1"), val)

	cache.Del(key1)
	val, exists = cache.Get(key1)
	assert.False(t, exists)
	assert.Equal(t, []byte(nil), val)
}

func TestTinyLFUGetCorruptionOnExpiry(t *testing.T) {
	strFor := func(i int) string {
		return fmt.Sprintf("a string %d", i)
	}
	keyName := func(i int) string {
		return fmt.Sprintf("key-%00000d", i)
	}

	cache := NewTinyLFU(1000, 1*time.Second)
	size := 50000
	// Put a bunch of stuff in the cache with a TTL of 1 second
	for i := 0; i < size; i++ {
		key := keyName(i)
		cache.Set(key, []byte(strFor(i)))
	}

	// Read stuff for a bit longer than the TTL - that's when the corruption occurs
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := ctx.Done()
loop:
	for {
		select {
		case <-done:
			// this is expected
			break loop
		default:
			i := rand.Intn(size)
			key := keyName(i)

			b, ok := cache.Get(key)
			if !ok {
				continue loop
			}

			got := string(b)
			expected := strFor(i)
			if got != expected {
				t.Fatalf("expected=%q got=%q key=%q", expected, got, key)
			}
		}
	}
}
