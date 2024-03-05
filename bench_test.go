package cache

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/daoshenzzg/jetcache-go/local"
	"github.com/daoshenzzg/jetcache-go/logger"
	"github.com/daoshenzzg/jetcache-go/remote"
)

var (
	tOnce sync.Once
	rdb   *redis.Client
)

func tInit() {
	tOnce.Do(func() {
		rdb = newRdb()
	})
}

func BenchmarkOnceWithTinyLFU(b *testing.B) {
	tInit()

	cache := newBoth(rdb, tinyLFU)
	obj := &object{
		Str: strings.Repeat("my very large string", 10),
		Num: 42,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dst object
			err := cache.Once(context.TODO(), "bench-once", Value(&dst), Do(func(context.Context) (any, error) {
				return obj, nil
			}))
			if err != nil {
				b.Fatal(err)
			}
			if dst.Num != 42 {
				b.Fatalf("%d != 42", dst.Num)
			}
		}
	})
}

func BenchmarkSetWithTinyLFU(b *testing.B) {
	tInit()

	cache := newBoth(rdb, tinyLFU)
	obj := &object{
		Str: strings.Repeat("my very large string", 10),
		Num: 42,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := cache.Set(context.TODO(), "bench-set", Value(obj)); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkOnceWithFreeCache(b *testing.B) {
	tInit()

	cache := newBoth(rdb, freeCache)
	obj := &object{
		Str: strings.Repeat("my very large string", 10),
		Num: 42,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dst object
			err := cache.Once(context.TODO(), "bench-once", Value(&dst),
				Do(func(context.Context) (any, error) {
					return obj, nil
				}))
			if err != nil {
				b.Fatal(err)
			}
			if dst.Num != 42 {
				b.Fatalf("%d != 42", dst.Num)
			}
		}
	})
}

func BenchmarkSetWithFreeCache(b *testing.B) {
	tInit()

	cache := newBoth(rdb, freeCache)
	obj := &object{
		Str: strings.Repeat("my very large string", 10),
		Num: 42,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := cache.Set(context.TODO(), "bench-set", Value(obj)); err != nil {
				b.Fatal(err)
			}
		}
	})
}

var (
	asyncCache Cache[int, *object]
	newOnce    sync.Once
)

func BenchmarkOnceWithStats(b *testing.B) {
	cache := newRefreshBoth()
	obj := &object{
		Str: strings.Repeat("my very large string", 10),
		Num: 42,
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dst object
			err := cache.Once(context.TODO(), "bench-once_"+strconv.Itoa(rand.Intn(256)),
				Value(&dst), Do(func(context.Context) (any, error) {
					time.Sleep(50 * time.Millisecond)
					return obj, nil
				}))
			if err != nil {
				b.Fatal(err)
			}
			if dst.Num != 42 {
				b.Fatalf("%d != 42", dst.Num)
			}
		}
	})
}

func BenchmarkOnceRefreshWithStats(b *testing.B) {
	logger.SetLevel(logger.LevelInfo)
	cache := newRefreshBoth()
	obj := &object{
		Str: strings.Repeat("my very large string", 10),
		Num: 42,
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dst object
			err := cache.Once(context.TODO(), "bench-refresh_"+strconv.Itoa(rand.Intn(256)),
				Value(&dst), Do(func(context.Context) (any, error) {
					time.Sleep(50 * time.Millisecond)
					return obj, nil
				}),
				Refresh(true))
			if err != nil {
				b.Fatal(err)
			}
			if dst.Num != 42 {
				b.Fatalf("%d != 42", dst.Num)
			}
		}
	})
}

func BenchmarkMGetWithStats(b *testing.B) {
	logger.SetLevel(logger.LevelInfo)
	cache := newRefreshBoth()
	ids := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.MGet(context.TODO(), "key", ids, func(ctx context.Context, ids []int) (map[int]*object, error) {
				return mockDBMGetObject(ids)
			})
		}
	})
}

func mockDBMGetObject(ids []int) (map[int]*object, error) {
	ret := make(map[int]*object)
	for _, id := range ids {
		if id == 3 {
			continue
		}
		ret[id] = &object{Str: "mystring", Num: id}
	}
	return ret, nil
}

func newRefreshBoth() Cache[int, *object] {
	tInit()

	newOnce.Do(func() {
		name := "bench"
		asyncCache = New[int, *object](WithName(name),
			WithRemote(remote.NewGoRedisV8Adaptor(rdb)),
			WithLocal(local.NewFreeCache(256*local.MB, 3*time.Second)),
			WithErrNotFound(errTestNotFound),
			WithRefreshDuration(2*time.Second),
			WithStopRefreshAfterLastAccess(3*time.Second),
			WithRefreshConcurrency(1000))
	})
	return asyncCache
}
