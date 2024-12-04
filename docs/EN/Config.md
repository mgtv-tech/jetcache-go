<!-- TOC -->
* [Cache Configuration Options](#cache-configuration-options)
* [Creating Cache Instances](#creating-cache-instances)
  * [Example 1: Creating a Two-Level Cache Instance ("Both")](#example-1-creating-a-two-level-cache-instance-both)
  * [Example 2: Creating a Local-Only Cache Instance ("Local")](#example-2-creating-a-local-only-cache-instance-local)
  * [Example 3: Creating a Remote-Only Cache Instance ("Remote")](#example-3-creating-a-remote-only-cache-instance-remote)
  * [Example 4: Creating a Cache Instance with the jetcache-go-plugin Prometheus Statistics Plugin](#example-4-creating-a-cache-instance-with-the-jetcache-go-plugin-prometheus-statistics-plugin)
  * [Example 5: Creating a Cache Instance and Configuring `errNotFound` to Prevent Cache Penetration](#example-5-creating-a-cache-instance-and-configuring-errnotfound-to-prevent-cache-penetration)
<!-- TOC -->

# Cache Configuration Options

| Configuration Item           | Type                      | Default Value                | Description                                                                                                                                                                                                                                                          |
|------------------------------|---------------------------|------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name`                       | `string`                  | `default`                    | Cache name, used for log identification and metrics reporting.                                                                                                                                                                                                       |
| `remote`                     | `remote.Remote` interface | `nil`                        | Distributed cache, such as Redis.  A custom implementation can be provided by implementing the `remote.Remote` interface.                                                                                                                                            |
| `local`                      | `local.Local` interface   | `nil`                        | In-memory cache, such as FreeCache, TinyLFU. A custom implementation can be provided by implementing the `local.Local` interface.                                                                                                                                    |
| `codec`                      | `string`                  | `msgpack`                    | Encoding and decoding method for values. Defaults to "msgpack". Options: `msgpack` \| `sonic`. A custom implementation can be provided by implementing the `encoding.Codec` interface and registering it.                                                            |
| `separatorDisabled`          | `bool`                    | `false`                      | Disables the cache key separator. Defaults to `false`. If `true`, the cache key will not use a separator. Primarily used for concatenating generic interface cache keys and IDs.                                                                                     |
| `separator`                  | `string`                  | `:`                          | Cache key separator. Defaults to ":". Primarily used for concatenating generic interface cache keys and IDs.                                                                                                                                                         |
| `errNotFound`                | `error`                   | `nil`                        | Error returned when a cache miss occurs, e.g., `gorm.ErrRecordNotFound`. Used to prevent cache penetration (caching null objects).                                                                                                                                   |
| `remoteExpiry`               | `time.Duration`           | 1 hour                       | Remote cache TTL, defaults to 1 hour.                                                                                                                                                                                                                                |
| `notFoundExpiry`             | `time.Duration`           | 1 minute                     | Expiration time for placeholder cache entries on cache misses. Defaults to 1 minute.                                                                                                                                                                                 |
| `offset`                     | `time.Duration`           | (0,10] seconds               | Expiration time jitter factor for cache misses.                                                                                                                                                                                                                      |
| `refreshDuration`            | `time.Duration`           | 0                            | Interval for asynchronous cache refresh. Defaults to 0 (refresh disabled).                                                                                                                                                                                           |
| `stopRefreshAfterLastAccess` | `time.Duration`           | `refreshDuration + 1 second` | Duration before cache refresh stops (after last access).                                                                                                                                                                                                             |
| `refreshConcurrency`         | `int`                     | 4                            | Maximum number of concurrent refresh tasks in the cache refresh task pool.                                                                                                                                                                                           |
| `statsDisabled`              | `bool`                    | `false`                      | Flag to disable cache statistics.                                                                                                                                                                                                                                    |
| `statsHandler`               | `stats.Handler` interface | `stats.NewStatsLogger`       | Metrics collector.  Defaults to a built-in `log` based collector.  The [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) provides a `Prometheus` plugin.  A custom implementation can be provided by implementing the `stats.Handler` interface. |
| `sourceID`                   | `string`                  | 16-character random string   | **Cache Event Broadcasting:** Unique identifier for the cache instance.                                                                                                                                                                                              |
| `syncLocal`                  | `bool`                    | `false`                      | **Cache Event Broadcasting:** Enables synchronization of local cache events (only applicable to "Both" cache type).                                                                                                                                                  |
| `eventChBufSize`             | `int`                     | 100                          | **Cache Event Broadcasting:** Buffer size for the event channel (defaults to 100).                                                                                                                                                                                   |
| `eventHandler`               | `func(event *Event)`      | `nil`                        | **Cache Event Broadcasting:** Function to handle local cache invalidation events.                                                                                                                                                                                    |


# Creating Cache Instances

## Example 1: Creating a Two-Level Cache Instance ("Both")

```go
import (
    "context"
    "time"

    "github.com/jinzhu/gorm"
	"github.com/mgtv-tech/jetcache-go"
    "github.com/mgtv-tech/jetcache-go/local"
    "github.com/mgtv-tech/jetcache-go/remote"
    "github.com/redis/go-redis/v9"
)
ring := redis.NewRing(&redis.RingOptions{
    Addrs: map[string]string{
        "localhost": ":6379",
    },
})

// Create a two-level cache instance
mycache := cache.New(cache.WithName("any"),
    cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
    cache.WithLocal(local.NewTinyLFU(10000, time.Minute)),
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
    // Handle error
}
```

## Example 2: Creating a Local-Only Cache Instance ("Local")

```go
import (
    "context"
    "time"

    "github.com/jinzhu/gorm"
	"github.com/mgtv-tech/jetcache-go"
    "github.com/mgtv-tech/jetcache-go/local"
)
ring := redis.NewRing(&redis.RingOptions{
    Addrs: map[string]string{
        "localhost": ":6379",
    },
})

// Create a local-only cache instance
mycache := cache.New(cache.WithName("any"),
    cache.WithLocal(local.NewTinyLFU(10000, time.Minute)),
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
    // Handle error
}
```

## Example 3: Creating a Remote-Only Cache Instance ("Remote")

```go
import (
    "context"
    "time"

    "github.com/jinzhu/gorm"
	"github.com/mgtv-tech/jetcache-go"
    "github.com/mgtv-tech/jetcache-go/remote"
    "github.com/redis/go-redis/v9"
)
ring := redis.NewRing(&redis.RingOptions{
    Addrs: map[string]string{
        "localhost": ":6379",
    },
})

// Create a remote-only cache instance
mycache := cache.New(cache.WithName("any"),
    cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
    // Handle error
}
```


## Example 4: Creating a Cache Instance with the [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) Prometheus Statistics Plugin

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

## Example 5: Creating a Cache Instance and Configuring `errNotFound` to Prevent Cache Penetration

```go
import (
    "context"
	"fmt"
    "time"

    "github.com/jinzhu/gorm"
	"github.com/mgtv-tech/jetcache-go"
)
ring := redis.NewRing(&redis.RingOptions{
    Addrs: map[string]string{
        "localhost": ":6379",
    },
})

// Create a cache instance and configure errNotFound to prevent cache penetration
mycache := cache.New(cache.WithName("any"),
	// ...
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

var value string
err := mycache.Once(ctx, key, Value(&value), Do(func(context.Context) (any, error) {
    return nil, gorm.ErrRecordNotFound
}))
fmt.Println(err)

// Output: record not found
```

`jetcache-go` uses a lightweight approach of caching null objects to address cache penetration:

- Specify the errNotFound error: When creating the cache instance, specify the error to be returned when a key is not found. Examples include gorm.ErrRecordNotFound or redis.Nil.
- Cache nil values: If a query encounters the specified errNotFound error, a placeholder value (*) is cached.
- Handle nil values: When retrieving a value, the cache checks if the retrieved value is the placeholder(*). If it is, the corresponding `errNotFound` error is returned.
