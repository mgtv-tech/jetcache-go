# Embedded Components Guide

`jetcache-go` is interface-driven. You can use built-in components or replace each part with your own implementation.

## Component Map

| Layer | Interface | Built-in Choices |
| --- | --- | --- |
| Local cache | `local.Local` | `local.NewTinyLFU`, `local.NewFreeCache` |
| Remote cache | `remote.Remote` | `remote.NewGoRedisV9Adapter` |
| Codec | `encoding.Codec` | `msgpack` (default), `json`, `sonic` |
| Metrics | `stats.Handler` | `stats.NewStatsLogger`, multi-handler chain |
| Logging | `logger.Logger` | default logger, replaceable |

## Local Cache

`local.Local`:

```go
type Local interface {
	Set(key string, data []byte)
	Get(key string) ([]byte, bool)
	Del(key string)
}
```

Built-in local implementations:

- `local.NewTinyLFU(size, ttl)`
- `local.NewFreeCache(size, ttl, innerKeyPrefix...)`

### TinyLFU notes

- Based on Ristretto.
- Good default for high hit-ratio workloads.
- Uses TTL with optional random offset internally.

### FreeCache notes

- Strict memory boundaries.
- Shared internal cache instance in process (`once.Do(...)`).
- Practical constraints from FreeCache:
  - key length must be less than 65535 bytes.
  - value size must be smaller than 1/1024 of total cache size.

## Remote Cache

`remote.Remote`:

```go
type Remote interface {
	SetEX(ctx context.Context, key string, value any, expire time.Duration) error
	SetNX(ctx context.Context, key string, value any, expire time.Duration) (bool, error)
	SetXX(ctx context.Context, key string, value any, expire time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) (int64, error)
	MGet(ctx context.Context, keys ...string) (map[string]any, error)
	MSet(ctx context.Context, value map[string]any, expire time.Duration) error
	Nil() error
}
```

Built-in adapter (runnable):

```go
package main

import (
	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)))
	defer c.Close()
}
```

## Codec

`encoding.Codec`:

```go
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
	Name() string
}
```

Built-in codecs are imported by default init side effects:

- `msgpack` (default)
- `json`
- `sonic`

Choose codec (runnable):

```go
package main

import (
	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithCodec("json"),
	)
	defer c.Close()
}
```

Register custom codec (runnable):

```go
package main

import (
	stdjson "encoding/json"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/encoding"
)

type myCodec struct{}

func (myCodec) Marshal(v any) ([]byte, error)   { return stdjson.Marshal(v) }
func (myCodec) Unmarshal(b []byte, v any) error { return stdjson.Unmarshal(b, v) }
func (myCodec) Name() string                    { return "my-json" }

func main() {
	encoding.RegisterCodec(myCodec{})
	c := cache.New(cache.WithCodec("my-json"))
	defer c.Close()
}
```

## Stats Handler

`stats.Handler`:

```go
type Handler interface {
	IncrHit()
	IncrMiss()
	IncrLocalHit()
	IncrLocalMiss()
	IncrRemoteHit()
	IncrRemoteMiss()
	IncrQuery()
	IncrQueryFail(err error)
}
```

Compose multiple handlers (runnable):

```go
package main

import (
	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/mgtv-tech/jetcache-go/stats"
	pstats "github.com/mgtv-tech/jetcache-go-plugin/stats"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	cacheName := "profile"
	c := cache.New(
		cache.WithName(cacheName),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithStatsHandler(
			stats.NewHandles(false,
				stats.NewStatsLogger(cacheName),
				pstats.NewPrometheus(cacheName),
			),
		),
	)
	defer c.Close()
}
```

## Logger

Replace global logger implementation (runnable):

```go
package main

import (
	"log"

	"github.com/mgtv-tech/jetcache-go/logger"
)

type stdLogger struct{}

func (stdLogger) Debug(format string, v ...any) { log.Printf("[DEBUG] "+format, v...) }
func (stdLogger) Info(format string, v ...any)  { log.Printf("[INFO] "+format, v...) }
func (stdLogger) Warn(format string, v ...any)  { log.Printf("[WARN] "+format, v...) }
func (stdLogger) Error(format string, v ...any) { log.Printf("[ERROR] "+format, v...) }

func main() {
	logger.SetDefaultLogger(stdLogger{})
}
```

## Custom Component Checklist

- Keep implementations thread-safe.
- Respect context cancellation and timeout in remote methods.
- Return deterministic `Nil()` error for remote "key not found" semantics.
- Avoid allocations in hot path (`Get`, `MGet`) where possible.
- Add unit tests for edge cases: empty value, TTL, timeout, serialization failure.
