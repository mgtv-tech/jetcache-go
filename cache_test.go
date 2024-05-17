package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mgtv-tech/jetcache-go/encoding"
	"github.com/mgtv-tech/jetcache-go/local"
	"github.com/mgtv-tech/jetcache-go/logger"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/mgtv-tech/jetcache-go/util"
)

var (
	localId         int32
	errTestNotFound = errors.New("not found")
	localTypes      = []localType{tinyLFU, freeCache}
)

const (
	freeCache localType = 1
	tinyLFU   localType = 2

	localExpire                = time.Minute
	refreshDuration            = time.Second
	stopRefreshAfterLastAccess = 3 * refreshDuration
	testEventChSize            = 10
)

type (
	localType int
	object    struct {
		Str string
		Num int
	}
)

func TestGinkgo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cache")
}

func perform(n int, cbs ...func(int)) {
	var wg sync.WaitGroup
	for _, cb := range cbs {
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(cb func(int), i int) {
				defer wg.Done()
				defer GinkgoRecover()

				cb(i)
			}(cb, i)
		}
	}
	wg.Wait()
}

var _ = Describe("Cache", func() {
	ctx := context.TODO()

	const key = "mykey"
	var (
		obj    *object
		rdb    *redis.Client
		cache  Cache
		cacheT *T[int, *object]
	)

	testCache := func() {
		It("Remote and Local both nil", func() {
			nilCache := New().(*jetCache)

			err := nilCache.Get(ctx, "key", nil)
			Expect(err).To(Equal(ErrRemoteLocalBothNil))

			err = nilCache.Delete(ctx, "key")
			Expect(err).To(Equal(ErrRemoteLocalBothNil))

			err = nilCache.Set(ctx, "key", Do(func(context.Context) (any, error) {
				return "getValue", nil
			}))
			Expect(err).To(Equal(ErrRemoteLocalBothNil))

			err = nilCache.setNotFound(ctx, "key", false)
			Expect(err).To(Equal(ErrRemoteLocalBothNil))
		})

		It("Gets and Sets nil", func() {
			err := cache.Set(ctx, key, TTL(time.Hour))
			Expect(err).NotTo(HaveOccurred())

			err = cache.Get(ctx, key, nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(cache.Exists(ctx, key)).To(BeTrue())
		})

		It("Deletes key", func() {
			err := cache.Set(ctx, key, TTL(time.Hour))
			Expect(err).NotTo(HaveOccurred())

			Expect(cache.Exists(ctx, key)).To(BeTrue())

			if cache.CacheType() == TypeLocal {
				cache.DeleteFromLocalCache(key)
				Expect(cache.Exists(ctx, key)).To(BeFalse())
			}

			if cache.CacheType() == TypeRemote {
				cache.DeleteFromLocalCache(key)
				Expect(cache.Exists(ctx, key)).To(BeTrue())
			}

			err = cache.Delete(ctx, key)
			Expect(err).NotTo(HaveOccurred())

			err = cache.Get(ctx, key, nil)
			Expect(err).To(Equal(ErrCacheMiss))

			Expect(cache.Exists(ctx, key)).To(BeFalse())
		})

		It("SetXxNx", func() {
			if cache.CacheType() == TypeRemote {
				err := cache.Set(ctx, key, TTL(time.Hour), Value(obj), SetXX(true))
				Expect(err).NotTo(HaveOccurred())
				err = cache.Get(ctx, key, nil)
				Expect(err).To(Equal(ErrCacheMiss))

				err = cache.Set(ctx, key, TTL(time.Hour), Value(obj), SetNX(true))
				Expect(err).NotTo(HaveOccurred())
				Expect(cache.Exists(ctx, key)).To(BeTrue())
			}
		})

		It("Gets and Sets data", func() {
			err := cache.Set(ctx, key, Value(obj), TTL(time.Hour))
			Expect(err).NotTo(HaveOccurred())

			wanted := new(object)
			err = cache.Get(ctx, key, wanted)
			Expect(err).NotTo(HaveOccurred())
			Expect(wanted).To(Equal(obj))

			Expect(cache.Exists(ctx, key)).To(BeTrue())

			if cache.CacheType() == TypeRemote || cache.CacheType() == TypeBoth {
				err = cache.GetSkippingLocal(ctx, key, wanted)
				Expect(err).NotTo(HaveOccurred())
				Expect(wanted).To(Equal(obj))
			}
		})

		It("Sets string as is", func() {
			value := "str_value"

			err := cache.Set(ctx, key, Value(value))
			Expect(err).NotTo(HaveOccurred())

			var dst string
			err = cache.Get(ctx, key, &dst)
			Expect(err).NotTo(HaveOccurred())
			Expect(dst).To(Equal(value))
		})

		It("Sets bytes as is", func() {
			value := []byte("str_value")

			err := cache.Set(ctx, key, Value(value))
			Expect(err).NotTo(HaveOccurred())

			var dst []byte
			err = cache.Get(ctx, key, &dst)
			Expect(err).NotTo(HaveOccurred())
			Expect(dst).To(Equal(value))
		})

		It("can be used with Incr", func() {
			if rdb == nil {
				return
			}

			value := "123"

			err := cache.Set(ctx, key, Value(value))
			Expect(err).NotTo(HaveOccurred())

			n, err := rdb.Incr(ctx, key).Result()
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(int64(124)))
		})

		Describe("MGet func", func() {
			It("cache hit with set first", func() {
				err := cache.Set(context.Background(), "key:1", Value(&object{Str: "str1", Num: 1}))
				Expect(err).NotTo(HaveOccurred())
				Expect(cache.Exists(ctx, "key:1")).To(BeTrue())

				err = cache.Set(context.Background(), "key:2", Value(&object{Str: "str2", Num: 2}))
				Expect(err).NotTo(HaveOccurred())
				Expect(cache.Exists(ctx, "key:2")).To(BeTrue())

				ids := []int{1, 2}
				ret := cacheT.MGet(context.Background(), "key", ids, nil)
				Expect(ret).To(Equal(map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}))
			})

			It("cache hit with fn", func() {
				ids := []int{1, 2, 3}
				ret := cacheT.MGet(context.Background(), "key", ids,
					func(ctx context.Context, ints []int) (map[int]*object, error) {
						return map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}, nil
					})
				Expect(ret).To(Equal(map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}))
			})

			It("cache hit with fn load error", func() {
				err := cache.Set(context.Background(), "key:1", Value(&object{Str: "str1", Num: 1}))
				Expect(err).NotTo(HaveOccurred())
				Expect(cache.Exists(ctx, "key:1")).To(BeTrue())

				ids := []int{1, 2}
				ret := cacheT.MGet(context.Background(), "key", ids,
					func(ctx context.Context, ints []int) (map[int]*object, error) {
						return nil, errors.New("any")
					})
				Expect(ret).To(Equal(map[int]*object{1: {Str: "str1", Num: 1}}))
			})

			It("cache miss", func() {
				ids := []int{1, 2}
				ret := cacheT.MGet(context.Background(), "key", ids,
					func(ctx context.Context, ints []int) (map[int]*object, error) {
						return nil, nil
					})
				Expect(ret).To(BeEmpty())
			})

			It("with skip elements that unmarshal error", func() {
				if cache.CacheType() == TypeRemote {
					codecErrCache := New(WithName("codecErr"),
						WithRemote(remote.NewGoRedisV8Adaptor(rdb)),
						WithCodec(mockUnmarshalErr))

					err := codecErrCache.Set(context.Background(), "key:1", Value("value1"))
					Expect(err).NotTo(HaveOccurred())

					ids := []int{1, 2}
					ret := cacheT.MGet(context.Background(), "key", ids, nil)
					Expect(ret).To(Equal(map[int]*object{}))
				}

				if cache.CacheType() == TypeLocal {
					local := localNew(freeCache)
					codecErrCache := New(WithName("codecErr"),
						WithLocal(local),
						WithCodec(mockUnmarshalErr))
					cacheT := NewT[int, *object](codecErrCache)

					local.Set("key:1", []byte("value1"))

					ids := []int{1, 2}
					ret := cacheT.MGet(context.Background(), "key", ids, nil)
					Expect(ret).To(Equal(map[int]*object{}))
				}
			})

			It("with skip elements that marshal error", func() {
				if cache.CacheType() == TypeRemote {
					codecErrCache := New(WithName("codecErr"),
						WithRemote(remote.NewGoRedisV8Adaptor(rdb)),
						WithCodec(mockMarshalErr))
					cacheT := NewT[int, *object](codecErrCache)
					ids := []int{1, 2}
					// 1st marshal error, but return origin load func data
					ret := cacheT.MGet(context.Background(), "key", ids,
						func(ctx context.Context, ids []int) (map[int]*object, error) {
							return map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}, nil
						})
					Expect(ret).To(Equal(map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}))
					// 2nd cache hit placeholder "*", then return miss
					ret = cacheT.MGet(context.Background(), "key", ids, nil)
					Expect(ret).To(Equal(map[int]*object{}))
				}

				if cache.CacheType() == TypeLocal {
					local := localNew(freeCache)
					codecErrCache := New(WithName("codecErr"),
						WithLocal(local),
						WithCodec(mockMarshalErr))
					cacheT := NewT[int, *object](codecErrCache)
					ids := []int{1, 2}
					// 1st marshal error, but return origin load func data
					ret := cacheT.MGet(context.Background(), "key", ids,
						func(ctx context.Context, ids []int) (map[int]*object, error) {
							return map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}, nil
						})
					Expect(ret).To(Equal(map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}))
					// 2nd cache hit placeholder "*", then return miss
					ret = cacheT.MGet(context.Background(), "key", ids, nil)
					Expect(ret).To(Equal(map[int]*object{}))
				}
			})

			It("with will skip elements that remote MSet error", func() {
				if cache.CacheType() == TypeRemote {
					codecErrCache := New(WithName("codecErr"),
						WithRemote(&mockGoRedisMGetMSetErrAdapter{}))
					cacheT := NewT[int, *object](codecErrCache)
					ids := []int{1, 2, 3}
					// 1st marshal error, but return origin load func data
					ret := cacheT.MGet(context.Background(), "key", ids,
						func(ctx context.Context, ids []int) (map[int]*object, error) {
							return map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}, nil
						})
					Expect(ret).To(Equal(map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}))
					// 2nd cache hit placeholder "*", then return miss
					ret = cacheT.MGet(context.Background(), "key", ids, nil)
					Expect(ret).To(Equal(map[int]*object{}))
				}
			})
		})

		Describe("Once func", func() {
			It("works with err not found", func() {
				key := "cache-err-not-found"
				do := func(context.Context) (any, error) {
					return nil, errTestNotFound
				}
				var value string
				err := cache.Once(ctx, key, Value(&value), Do(do))
				Expect(err).To(Equal(errTestNotFound))
				Expect(cache.Get(context.Background(), key, &value)).To(Equal(errTestNotFound))
				Expect(cache.Exists(context.Background(), key)).To(BeFalse())
				if cache.CacheType() == TypeRemote || cache.CacheType() == TypeBoth {
					val, err := rdb.Get(context.Background(), key).Result()
					Expect(err).To(BeNil())
					Expect(val).To(Equal(string(notFoundPlaceholder)))
				}

				_ = cache.Set(ctx, key, Value(value), Do(do))
				do = func(context.Context) (any, error) {
					return nil, nil
				}
				err = cache.Once(ctx, key, Value(&value), Do(do))
				Expect(err).To(Equal(errTestNotFound))
				Expect(cache.Get(context.Background(), key, &value)).To(Equal(errTestNotFound))
				Expect(cache.Exists(context.Background(), key)).To(BeFalse())

				_ = cache.Delete(context.Background(), key)
				errAny := errors.New("any")
				do = func(context.Context) (any, error) {
					return nil, errAny
				}
				err = cache.Once(ctx, key, Value(&value), Do(do))
				Expect(err).To(Equal(errAny))
			})

			It("works without value and error result", func() {
				var callCount int64
				perform(100, func(int) {
					err := cache.Once(ctx, key, Do(func(context.Context) (any, error) {
						time.Sleep(100 * time.Millisecond)
						atomic.AddInt64(&callCount, 1)
						return nil, errors.New("error stub")
					}))
					Expect(err).To(MatchError("error stub"))
				})
				Expect(callCount).To(Equal(int64(1)))
			})

			It("does not cache error result", func() {
				var callCount int64
				do := func(sleep time.Duration) (int, error) {
					var n int
					err := cache.Once(ctx, key, Value(&n), Do(func(context.Context) (any, error) {
						time.Sleep(sleep)

						n := atomic.AddInt64(&callCount, 1)
						if n == 1 {
							return nil, errors.New("error stub")
						}
						return 42, nil
					}))
					if err != nil {
						return 0, err
					}
					return n, nil
				}

				perform(100, func(int) {
					n, err := do(100 * time.Millisecond)
					Expect(err).To(MatchError("error stub"))
					Expect(n).To(Equal(0))
				})

				perform(100, func(int) {
					n, err := do(0)
					Expect(err).NotTo(HaveOccurred())
					Expect(n).To(Equal(42))
				})

				Expect(callCount).To(Equal(int64(2)))
			})

			It("skips Set when getTtl = -1", func() {
				key := "skip-set"

				var value string
				err := cache.Once(ctx, key, Value(&value), TTL(-1), Do(func(context.Context) (any, error) {
					return "hello", nil
				}))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("hello"))

				if rdb != nil {
					exists, err := rdb.Exists(ctx, key).Result()
					Expect(err).NotTo(HaveOccurred())
					Expect(exists).To(Equal(int64(0)))
				}
			})
		})

		Describe("Once func with refresh", func() {
			It("refresh ok", func() {
				var (
					key       = util.JoinAny(":", cache.CacheType(), "K1")
					callCount int64
					value     string
					err       error
				)
				err = cache.Once(ctx, key, Value(&value), TTL(time.Minute), Refresh(true),
					Do(func(context.Context) (any, error) {
						if atomic.AddInt64(&callCount, 1) == 1 {
							return "V1", nil
						}
						return "V2", nil
					}))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				time.Sleep(refreshDuration / 2)
				err = cache.Get(ctx, key, &value)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				time.Sleep(refreshDuration)
				err = cache.Get(ctx, key, &value)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V2"))

				time.Sleep(2 * refreshDuration)
			})

			It("refresh err", func() {
				var (
					key       = util.JoinAny(":", cache.CacheType(), "K1")
					callCount int64
					value     string
					err       error
				)
				err = cache.Once(ctx, key, Value(&value), TTL(time.Minute), Refresh(true),
					Do(func(context.Context) (any, error) {
						if atomic.AddInt64(&callCount, 1) == 1 {
							return "", errors.New("any")
						}
						return "V1", nil
					}))
				Expect(err).To(Equal(errors.New("any")))
				Expect(value).To(BeEmpty())

				time.Sleep(refreshDuration / 2)
				err2 := cache.Get(ctx, key, &value)
				Expect(err2).To(Equal(err2))

				time.Sleep(refreshDuration)
				err = cache.Get(ctx, key, &value)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))
			})

			It("work with refreshLocal", func() {
				if cache.CacheType() != TypeBoth {
					return
				}
				var (
					jetCache = cache.(*jetCache)
					key      = util.JoinAny(":", cache.CacheType(), "K1")
					value    string
					err      error
				)
				err = jetCache.Once(ctx, key, Value(&value), TTL(time.Minute), Refresh(true),
					Do(func(context.Context) (any, error) {
						return "V1", nil
					}))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				_, err = rdb.SetEX(ctx, key, "V2", time.Minute).Result()
				Expect(err).NotTo(HaveOccurred())
				jetCache.refreshLocal(ctx, &refreshTask{key: key})

				err = cache.Get(ctx, key, &value)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V2"))
			})

			It("work with externalLoad", func() {
				if cache.CacheType() != TypeBoth {
					return
				}
				var (
					callCount int64
					jetCache  = cache.(*jetCache)
					key       = util.JoinAny(":", cache.CacheType(), "K1")
					doFunc    = func(context.Context) (any, error) {
						if atomic.AddInt64(&callCount, 1) == 1 {
							return "V1", nil
						}
						return "V2", nil
					}
					value string
					err   error
				)
				err = jetCache.Once(ctx, key, Value(&value), TTL(time.Minute), Refresh(true), Do(doFunc))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				// shouldLoad SetNX true
				jetCache.externalLoad(ctx, &refreshTask{key: key, do: doFunc, ttl: time.Minute}, time.Now())
				err = cache.Get(ctx, key, &value)
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V2"))

				// shouldLoad SetNX false, must refreshLocal
				_, err = rdb.SetEX(ctx, key, "V3", time.Minute).Result()
				Expect(err).NotTo(HaveOccurred())
				jetCache.externalLoad(ctx, &refreshTask{key: key, do: doFunc, ttl: time.Minute}, time.Now())
				b, ok := jetCache.local.Get(key)
				Expect(ok).To(BeTrue())
				Expect(string(b)).To(Equal("V3"))
			})

			It("work with concurrency externalLoad", func() {
				if cache.CacheType() != TypeBoth {
					return
				}

				var (
					jetCache = cache.(*jetCache)
					key      = util.JoinAny(":", cache.CacheType(), "K1")
					lockKey  = fmt.Sprintf("%s%s", key, lockKeySuffix)
					doFunc   = func(context.Context) (any, error) {
						return "V1", nil
					}
					value string
					err   error
				)
				err = jetCache.Once(ctx, key, Value(&value), TTL(time.Minute), Refresh(true), Do(doFunc))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				perform(200, func(i int) {
					rdb.Del(context.TODO(), lockKey)
					jetCache.externalLoad(ctx, &refreshTask{key: key, do: doFunc, ttl: time.Minute}, time.Now())
				})
				b, ok := jetCache.local.Get(key)
				Expect(ok).To(BeTrue())
				Expect(string(b)).To(Equal("V1"))

				_, err = rdb.SetEX(ctx, key, "V2", time.Minute).Result()
				b, ok = jetCache.local.Get(key)
				Expect(ok).To(BeTrue())
				Expect(string(b)).To(Equal("V1"))

				// time.AfterFunc(c.refreshDuration/5, refreshLocal())
				time.Sleep(refreshDuration/5 + 100*time.Millisecond)
				b, ok = jetCache.local.Get(key)
				Expect(ok).To(BeTrue())
				Expect(string(b)).To(Equal("V2"))
			})

			It("test addOrUpdateRefreshTask", func() {
				var jetCache = cache.(*jetCache)
				Expect(jetCache.TaskSize()).To(Equal(0))

				key := util.JoinAny(":", cache.CacheType(), "K1")
				now := time.Now()
				item := &item{
					key: key,
					ttl: time.Minute,
					do: func(context.Context) (any, error) {
						return "V1", nil
					},
					refresh: true,
				}
				jetCache.addOrUpdateRefreshTask(item)
				Expect(jetCache.TaskSize()).To(Equal(1))
				ins, ok := jetCache.refreshTaskMap.Load(key)
				Expect(ok).To(BeTrue())
				task, ok := ins.(*refreshTask)
				lastAccessTime := task.lastAccessTime
				Expect(ok).To(BeTrue())
				Expect(lastAccessTime.After(now)).To(BeTrue())

				jetCache.addOrUpdateRefreshTask(item)
				Expect(jetCache.TaskSize()).To(Equal(1))
				ins, ok = jetCache.refreshTaskMap.Load(key)
				Expect(ok).To(BeTrue())
				task, ok = ins.(*refreshTask)
				Expect(ok).To(BeTrue())
				Expect(task.lastAccessTime.After(lastAccessTime)).To(BeTrue())

				jetCache.cancel(key)
				Expect(jetCache.TaskSize()).To(Equal(0))
			})
		})

		Describe("Sync Local", func() {
			It("Set with sync local", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				err := jetCache.Set(ctx, key, Value(obj), TTL(time.Hour))
				Expect(err).NotTo(HaveOccurred())

				e, ok := <-jetCache.eventCh
				Expect(ok).To(BeTrue())
				Expect(e.Keys[0]).To(Equal(key))
				Expect(e.EventType).To(Equal(EventTypeSet))
				Expect(e.CacheName).To(Equal(jetCache.name))
				Expect(e.SourceID).NotTo(BeEmpty())
			})

			It("Delete with sync local", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				err := jetCache.Delete(ctx, key)
				Expect(err).NotTo(HaveOccurred())

				e, ok := <-jetCache.eventCh
				Expect(ok).To(BeTrue())
				Expect(e.Keys[0]).To(Equal(key))
				Expect(e.EventType).To(Equal(EventTypeDelete))
			})

			It("MGet with sync local", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				ids := []int{1, 2, 3}
				_ = cacheT.MGet(context.Background(), "key", ids,
					func(ctx context.Context, ints []int) (map[int]*object, error) {
						return map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}, nil
					})

				e, ok := <-jetCache.eventCh
				Expect(ok).To(BeTrue())
				Expect(len(e.Keys)).To(Equal(3))
				Expect(e.EventType).To(Equal(EventTypeMGet))

				_ = cacheT.MGet(context.Background(), "key", ids,
					func(ctx context.Context, ints []int) (map[int]*object, error) {
						return map[int]*object{1: {Str: "str1", Num: 1}, 2: {Str: "str2", Num: 2}}, nil
					})

				timeout := make(chan bool, 1)
				go func() {
					time.Sleep(100 * time.Millisecond)
					timeout <- true
				}()
				select {
				case <-jetCache.eventCh:
					Expect(1).To(Equal(2))
				case ok := <-timeout:
					Expect(ok).To(BeTrue())
				}
			})

			It("Once with sync local", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				do := func(context.Context) (any, error) {
					return nil, errTestNotFound
				}
				var value string
				_ = jetCache.Once(ctx, key, Value(&value), Do(do))
				e, ok := <-jetCache.eventCh
				Expect(ok).To(BeTrue())
				Expect(e.Keys[0]).To(Equal(key))
				Expect(e.EventType).To(Equal(EventTypeSetByOnce))
			})

			It("Once with refresh and sync local", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				do := func(context.Context) (any, error) {
					return nil, errTestNotFound
				}
				var value string
				_ = jetCache.Once(ctx, key, Value(&value), Do(do), Refresh(true))
				e, ok := <-jetCache.eventCh
				Expect(ok).To(BeTrue())
				Expect(e.Keys[0]).To(Equal(key))
				Expect(e.EventType).To(Equal(EventTypeSetByOnce))

				timeout := make(chan bool, 1)
				go func() {
					time.Sleep(refreshDuration + 10*time.Millisecond)
					timeout <- true
				}()
				select {
				case e, ok := <-jetCache.eventCh:
					Expect(ok).To(BeTrue())
					Expect(e.EventType).To(Equal(EventTypeSetByRefresh))
				case ok := <-timeout:
					Expect(ok).To(BeFalse())
				}
			})

			It("send when eventCh full", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				var buf = new(bytes.Buffer)
				logger.SetDefaultLogger(&testLogger{})
				log.SetOutput(buf)

				for i := 0; i < testEventChSize+1; i++ {
					jetCache.send(EventTypeSet, key)
				}
				Expect(buf.String()).To(ContainSubstring("reach max send buffer"))
			})

			It("send wend eventCh closed", func() {
				var jetCache = cache.(*jetCache)
				if !jetCache.isSyncLocal() {
					return
				}

				var buf = new(bytes.Buffer)
				logger.SetDefaultLogger(&testLogger{})
				log.SetOutput(buf)

				close(jetCache.eventCh)
				jetCache.send(EventTypeSet, key)
				Expect(buf.String()).To(ContainSubstring("send syncEvent error(send on closed channel)"))
			})
		})
	}

	BeforeEach(func() {
		obj = &object{
			Str: "mystring",
			Num: 42,
		}
	})

	Context("with only remote", func() {
		BeforeEach(func() {
			rdb = newRdb()
			cache = newRemote(rdb)
			cacheT = NewT[int, *object](cache)
		})

		testCache()

		AfterEach(func() {
			_ = rdb.Close()
			cache.Close()
		})
	})

	for _, typ := range localTypes {
		Context(fmt.Sprintf("with both remote and local(%v)", typ), func() {
			BeforeEach(func() {
				rdb = newRdb()
				cache = newBoth(rdb, typ)
				cacheT = NewT[int, *object](cache)
			})

			testCache()
		})

		Context(fmt.Sprintf("with only local(%v)", typ), func() {
			BeforeEach(func() {
				rdb = nil
				cache = newLocal(typ)
				cacheT = NewT[int, *object](cache)
			})

			testCache()
		})
	}

	Context("with sync local", func() {
		BeforeEach(func() {
			rdb = newRdb()
			cache = newBoth(rdb, freeCache, true)
			cacheT = NewT[int, *object](cache)
		})

		testCache()

		AfterEach(func() {
			_ = rdb.Close()
			cache.Close()
		})
	})
})

