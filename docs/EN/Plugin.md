# Plugin Ecosystem

`jetcache-go-plugin` provides optional integrations maintained by the jetcache team.

- Repository: <https://github.com/mgtv-tech/jetcache-go-plugin>

## Available Integrations

| Area | Package | Status | Description |
| --- | --- | --- | --- |
| Remote adapter | `github.com/mgtv-tech/jetcache-go-plugin/remote` | Available | Redis adapter for `go-redis/v8`. |
| Stats | `github.com/mgtv-tech/jetcache-go-plugin/stats` | Available | Prometheus metrics handler. |
| Local cache | - | Use built-in interfaces | Prefer built-in `local` package or custom `local.Local`. |
| Codec | - | Use built-in interfaces | Prefer built-in codec package or custom `encoding.Codec`. |

## Install

```bash
go get github.com/mgtv-tech/jetcache-go-plugin@latest
```

## Remote Adapter: go-redis v8

Use this when your project is still on `github.com/go-redis/redis/v8`.

```go
import (
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/local"
	premote "github.com/mgtv-tech/jetcache-go-plugin/remote"
	"github.com/go-redis/redis/v8"
)

rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
c := cache.New(
	cache.WithName("demo"),
	cache.WithRemote(premote.NewGoRedisV8Adapter(rdb)),
	cache.WithLocal(local.NewTinyLFU(50_000, time.Minute)),
)
```

If you use `go-redis/v9`, use built-in `remote.NewGoRedisV9Adapter(...)` from this repository.

## Stats Plugin: Prometheus

```go
import (
	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/mgtv-tech/jetcache-go/stats"
	pstats "github.com/mgtv-tech/jetcache-go-plugin/stats"
)

cacheName := "order-cache"
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
```

This setup exports metrics to Prometheus while keeping built-in log stats.

## Extending Beyond Official Plugins

When you need non-official integrations:

- Implement `remote.Remote` for custom remote stores.
- Implement `local.Local` for custom local cache engines.
- Implement `encoding.Codec` and register with `encoding.RegisterCodec(...)`.
- Implement `stats.Handler` for custom observability backend.

See:

- [Embedded Components](Embedded.md)
- [Configuration](Config.md)
