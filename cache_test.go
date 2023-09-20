package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redis/go-redis/v9"

	"github.com/daoshenzzg/jetcache-go/local"
	"github.com/daoshenzzg/jetcache-go/remote"
	"github.com/daoshenzzg/jetcache-go/util"
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
		obj   *object
		rdb   *redis.Client
		cache Cache
	)

	testCache := func() {
		It("Remote and Local both nil", func() {
			nilCache := New().(*jetCache)

			err := nilCache.Get(ctx, "key", nil)
			Expect(err).To(Equal(ErrRemoteLocalBothNil))

			err = nilCache.Delete(ctx, "key")
			Expect(err).To(Equal(ErrRemoteLocalBothNil))

			err = nilCache.Set(ctx, "key", Do(func() (interface{}, error) {
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

		Describe("Once func", func() {
			It("works with err not found", func() {
				key := "cache-err-not-found"
				do := func() (interface{}, error) {
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
					Expect(val).To(Equal(string(NotFoundPlaceholder)))
				}

				_ = cache.Set(ctx, key, Value(value), Do(do))
				do = func() (interface{}, error) {
					return nil, nil
				}
				err = cache.Once(ctx, key, Value(&value), Do(do))
				Expect(err).To(Equal(errTestNotFound))
				Expect(cache.Get(context.Background(), key, &value)).To(Equal(errTestNotFound))
				Expect(cache.Exists(context.Background(), key)).To(BeFalse())

				_ = cache.Delete(context.Background(), key)
				errAny := errors.New("any")
				do = func() (interface{}, error) {
					return nil, errAny
				}
				err = cache.Once(ctx, key, Value(&value), Do(do))
				Expect(err).To(Equal(errAny))
			})

			It("works without value and error result", func() {
				var callCount int64
				perform(100, func(int) {
					err := cache.Once(ctx, key, Do(func() (interface{}, error) {
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
					err := cache.Once(ctx, key, Value(&n), Do(func() (interface{}, error) {
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
				err := cache.Once(ctx, key, Value(&value), TTL(-1), Do(func() (interface{}, error) {
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
					Do(func() (interface{}, error) {
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
					Do(func() (interface{}, error) {
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
					Do(func() (interface{}, error) {
						return "V1", nil
					}))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				_, err = rdb.SetEx(ctx, key, "V2", time.Minute).Result()
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
					doFunc    = func() (interface{}, error) {
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
				_, err = rdb.SetEx(ctx, key, "V3", time.Minute).Result()
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
					doFunc   = func() (interface{}, error) {
						return "V1", nil
					}
					value string
					err   error
				)
				err = jetCache.Once(ctx, key, Value(&value), TTL(time.Minute), Refresh(true), Do(doFunc))
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal("V1"))

				perform(100, func(i int) {
					rdb.Del(context.TODO(), lockKey)
					jetCache.externalLoad(ctx, &refreshTask{key: key, do: doFunc, ttl: time.Minute}, time.Now())
				})
				b, ok := jetCache.local.Get(key)
				Expect(ok).To(BeTrue())
				Expect(string(b)).To(Equal("V1"))

				_, err = rdb.SetEx(ctx, key, "V2", time.Minute).Result()
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
					do: func() (interface{}, error) {
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
			})

			testCache()
		})

		Context(fmt.Sprintf("with only local(%v)", typ), func() {
			BeforeEach(func() {
				rdb = nil
				cache = newLocal(typ)
			})

			testCache()
		})
	}
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
		WithRemote(remote.NewGoRedisV9Adaptor(rds)),
		WithErrNotFound(errTestNotFound),
		WithRefreshDuration(refreshDuration),
		WithStopRefreshAfterLastAccess(stopRefreshAfterLastAccess))
}

func newBoth(rds *redis.Client, localType localType) Cache {
	return New(WithName("both"),
		WithRemote(remote.NewGoRedisV9Adaptor(rds)),
		WithLocal(localNew(localType)),
		WithErrNotFound(errTestNotFound),
		WithRefreshDuration(refreshDuration),
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