func newRdb() *redis.Client {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	return redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
}

func newLocal(localType localType) Cache {
	return New(WithName("local"),
		WithLocal(localNew(localType)),
		WithErrNotFound(errTestNotFound),
		WithRefreshDuration(refreshDuration),
		WithStopRefreshAfterLastAccess(stopRefreshAfterLastAccess))
}

func newRemote(rds *redis.Client) Cache {
	return New(WithName("remote"),
		WithRemote(remote.NewGoRedisV8Adaptor(rds)),
		WithErrNotFound(errTestNotFound),
		WithRefreshDuration(refreshDuration),
		WithStopRefreshAfterLastAccess(stopRefreshAfterLastAccess))
}

func newBoth(rds *redis.Client, localType localType, syncLocal ...bool) Cache {
	return New(WithName("both"),
		WithRemote(remote.NewGoRedisV8Adaptor(rds)),
		WithLocal(localNew(localType)),
		WithErrNotFound(errTestNotFound),
		WithRefreshDuration(refreshDuration),
		WithSyncLocal(len(syncLocal) > 0 && syncLocal[0]),
		WithEventChBufSize(testEventChSize),
		WithStopRefreshAfterLastAccess(stopRefreshAfterLastAccess))
}

func localNew(localType localType) local.Local {
	if localType == tinyLFU {
		return local.NewTinyLFU(100000, localExpire)
	} else {
		id := atomic.AddInt32(&localId, 1)
		return local.NewFreeCache(256*local.MB, localExpire, strconv.Itoa(int(id)))
	}
}

