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
		assert.Nil(t, o.Value)
		assert.Nil(t, o.Do)
		assert.Equal(t, defaultTTL, o.ttl())
		assert.False(t, o.SetXX)
		assert.False(t, o.SetNX)
		assert.False(t, o.SkipLocal)
		assert.False(t, o.Refresh)
	})

	t.Run("nil context", func(t *testing.T) {
		o := newItemOptions(nil, "key")
		assert.Equal(t, context.Background(), o.Context())
	})

	t.Run("with item options", func(t *testing.T) {
		o := newItemOptions(context.TODO(), "key", Value("value"),
			TTL(time.Minute), SetXX(true), SetNX(true), SkipLocal(true),
			Refresh(true), Do(func() (interface{}, error) {
				return "any", nil
			}))
		assert.Equal(t, "value", o.Value)
		assert.NotNil(t, o.Do)
		assert.Equal(t, time.Minute, o.ttl())
		assert.True(t, o.SetXX)
		assert.True(t, o.SetNX)
		assert.True(t, o.SkipLocal)
		assert.True(t, o.Refresh)
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
			expect: defaultTTL,
		},
		{
			input:  time.Minute,
			expect: time.Minute,
		},
	}

	for _, v := range tests {
		item := &item{TTL: v.input}
		assert.Equal(t, v.expect, item.ttl())
	}
}
