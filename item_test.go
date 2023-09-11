package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
			expect: time.Hour,
		},
		{
			input:  time.Minute,
			expect: time.Minute,
		},
	}

	for _, v := range tests {
		item := &Item{TTL: v.input}
		assert.Equal(t, v.expect, item.ttl())
	}
}