const (
	mockUnmarshalErr = "err1"
	mockMarshalErr   = "err2"
)

func init() {
	encoding.RegisterCodec(mockDecode{})
	encoding.RegisterCodec(mockEncode{})
}

type (
	mockDecode struct{}
	mockEncode struct{}
)

func (mockDecode) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (mockDecode) Unmarshal([]byte, interface{}) error {
	return errors.New("mock Unmarshal error")
}

func (mockDecode) Name() string {
	return mockUnmarshalErr
}

func (mockEncode) Marshal(interface{}) ([]byte, error) {
	return nil, errors.New("mock Marshal error")
}

func (mockEncode) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (mockEncode) Name() string {
	return mockMarshalErr
}

var _ remote.Remote = (*mockGoRedisMGetMSetErrAdapter)(nil)

type mockGoRedisMGetMSetErrAdapter struct {
}

func (m mockGoRedisMGetMSetErrAdapter) SetEX(ctx context.Context, key string, value any, expire time.Duration) error {
	panic("implement me")
}

func (m mockGoRedisMGetMSetErrAdapter) SetNX(ctx context.Context, key string, value any, expire time.Duration) (val bool, err error) {
	panic("implement me")
}

func (m mockGoRedisMGetMSetErrAdapter) SetXX(ctx context.Context, key string, value any, expire time.Duration) (val bool, err error) {
	panic("implement me")
}

func (m mockGoRedisMGetMSetErrAdapter) Get(ctx context.Context, key string) (val string, err error) {
	panic("implement me")
}

func (m mockGoRedisMGetMSetErrAdapter) Del(ctx context.Context, key string) (val int64, err error) {
	panic("implement me")
}

func (m mockGoRedisMGetMSetErrAdapter) MGet(ctx context.Context, keys ...string) (map[string]any, error) {
	return nil, errors.New("any")
}

func (m mockGoRedisMGetMSetErrAdapter) MSet(ctx context.Context, value map[string]any, expire time.Duration) error {
	return errors.New("any")
}

func (m mockGoRedisMGetMSetErrAdapter) Nil() error {
	panic("implement me")
}

type testLogger struct{}

func (l *testLogger) Debug(format string, v ...any) {
	log.Println(fmt.Sprintf(format, v...))
}

func (l *testLogger) Info(format string, v ...any) {
	log.Println(fmt.Sprintf(format, v...))
}

func (l *testLogger) Warn(format string, v ...any) {
	log.Println(fmt.Sprintf(format, v...))
}

func (l *testLogger) Error(format string, v ...any) {
	log.Println(fmt.Sprintf(format, v...))
}
