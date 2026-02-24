# jetcache-go

![banner](docs/images/banner.png)

<p>
<a href="https://github.com/mgtv-tech/jetcache-go/actions"><img src="https://github.com/mgtv-tech/jetcache-go/workflows/Go/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/mgtv-tech/jetcache-go"><img src="https://codecov.io/gh/mgtv-tech/jetcache-go/master/graph/badge.svg" alt="codeCov"></a>
<a href="https://goreportcard.com/report/github.com/mgtv-tech/jetcache-go"><img src="https://goreportcard.com/badge/github.com/mgtv-tech/jetcache-go" alt="Go Report Card"></a>
<a href="https://github.com/mgtv-tech/jetcache-go/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
<a href="https://github.com/mgtv-tech/jetcache-go/releases"><img src="https://img.shields.io/github/release/mgtv-tech/jetcache-go" alt="Release"></a>
</p>

Language: [简体中文](README_zh.md)

## Overview

`jetcache-go` is a production-grade cache framework for Go. It is inspired by Java JetCache and extends the `go-redis/cache` model with two-level caching, singleflight-based miss protection, typed batch APIs, and operational features for large-scale services.

## Why jetcache-go

- Two-level cache: local (`FreeCache`/`TinyLFU`) + remote (`Redis`)
- Singleflight miss collapse and optional auto-refresh
- Generic `MGet` with pipeline optimization
- Cache penetration protection via not-found placeholder strategy
- Built-in stats and Prometheus plugin integration
- Interface-driven design for local/remote/codec/stats extensions

## Feature Availability

- Generic `MGet` + load callback + pipeline optimization: `v1.1.0+`
- Cross-process local cache invalidation after updates: `v1.1.1+`

See [Versioning](docs/EN/Versioning.md) for details.

## Quick Start

Install:

```bash
go get github.com/mgtv-tech/jetcache-go
```

Minimal usage:

```go
package main

import (
	"context"
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/local"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithName("user-cache"),
		cache.WithLocal(local.NewTinyLFU(100_000, time.Minute)),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
	)
	defer c.Close()

	var user string
	_ = c.Once(context.Background(), "user:1001",
		cache.Value(&user),
		cache.Do(func(context.Context) (any, error) {
			return "alice", nil
		}),
	)
}
```

See full quick start and scenarios:

- [Quick Start](docs/EN/QuickStart.md)
- [Examples](docs/EN/Examples/README.md)

## Documentation

Getting started:

- [Quick Start](docs/EN/QuickStart.md)
- [Architecture](docs/EN/Architecture.md)
- [Examples](docs/EN/Examples/README.md)

Configuration and API:

- [Configuration](docs/EN/Config.md)
- [API Reference](docs/EN/CacheAPI.md)
- [Versioning](docs/EN/Versioning.md)
- [Terminology](docs/EN/Terminology.md)
- [Embedded Components](docs/EN/Embedded.md)
- [Plugin Ecosystem](docs/EN/Plugin.md)

Production operations:

- [Best Practices](docs/EN/BestPractices.md)
- [Monitoring](docs/EN/Monitoring.md)
- [Troubleshooting](docs/EN/Troubleshooting.md)
- [FAQ](docs/EN/FAQ.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).

## Contact

- Email: `daoshenzzg@gmail.com`
- Issues: <https://github.com/mgtv-tech/jetcache-go/issues>
