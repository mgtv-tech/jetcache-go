<!-- TOC -->
* [Introduction](#introduction)
* [Multiple Serialization Methods](#multiple-serialization-methods)
* [Local and Remote Cache Options](#local-and-remote-cache-options)
* [Metrics Collection and Statistics](#metrics-collection-and-statistics)
* [Custom Logger](#custom-logger)
<!-- TOC -->

# Introduction

`jetcache-go` provides a unified interface for various built-in components, simplifying integration and providing examples for developers to create their own components.


# Multiple Serialization Methods

`jetcache-go` supports several serialization methods, offering flexibility and performance optimization depending on your needs.  Here's a comparison:

| Codec Method  | Description                                           | Advantages                             |
|---------------|-------------------------------------------------------|----------------------------------------|
| `native json` | Golang's built-in JSON serialization tool             | Simplicity, readily available          |
| `msgpack`     | msgpack with snappy compression (for data > 64 bytes) | High performance, low memory footprint |
| `sonic`       | ByteDance's high-performance JSON serialization tool  | High performance                       |


You can also customize your serialization by implementing the `encoding.Codec` interface and registering it using `encoding.RegisterCodec`.  This allows for integration with other serialization libraries or custom serialization logic tailored to your specific data structures.


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

`jetcache-go` offers a variety of local and remote cache implementations, providing flexibility to choose the best option for your application's needs.

| Name                                                | Type   | Features                                                     |
|-----------------------------------------------------|--------|--------------------------------------------------------------|
| [ristretto](https://github.com/dgraph-io/ristretto) | Local  | High performance, high hit ratio                             |
| [freecache](https://github.com/coocood/freecache)   | Local  | Zero garbage collection overhead, strict memory usage limits |
| [go-redis](https://github.com/redis/go-redis)       | Remote | Popular Go Redis client                                      |


You can also implement your own local and remote caches by implementing the `remote.Remote` and `local.Local` interfaces respectively.


> **FreeCache Usage Notes:**
>
> * Keys must be less than 65535 bytes.  Larger keys will result in an error ("The key is larger than 65535").
> * Values must be less than 1/1024 of the total cache size. Larger values will result in an error ("The entry size needs to be less than 1/1024 of the cache size").
> * Embedded FreeCache instances share an internal `innerCache` instance. This prevents excessive memory consumption when multiple cache instances use FreeCache.  Therefore, the memory capacity and expiration time will be determined by the configuration of the first created instance.


# Metrics Collection and Statistics

`jetcache-go` provides several ways to collect and report cache metrics:

| Name              | Type     | Description                                                                                                                     |
|-------------------|----------|---------------------------------------------------------------------------------------------------------------------------------|
| `logStats`        | Built-in | Default metrics collector; statistics are printed to the log.                                                                   |
| `PrometheusStats` | Plugin   | Statistics plugin provided by [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) for Prometheus integration. |


You can also create custom metrics collectors by implementing the `stats.Handler` interface.


Example: Using multiple Metrics collectors simultaneously

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
    // Handle error
}
```

# Custom Logger

```go
import "github.com/mgtv-tech/jetcache-go/logger"

// Set your Logger
logger.SetDefaultLogger(l logger.Logger)
```

