<!-- TOC -->
* [Cache Configuration Options](#cache-configuration-options)
* [Cache Instance Creation](#cache-instance-creation)
  * [Example 1: Creating a Two-Level Cache Instance (Both)](#example-1-creating-a-two-level-cache-instance-both)
  * [Example 2: Creating a Local-Only Cache Instance (Local)](#example-2-creating-a-local-only-cache-instance-local)
  * [Example 3: Creating a Remote-Only Cache Instance (Remote)](#example-3-creating-a-remote-only-cache-instance-remote)
  * [Example 4: Creating a Cache Instance and Configuring the jetcache-go-plugin Prometheus Statistics Plugin](#example-4-creating-a-cache-instance-and-configuring-the-jetcache-go-plugin-prometheus-statistics-plugin)
  * [Example 5: Creating a Cache Instance and Configuring `errNotFound` to Prevent Cache Penetration](#example-5-creating-a-cache-instance-and-configuring-errnotfound-to-prevent-cache-penetration)
<!-- TOC -->

# Cache Configuration Options

| Configuration Item         | Data Type                 | Default Value              | Description                                                                                                                                                                                                                                   |
|----------------------------|---------------------------|----------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| name                       | string                    | default                    | Cache name, used for log identification and metrics reporting.                                                                                                                                                                                |
| remote                     | `remote.Remote` interface | nil                        | Distributed cache, such as Redis.  Can be customized by implementing the `remote.Remote` interface.                                                                                                                                           |
| local                      | `local.Local` interface   | nil                        | In-memory cache, such as FreeCache, TinyLFU. Can be customized by implementing the `local.Local` interface.                                                                                                                                   |
| codec                      | string                    | msgpack                    | Encoding and decoding method for values. Defaults to "msgpack". Options: `json` \| `msgpack` \| `sonic`. Can be customized by implementing the `encoding.Codec` interface and registering it.                                                 |
| errNotFound                | error                     | nil                        | Error returned when an origin record is not found, e.g., `gorm.ErrRecordNotFound`. Used to prevent cache penetration (i.e., caching empty objects).                                                                                           |
| remoteExpiry               | `time.Duration`           | 1 hour                     | Remote cache TTL, defaults to 1 hour.                                                                                                                                                                                                         |
| notFoundExpiry             | `time.Duration`           | 1 minute                   | Expiration time for placeholder caches when a cache miss occurs. Defaults to 1 minute.                                                                                                                                                        |
| offset                     | `time.Duration`           | (0,10] seconds             | Expiration time jitter factor for cache misses.                                                                                                                                                                                               |
| refreshDuration            | `time.Duration`           | 0                          | Interval for asynchronous cache refresh. Defaults to 0 (refresh disabled).                                                                                                                                                                    |
| stopRefreshAfterLastAccess | `time.Duration`           | refreshDuration + 1 second | Duration before cache refresh stops (after last access).                                                                                                                                                                                      |
| refreshConcurrency         | int                       | 4                          | Maximum number of concurrent refreshes in the cache refresh task pool.                                                                                                                                                                        |
| statsDisabled              | bool                      | false                      | Flag to disable cache statistics.                                                                                                                                                                                                             |
| statsHandler               | `stats.Handler` interface | stats.NewStatsLogger       | Metrics collector.  Defaults to an embedded `log` collector.  Can use the [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) `Prometheus` plugin or a custom implementation that implements the `stats.Handler` interface. |
| sourceID                   | string                    | 16-character random string | 【Cache Event Broadcasting】Unique identifier for the cache instance.                                                                                                                                                                           |
| syncLocal                  | bool                      | false                      | 【Cache Event Broadcasting】Enable events to synchronize local caches (only applicable to "Both" cache types).                                                                                                                                  |
| eventChBufSize             | int                       | 100                        | 【Cache Event Broadcasting】Buffer size of the event channel (defaults to 100).                                                                                                                                                                 |
| eventHandler               | `func(event *Event)`      | nil                        | 【Cache Event Broadcasting】Function to handle local cache invalidation events.                                                                                                                                                                 |
| separatorDisabled          | bool                      | false                      | Disable the cache key separator. Defaults to false. If true, the cache key will not use a separator. Currently mainly used for concatenating cache keys and IDs in generic interfaces.                                                        |
| separator                  | string                    | :                          | Cache key separator. Defaults to ":". Currently mainly used for concatenating cache keys and IDs in generic interfaces.                                                                                                                       |


# Cache Instance Creation

## Example 1: Creating a Two-Level Cache Instance (Both)

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
	cache.WithLocal(local.NewTinyLFU(10000, time.Minute)), // Local cache expiration time is uniformly set to 1 minute
	cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
	Name string
	Age  int
}{Name: "John Doe", Age: 30}

// Set cache, where the remote cache expiration time TTL is 1 hour
err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
	// Error handling
}
```

## Example 2: Creating a Local-Only Cache Instance (Local)

```go
import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/local"
)

// Create a local-only cache instance
mycache := cache.New(cache.WithName("any"),
	cache.WithLocal(local.NewTinyLFU(10000, time.Minute)),
	cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
	Name string
	Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj))
if err != nil {
	// Error handling
}
```

## Example 3: Creating a Remote-Only Cache Instance (Remote)

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
	// Error handling
}
```

## Example 4: Creating a Cache Instance and Configuring the [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) Prometheus Statistics Plugin

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

> Example 4 integrates both `Log` and `Prometheus` statistics.  See: [Stat](/docs/EN/Stat.md)


## Example 5: Creating a Cache Instance and Configuring `errNotFound` to Prevent Cache Penetration

```go
import (
    "context"
    "fmt"
    "time"
  
    "github.com/jinzhu/gorm"
    "github.com/mgtv-tech/jetcache-go"
    "github.com/redis/go-redis/v9"
)

ring := redis.NewRing(&redis.RingOptions{
    Addrs: map[string]string{
        "localhost": ":6379",
    },
})

// Create a cache instance and configure errNotFound to prevent cache penetration
mycache := cache.New(cache.WithName("any"),
    cache.WithRemote(remote.NewGoRedisV9Adapter(ring)), // Assuming you still want a remote cache
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

var value string
err := mycache.Once(ctx, key, cache.Value(&value), cache.Do(func(context.Context) (any, error) {
    return nil, gorm.ErrRecordNotFound
}))
fmt.Println(err)

// Output: record not found
```

`jetcache-go` uses a lightweight approach of [caching null objects] to address cache penetration:

- When creating a cache instance, specify a "not found" error. For example: `gorm.ErrRecordNotFound`, `redis.Nil`.
- If a "not found" error is encountered during a query, a placeholder value (e.g., a special marker) is cached.
- When retrieving the value, check if it's the placeholder. If so, return the corresponding "not found" error.
