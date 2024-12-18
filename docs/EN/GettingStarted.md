![banner](/docs/images/banner.png)

<!-- TOC -->
* [Overview](#overview)
* [Product Comparison](#product-comparison)
* [Learning jetcache-go](#learning-jetcache-go)
* [Installation](#installation)
* [Quick started](#quick-started)
<!-- TOC -->

# Overview

`jetcache-go` is a general-purpose caching framework built upon and extending [go-redis/cache](https://github.com/go-redis/cache). It implements core features similar to the Java version of [JetCache](https://github.com/alibaba/jetcache), including:

- ✅ **Flexible Two-Level Caching:**  Supports local cache, distributed cache, and a combination of both.
- ✅ **`Once` Interface with Singleflight:**  High concurrency and thread safety using the singleflight pattern.
- ✅ **Multiple Encoding Options:** Defaults to [MsgPack](https://github.com/vmihailenco/msgpack) for value encoding/decoding.  [sonic](https://github.com/bytedance/sonic) and native `json` are also supported.
- ✅ **Built-in Local Cache Implementations:**  Provides implementations using [Ristretto](https://github.com/dgraph-io/ristretto) and [FreeCache](https://github.com/coocood/freecache).
- ✅ **Distributed Cache Adapter:**  Defaults to an adapter for [go-redis/v9](https://github.com/redis/go-redis), but custom implementations are also supported.
- ✅ **`errNotFound` Customization:**  Prevents cache penetration by caching null results using a placeholder.
- ✅ **Asynchronous Distributed Cache Refresh:**  Supports enabling asynchronous refresh of distributed caches.
- ✅ **Metrics Collection:**  Provides default logging of cache statistics (QPM, Hit, Miss, Query, QueryFail).
- ✅ **Automatic Distributed Cache Query Degradation:**  Handles failures gracefully.
- ✅ **`MGet` Interface with `Load` Function:**  Supports pipeline mode for distributed cache scenarios (v1.1.0+).
- ✅ **Support for Invalidating Local Caches Across All Go Processes:** After cache updates (v1.1.1+).


# Product Comparison

| Feature               | eko/gocache | go-redis/cache | mgtv-tech/jetcache-go |
|-----------------------|-------------|----------------|-----------------------|
| Multi-level Caching   | Yes         | Yes            | Yes                   |
| Loadable Caching      | Yes         | Yes            | Yes                   |
| Generics Support      | Yes         | No             | Yes                   |
| Singleflight Pattern  | Yes         | Yes            | Yes                   |
| Cache Update Listener | No          | No             | Yes                   |
| Auto Refresh          | No          | No             | Yes                   |
| Metrics Collection    | Yes         | Yes (simple)   | Yes                   |
| Null Object Caching   | No          | No             | Yes                   |
| Bulk Query            | No          | No             | Yes                   |
| Sparse List Cache     | No          | No             | Yes                   |

# Learning jetcache-go
- GettingStarted
- [Cache API](/docs/EN/CacheAPI.md)
- [Config](/docs/EN/Config.md)
- [Embedded](/docs/EN/Embedded.md)
- [Metrics](/docs/EN/Stat.md)
- [Plugin](/docs/EN/Plugin.md)

# Installation

To use the latest version of `jetcache-go`, import the library into your project:

```shell
go get github.com/mgtv-tech/jetcache-go
```

# Quick started

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/local"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

var errRecordNotFound = errors.New("mock gorm.ErrRecordNotFound")

type object struct {
	Str string
	Num int
}

func Example_basicUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"localhost": ":6379",
		},
	})

	mycache := cache.New(cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound))

	ctx := context.TODO()
	key := "mykey:1"
	obj, _ := mockDBGetObject(1)
	if err := mycache.Set(ctx, key, cache.Value(obj), cache.TTL(time.Hour)); err != nil {
		panic(err)
	}

	var wanted object
	if err := mycache.Get(ctx, key, &wanted); err == nil {
		fmt.Println(wanted)
	}
	// Output: {mystring 42}

	mycache.Close()
}

func Example_advancedUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"localhost": ":6379",
		},
	})

	mycache := cache.New(cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRefreshDuration(time.Minute))

	ctx := context.TODO()
	key := "mykey:1"
	obj := new(object)
	if err := mycache.Once(ctx, key, cache.Value(obj), cache.TTL(time.Hour), cache.Refresh(true),
		cache.Do(func(ctx context.Context) (any, error) {
			return mockDBGetObject(1)
		})); err != nil {
		panic(err)
	}
	fmt.Println(obj)
	// Output: &{mystring 42}

	mycache.Close()
}

func Example_mGetUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"localhost": ":6379",
		},
	})

	mycache := cache.New(cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRemoteExpiry(time.Minute),
	)
	cacheT := cache.NewT[int, *object](mycache)

	ctx := context.TODO()
	key := "mget"
	ids := []int{1, 2, 3}

	ret := cacheT.MGet(ctx, key, ids, func(ctx context.Context, ids []int) (map[int]*object, error) {
		return mockDBMGetObject(ids)
	})

	var b bytes.Buffer
	for _, id := range ids {
		b.WriteString(fmt.Sprintf("%v", ret[id]))
	}
	fmt.Println(b.String())
	// Output: &{mystring 1}&{mystring 2}<nil>

	cacheT.Close()
}

func Example_syncLocalUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"localhost": ":6379",
		},
	})

	sourceID := "12345678" // Unique identifier for this cache instance
	channelName := "syncLocalChannel"
	pubSub := ring.Subscribe(context.Background(), channelName)

	mycache := cache.New(cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRemoteExpiry(time.Minute),
		cache.WithSourceId(sourceID),
		cache.WithSyncLocal(true),
		cache.WithEventHandler(func(event *cache.Event) {
			// Broadcast local cache invalidation for the received keys
			bs, _ := json.Marshal(event)
			ring.Publish(context.Background(), channelName, string(bs))
		}),
	)
	obj, _ := mockDBGetObject(1)
	if err := mycache.Set(context.TODO(), "mykey", cache.Value(obj), cache.TTL(time.Hour)); err != nil {
		panic(err)
	}

	go func() {
		for {
			msg := <-pubSub.Channel()
			var event *cache.Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				panic(err)
			}
			fmt.Println(event.Keys)

			// Invalidate local cache for received keys (except own events)
			if event.SourceID != sourceID {
				for _, key := range event.Keys {
					mycache.DeleteFromLocalCache(key)
				}
			}
		}
	}()

	// Output: [mykey]
	mycache.Close()
	time.Sleep(time.Second)
}

func mockDBGetObject(id int) (*object, error) {
	if id > 100 {
		return nil, errRecordNotFound
	}
	return &object{Str: "mystring", Num: 42}, nil
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
```
