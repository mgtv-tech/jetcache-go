# jetcache-go

![banner](docs/images/banner.png)

<p>
<a href="https://github.com/mgtv-tech/jetcache-go/actions"><img src="https://github.com/mgtv-tech/jetcache-go/workflows/Go/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/mgtv-tech/jetcache-go"><img src="https://codecov.io/gh/mgtv-tech/jetcache-go/master/graph/badge.svg" alt="codeCov"></a>
<a href="https://goreportcard.com/report/github.com/mgtv-tech/jetcache-go"><img src="https://goreportcard.com/badge/github.com/mgtv-tech/jetcache-go" alt="Go Repport Card"></a>
<a href="https://github.com/mgtv-tech/jetcache-go/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
<a href="https://github.com/mgtv-tech/jetcache-go/releases"><img src="https://img.shields.io/github/release/mgtv-tech/jetcache-go" alt="Release"></a>
</p>

Translate to: [简体中文](README_zh.md)

# Overview

[jetcache-go](https://github.com/mgtv-tech/jetcache-go) is a general-purpose cache access framework based on
[go-redis/cache](https://github.com/go-redis/cache). It implements the core features of the Java version of
[JetCache](https://github.com/alibaba/jetcache), including:

- ✅ Flexible combination of two-level caching: You can use memory, Redis, or your own custom storage method.
- ✅ The `Once` interface adopts the `singleflight` pattern, which is highly concurrent and thread-safe.
- ✅ By default, [MsgPack](https://github.com/vmihailenco/msgpack) is used for encoding and decoding values. Optional [sonic](https://github.com/bytedance/sonic) and native json.
- ✅ The default local cache implementation includes [Ristretto](https://github.com/dgraph-io/ristretto) and [FreeCache](https://github.com/coocood/freecache).
- ✅ The default distributed cache implementation is based on [go-redis/v9](https://github.com/redis/go-redis), and you can also customize your own implementation.
- ✅ You can customize the errNotFound error and use placeholders to prevent cache penetration by caching empty results.
- ✅ Supports asynchronous refreshing of distributed caches.
- ✅ Metrics collection: By default, it prints statistical metrics (QPM, Hit, Miss, Query, QueryFail) through logs.
- ✅ Automatic degradation of distributed cache query failures.
- ✅ The `MGet` interface supports the `Load` function. In a distributed caching scenario, the Pipeline mode is used to improve performance. (v1.1.0+)
- ✅ Invalidate local caches (in all Go processes) after updates (v1.1.1+)

# Learning JetCache-go

Visit [documentation](/docs/EN/GettingStarted.md) for more details.

# Installation

To start using the latest version of jetcache-go, you can import the library into your project:

```shell
go get github.com/mgtv-tech/jetcache-go
```

# Getting started

## Basic Usage

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

var errRecordNotFound = errors.New("mock gorm.errRecordNotFound")

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

# Contributing

Everyone is welcome to help improve jetcache-go. If you have any questions, suggestions, or want to add other features, please submit an issue or PR directly.

Please follow these steps to submit a PR:

- Clone the repository
- Create a new branch: name it feature-xxx for new features or bug-xxx for bug fixes
- Describe the changes in detail in the PR


# Contact

If you have any questions, please contact `daoshenzzg@gmail.com`.
