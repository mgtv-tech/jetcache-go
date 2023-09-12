<p>
<a href="https://github.com/daoshenzzg/jetcache-go/actions"><img src="https://github.com/daoshenzzg/jetcache-go/workflows/Go/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/daoshenzzg/jetcache-go"><img src="https://codecov.io/gh/daoshenzzg/jetcache-go/master/graph/badge.svg" alt="codeCov"></a>
<a href="https://github.com/daoshenzzg/jetcache-go/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
</p>


Translate to: [简体中文](README_zh.md)

# Introduction
[jetcache-go](https://github.com/daoshenzzg/jetcache-go) is a general-purpose cache access framework based on
[go-redis/cache](https://github.com/go-redis/cache). It implements the core features of the Java version of 
[JetCache](https://github.com/alibaba/jetcache), including:

- ✅ Flexible combination of two-level caching: You can use memory, Redis, or your own custom storage method.
- ✅ The Once interface adopts the `singleflight` pattern, which is highly concurrent and thread-safe.
- ✅ By default, [MsgPack](https://github.com/vmihailenco/msgpack) is used for encoding and decoding values.
- ✅ The default local cache implementation includes [TinyLFU](https://github.com/dgryski/go-tinylfu) and [FreeCache](https://github.com/coocood/freecache).
- ✅ The default centralized cache implementation is based on [go-redis/v8](https://github.com/redis/go-redis), and you can also customize your own implementation.
- ✅ You can customize the errNotFound error and use placeholders to prevent cache penetration by caching empty results.
- ✅ Supports asynchronous refreshing of distributed caches.
- ✅ Metrics collection: By default, it prints statistical metrics (QPM, Hit, Miss, Query, QueryFail) through logs.

# Installation
To start using the latest version of jetcache-go, you can import the library into your project:
```shell
go get https://github.com/daoshenzzg/jetcache-go
```

## Getting started

### Basic Usage
```go
package cache_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/jetcache-go"
	"github.com/jetcache-go/local"
	"github.com/jetcache-go/logger"
	"github.com/jetcache-go/remote"
	"github.com/jetcache-go/util"
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

func Example_basicUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": ":6379",
			"server2": ":6380",
		},
	})

	mycache := cache.New("basicUsage",
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound))

	ctx := context.TODO()
	key := util.JoinAny(":", "mykey", 1)
	obj, _ := mockDBGetObject(1)

	if err := mycache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: obj,
		TTL:   time.Hour,
	}); err != nil {
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
	logger.SetLevel(logger.LevelInfo)

	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": ":6379",
			"server2": ":6380",
		},
	})

	mycache := cache.New("advancedUsage",
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRefreshDuration(time.Minute))

	obj := new(object)
	err := mycache.Once(&cache.Item{
		Key:   util.JoinAny(":", "mykey", 1),
		Value: obj, // destination
		Do: func(*cache.Item) (interface{}, error) {
			return mockDBGetObject(1)
		},
		Refresh: true, // auto refreshment
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(obj)
	//Output: &{mystring 42}

	mycache.Close()
}
```

### Configure settings
```go
// Options are used to store cache options.
type Options struct {
    remote                     remote.Remote // Remote cache.
    local                      local.Local   // Local cache.
    codec                      string        // Value encoding and decoding method. Default is "json.Name" or "msgpack.Name". You can also customize it.
    errNotFound                error         // Error to return for cache miss. Used to prevent cache penetration.
    notFoundExpiry             time.Duration // Duration for placeholder cache when there is a cache miss. Default is 1 minute.
    refreshDuration            time.Duration // Interval for asynchronous cache refresh. Default is 0 (refresh is disabled).
    stopRefreshAfterLastAccess time.Duration // Duration for cache to stop refreshing after no access. Default is refreshDuration + 1 second.
    refreshConcurrency         int           // Maximum number of concurrent cache refreshes. Default is 4.
    statsDisabled              bool          // Flag to disable cache statistics.
    statsHandler               stats.Handler // Metrics statsHandler collector.
}
```

### Cache metrics collection and statistics.
You can implement the `stats.Handler` interface and register it with the Cache component. We have provided a default
implementation that logs the statistical metrics, as shown below:
```shell
2023/09/11 16:42:30.695294 statslogger.go:178: [INFO] jetcache-go stats last 1 minute.
cache       |         qpm|   hit_ratio|         hit|        miss|       query|  query_fail
------------+------------+------------+------------+------------+------------+------------
bench       |   216440123|     100.00%|   216439867|         256|         256|           0|
bench_local |   216440123|     100.00%|   216434970|        5153|           -|           -|
bench_remote|        5153|      95.03%|        4897|         256|           -|           -|
------------+------------+------------+------------+------------+------------+------------
```

### Custom Logger
```go
import "github.com/jetcache-go/logger"

// Set your Logger
logger.SetDefaultLogger(l logger.Logger)
```

### Custom Encoding and Decoding
```go
import (
    "github.com/jetcache-go"
    "github.com/jetcache-go/encoding"
)

// Register your codec
encoding.RegisterCodec(codec Codec)

// Set your codec name
cache.WithCodec(yourCodecName string)
```
