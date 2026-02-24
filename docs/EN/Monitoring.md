# Monitoring and Observability

This guide describes how to monitor `jetcache-go` in production.

## Monitoring Goals

Track at least:

- overall hit ratio,
- local vs remote hit structure,
- query fail trend,
- cache layer traffic volume.

## Built-in Log Stats

By default, jetcache can emit periodic stats logs via `stats.NewStatsLogger(...)`.

Runnable example:

```go
package main

import (
	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/mgtv-tech/jetcache-go/stats"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithName("order-cache"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithStatsHandler(stats.NewStatsLogger("order-cache")),
	)
	defer c.Close()
}
```

Set custom log interval:

```go
package main

import (
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/mgtv-tech/jetcache-go/stats"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithName("order-cache"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithStatsHandler(stats.NewStatsLogger("order-cache", stats.WithStatsInterval(30*time.Second))),
	)
	defer c.Close()
}
```

Note: `stats.NewStatsLogger(...)` uses a process-level singleton ticker. The first configured interval in a process is the effective interval for that process.

Output columns:

- `qpm`: requests in the latest interval,
- `hit_ratio`: hit/(hit+miss),
- `hit`, `miss`,
- `query`, `query_fail`.

The logger also prints `_local` and `_remote` rows when available.

## Prometheus Integration

Use plugin handler from `jetcache-go-plugin`.

Runnable example:

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
	defer c.Close()
}
```

This allows:

- local debug with log stats,
- centralized Prometheus scrape and Grafana dashboards.

## Recommended Dashboard Panels

- total request throughput,
- global hit ratio,
- local hit ratio,
- remote hit ratio,
- query fail rate,
- top cache names by miss volume.

Use a 5m view for stable trend and 1m view for burst diagnosis.

## Suggested Alert Rules

- Critical: hit ratio drops below threshold for N minutes.
- Warning: query fail continuously above baseline.
- Warning: remote miss grows while local hit declines.

Thresholds should be calibrated by business baseline, not fixed globally.

## Monitoring for Auto-Refresh

When refresh is enabled (`WithRefreshDuration` + `Refresh(true)`):

- watch `TaskSize()` trend,
- track query load change after refresh rollout,
- verify backend QPS does not spike on key expiration windows.

## Rollout Checklist

- Ensure every cache instance has explicit `WithName(...)`.
- Keep at least one stats handler enabled in production (`WithStatsHandler(...)` or default stats chain).
- Validate Prometheus scrape and labels before release.
- Keep one runbook entry per major cache namespace.

## Scope

This page is the monitoring source of truth for `jetcache-go`.
