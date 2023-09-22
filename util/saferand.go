package util

import (
	"math/rand"
	"sync"
	"time"
)

type SafeRand struct {
	mu   *sync.Mutex
	rand *rand.Rand
}

func NewSafeRand() *SafeRand {
	return &SafeRand{
		mu:   new(sync.Mutex),
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *SafeRand) Int63n(n int64) int64 {
	r.mu.Lock()
	val := r.rand.Int63n(n)
	r.mu.Unlock()
	return val
}
