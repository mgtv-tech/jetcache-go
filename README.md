<p>
<a href="https://github.com/daoshenzzg/jetcache-go/actions"><img src="https://github.com/daoshenzzg/jetcache-go/workflows/Go/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/daoshenzzg/jetcache-go"><img src="https://codecov.io/gh/daoshenzzg/jetcache-go/master/graph/badge.svg" alt="codeCov"></a>
<a href="https://goreportcard.com/report/github.com/daoshenzzg/jetcache-go"><img src="https://goreportcard.com/badge/github.com/daoshenzzg/jetcache-go" alt="Go Repport Card"></a>
<a href="https://github.com/daoshenzzg/jetcache-go/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
</p>

Translate to: [简体中文](README_zh.md)

# Introduction
[jetcache-go](https://github.com/daoshenzzg/jetcache-go) is a general-purpose cache access framework based on
[go-redis/cache](https://github.com/go-redis/cache). It implements the core features of the Java version of 
[JetCache](https://github.com/alibaba/jetcache), including:

- ✅ Flexible combination of two-level caching: You can use memory, Redis, or your own custom storage method.
- ✅ The `Once` interface adopts the `singleflight` pattern, which is highly concurrent and thread-safe.
- ✅ By default, [sonic](https://github.com/bytedance/sonic) is used for encoding and decoding values. Optional [MsgPack](https://github.com/vmihailenco/msgpack) and native json.
- ✅ The default local cache implementation includes [TinyLFU](https://github.com/dgryski/go-tinylfu) and [FreeCache](https://github.com/coocood/freecache).
- ✅ The default distributed cache implementation is based on [go-redis/v8](https://github.com/redis/go-redis), and you can also customize your own implementation.
- ✅ You can customize the errNotFound error and use placeholders to prevent cache penetration by caching empty results.
- ✅ Supports asynchronous refreshing of distributed caches.
- ✅ Metrics collection: By default, it prints statistical metrics (QPM, Hit, Miss, Query, QueryFail) through logs.
- ✅ Automatic degradation of distributed cache query failures.
- ✅ The `MGet` interface supports the `Load` function. In a distributed caching scenario, the Pipeline mode is used to improve performance.

# Installation
To start using the latest version of jetcache-go, you can import the library into your project:
```shell
go get github.com/daoshenzzg/jetcache-go
```

## Getting started

### Basic Usage
```go
package cache_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/daoshenzzg/jetcache-go"
	"github.com/daoshenzzg/jetcache-go/local"
	"github.com/daoshenzzg/jetcache-go/remote"
	"github.com/daoshenzzg/jetcache-go/util"
)

var errRecordNotFound = errors.New("mock gorm.errRecordNotFound")

type object struct {
	Str string
	Num int
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

func Example_basicUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"localhost": ":6379",
		},
	})

	mycache := cache.New[string, any](cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound))

	ctx := context.TODO()
	key := util.JoinAny(":", "mykey", 1)
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

	mycache := cache.New[string, any](cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRefreshDuration(time.Minute))

	ctx := context.TODO()
	key := util.JoinAny(":", "mykey", 1)
	obj := new(object)
	if err := mycache.Once(ctx, key, cache.Value(obj), cache.Refresh(true), cache.Do(func(ctx context.Context) (any, error) {
		return mockDBGetObject(1)
	})); err != nil {
		panic(err)
	}
	fmt.Println(obj)
	//Output: &{mystring 42}

	mycache.Close()
}

func Example_mGetUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"localhost": ":6379",
		},
	})

	mycache := cache.New[int, *object](cache.WithName("any"),
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRemoteExpiry(time.Minute),
	)

	ctx := context.TODO()
	key := "mykey"
	ids := []int{1, 2, 3}

	ret := mycache.MGet(ctx, key, ids, func(ctx context.Context, ids []int) (map[int]*object, error) {
		return mockDBMGetObject(ids)
	})

	var b bytes.Buffer
	for _, id := range ids {
		b.WriteString(fmt.Sprintf("%v", ret[id]))
	}
	fmt.Println(b.String())
	//Output: &{mystring 1}&{mystring 2}<nil>

	mycache.Close()
}
```

### Configure settings
```go
// Options are used to store cache options.
type Options struct {
    name                       string        // Cache name, used for log identification and metric reporting
    remote                     remote.Remote // Remote is distributed cache, such as Redis.
    local                      local.Local   // Local is memory cache, such as FreeCache.
    codec                      string        // Value encoding and decoding method. Default is "msgpack.Name". You can also customize it.
    errNotFound                error         // Error to return for cache miss. Used to prevent cache penetration.
    remoteExpiry               time.Duration // Remote cache ttl, Default is 1 hour.
    notFoundExpiry             time.Duration // Duration for placeholder cache when there is a cache miss. Default is 1 minute.
    offset                     time.Duration // Expiration time jitter factor for cache misses.
    refreshDuration            time.Duration // Interval for asynchronous cache refresh. Default is 0 (refresh is disabled).
    stopRefreshAfterLastAccess time.Duration // Duration for cache to stop refreshing after no access. Default is refreshDuration + 1 second.
    refreshConcurrency         int           // Maximum number of concurrent cache refreshes. Default is 4.
    statsDisabled              bool          // Flag to disable cache statistics.
    statsHandler               stats.Handler // Metrics statsHandler collector.
}
```

### Cache metrics collection and statistics.
You can implement the `stats.Handler` interface and register it with the Cache component to customize metric collection, 
for example, using [Prometheus](https://github.com/prometheus/client_golang) to collect metrics. We have provided a 
default implementation that logs the statistical metrics, as shown below:
```shell
2023/09/11 16:42:30.695294 statslogger.go:178: [INFO] jetcache-go stats last 1m0s.
cache       |         qpm|   hit_ratio|         hit|        miss|       query|  query_fail
------------+------------+------------+------------+------------+------------+------------
bench       |   216440123|     100.00%|   216439867|         256|         256|           0|
bench_local |   216440123|     100.00%|   216434970|        5153|           -|           -|
bench_remote|        5153|      95.03%|        4897|         256|           -|           -|
------------+------------+------------+------------+------------+------------+------------
```

### Custom Logger
```go
import "github.com/daoshenzzg/jetcache-go/logger"

// Set your Logger
logger.SetDefaultLogger(l logger.Logger)
```

### Custom Encoding and Decoding
```go
import (
    "github.com/daoshenzzg/jetcache-go"
    "github.com/daoshenzzg/jetcache-go/encoding"
)

// Register your codec
encoding.RegisterCodec(codec Codec)

// Set your codec name
mycache := cache.New("any",
    cache.WithRemote(...),
    cache.WithCodec(yourCodecName string))
```

### Usage Scenarios

#### Automatic Cache Refresh
`jetcache-go` provides automatic cache refresh capability to prevent cache avalanche from hitting the database when the cache expires. Automatic refresh is suitable for scenarios with fewer keys, low real-time requirements, and high loading overhead. The above code specifies a refresh every minute, and stops refreshing after 1 hour if there is no access. If the cache is redis or the last level of a multi-level cache is redis, the cache loading behavior is globally unique, which means that only one server is refreshing at a time regardless of the number of servers, in order to reduce the load on the backend.
```go
mycache := cache.New[string, any](cache.WithName("any"),
       // ...
       // cache.WithRefreshDuration sets the asynchronous refresh interval
       cache.WithRefreshDuration(time.Minute),
       // cache.WithStopRefreshAfterLastAccess sets the time to cancel the refresh task after the cache key is not accessed
        cache.WithStopRefreshAfterLastAccess(time.Hour))

// `Once` interface starts automatic refresh by `cache.Refresh(true)`
err := mycache.Once(ctx, key, cache.Value(obj), cache.Refresh(true), cache.Do(func(ctx context.Context) (any, error) {
    return mockDBGetObject(1)
}))
```

#### MGet Batch Query
`MGet` uses golang's Generics + Load function to provide a user-friendly way to batch query entities corresponding to IDs in multi-level cache. If the cache is redis or the last level of a multi-level cache is redis, Pipeline is used to implement read and write operations to improve performance. It is worth noting that for abnormal scenarios (IO exceptions, serialization exceptions, etc.), our design philosophy is to provide lossy services as much as possible to prevent penetration.
```go
mycache := cache.New[int, *object](cache.WithName("any"),
       // ...
       cache.WithRemoteExpiry(time.Minute),
    )

ctx := context.TODO()
key := "mykey"
ids := []int{1, 2, 3}

ret := mycache.MGet(ctx, key, ids, func(ctx context.Context, ids []int) (map[int]*object, error) {
    return mockDBMGetObject(ids)
})
```

### Codec Selection
`jetcache-go` implements three serialization and deserialization (codec) methods by default: [sonic](https://github.com/bytedance/sonic)、[MsgPack](https://github.com/vmihailenco/msgpack), and native json.

**Selection Guide:**
- **For high performance:** If the local cache hit rate is extremely high, but the deserialization operation of converting byte arrays to objects in the local cache consumes a lot of CPU, choose sonic.
- **For balance between performance and storage space:** Choose MsgPack. MsgPack uses MsgPack for serialization and deserialization, and snappy compression is used when the content exceeds 64 bytes.
