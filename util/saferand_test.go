package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandSafe_Int63n(t *testing.T) {
	rand := NewSafeRand()
	for i := 0; i < 1000; i++ {
		val := rand.Int63n(1000)
		assert.True(t, val >= 0)
		assert.True(t, val < 1000)
	}
}

func BenchmarkInt63ThreadSafe(b *testing.B) {
	rand := NewSafeRand()
	for n := b.N; n > 0; n-- {
		rand.Int63n(1000)
	}
}

func BenchmarkInt63ThreadSafeParallel(b *testing.B) {
	rand := NewSafeRand()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rand.Int63n(1000)
		}
	})
}
