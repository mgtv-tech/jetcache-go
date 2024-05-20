package util

import (
	"fmt"
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

func (r *SafeRand) RandN(n int) string {
	r.mu.Lock()
	randBytes := make([]byte, n/2)
	r.rand.Read(randBytes)
	val := fmt.Sprintf("%x", randBytes)
	r.mu.Unlock()
	return val
}
