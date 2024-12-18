<!-- TOC -->
* [Introduction](#introduction)
* [Serialization Methods](#serialization-methods)
* [Local and Remote Cache Options](#local-and-remote-cache-options)
* [Metrics Collection and Statistics](#metrics-collection-and-statistics)
* [Customizing Logging](#customizing-logging)
<!-- TOC -->

# Introduction

`jetcache-go` implements many embedded components through generic interfaces, making it easy for developers to integrate and reference for creating their own components.


# Serialization Methods

| Codec Method | Description                                          | Advantages                         |
|--------------|------------------------------------------------------|------------------------------------|
| native json  | Golang's built-in serialization tool                 | Good compatibility                 |
| msgpack      | msgpack with snappy compression (content > 64 bytes) | High performance, low memory usage |
| sonic        | Byte-based high-performance JSON serialization tool  | High performance                   |

You can also define your own serialization by implementing the `encoding.Codec` interface and registering it using `encoding.RegisterCodec`.

```go
import (
	_ "github.com/mgtv-tech/jetcache-go/encoding/yourCodec"
)

// Register your codec
encoding.RegisterCodec(yourCodec.Name)

mycache := cache.New(cache.WithName("any"),
	cache.WithRemote(...),
	cache.WithCodec(yourCodec.Name))
```


# Local and Remote Cache Options

| Name      | Type   | Features                                                     |
|-----------|--------|--------------------------------------------------------------|
| ristretto | Local  | High performance, high hit rate                              |
| freecache | Local  | Zero garbage collection overhead, strict memory usage limits |
| go-redis  | Remote | Popular GO Redis client                                      |

You can also implement your own local and remote caches by implementing the `remote.Remote` and `local.Local` interfaces.

> **FreeCache Usage Notes:**
>
> - The cache key size must be less than 65535, otherwise it cannot be stored in the local cache (The key is larger than 65535).
> - The cache value size must be less than 1/1024 of the total cache capacity, otherwise it cannot be stored in the local cache (The entry size needs to be less than 1/1024 of the cache size).
> - The embedded FreeCache instance internally shares an `innerCache` instance to prevent excessive memory usage when multiple cache instances use FreeCache. Therefore, the shared `innerCache` will use the memory capacity and expiration time configured during the first creation.


# Metrics Collection and Statistics

| Name            | Type     | Description                                                                                         |
|-----------------|----------|-----------------------------------------------------------------------------------------------------|
| logStats        | Embedded | Default metrics collector, statistics are printed to the log                                        |
| PrometheusStats | Plugin   | Statistics plugin provided by [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) |

You can also define your own metrics collector by implementing the `stats.Handler` interface.

Example: Using multiple metrics collectors simultaneously

```go
import (
	"context"
	"time"

	"github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
	pstats "github.com/mgtv-tech/jetcache-go-plugin/stats"
	"github.com/mgtv-tech/jetcache-go/stats"
)

mycache := cache.New(cache.WithName("any"),
	cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
	cache.WithStatsHandler(
		stats.NewHandles(false,
			stats.NewStatsLogger(cacheName),
			pstats.NewPrometheus(cacheName))))

obj := struct {
	Name string
	Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
	// Error handling
}
```

# Customizing Logging

```go
import "github.com/mgtv-tech/jetcache-go/logger"

// Set your Logger
logger.SetDefaultLogger(l logger.Logger)
```
