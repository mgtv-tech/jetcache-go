package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestItemOptions(t *testing.T) {
	t.Run("default item options", func(t *testing.T) {
		o := newItemOptions(context.TODO(), "key")
		assert.Nil(t, o.value)
		assert.Nil(t, o.do)
		assert.Equal(t, defaultRemoteExpiry, o.getTtl(defaultRemoteExpiry))
		assert.False(t, o.setXX)
		assert.False(t, o.setNX)
		assert.False(t, o.skipLocal)
		assert.False(t, o.refresh)
	})

	t.Run("nil context", func(t *testing.T) {
		o := newItemOptions(nil, "key")
		assert.Equal(t, context.Background(), o.Context())
	})

	t.Run("with item options", func(t *testing.T) {
		o := newItemOptions(context.TODO(), "key", Value("getValue"),
			TTL(time.Minute), SetXX(true), SetNX(true), SkipLocal(true),
			Refresh(true), Do(func(context.Context) (any, error) {
				return "any", nil
			}))
		assert.Equal(t, "getValue", o.value)
		assert.NotNil(t, o.do)
		assert.Equal(t, time.Minute, o.getTtl(defaultRemoteExpiry))
		assert.True(t, o.setXX)
		assert.True(t, o.setNX)
		assert.True(t, o.skipLocal)
		assert.True(t, o.refresh)
	})
}

func TestItemTTL(t *testing.T) {
	tests := []struct {
		input  time.Duration
		expect time.Duration
	}{
		{
			input:  -1,
			expect: 0,
		},
		{
			input:  time.Millisecond,
			expect: defaultRemoteExpiry,
		},
		{
			input:  time.Minute,
			expect: time.Minute,
		},
	}

	for _, v := range tests {
		item := &item{ttl: v.input}
		assert.Equal(t, v.expect, item.getTtl(defaultRemoteExpiry))
	}
}
