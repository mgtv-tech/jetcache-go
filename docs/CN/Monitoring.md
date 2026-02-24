# 监控与可观测性

本文档说明如何在生产环境监控 `jetcache-go`。

## 监控目标

至少持续跟踪：

- 总体命中率，
- 本地与远程命中结构，
- query fail 趋势，
- 缓存层流量规模。

## 内置日志统计

默认可通过 `stats.NewStatsLogger(...)` 周期输出统计日志。

可运行示例：

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

自定义统计周期：

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

说明：`stats.NewStatsLogger(...)` 在进程内使用单例 ticker。一个进程中首次配置的统计周期会成为该进程的实际周期。

日志列含义：

- `qpm`：统计窗口内请求量，
- `hit_ratio`：命中率，
- `hit`、`miss`，
- `query`、`query_fail`。

在可用时还会输出 `_local`、`_remote` 子行。

## Prometheus 集成

使用 `jetcache-go-plugin` 的统计处理器。

可运行示例：

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

这样可以同时获得：

- 本地日志调试能力，
- Prometheus 抓取与 Grafana 看板能力。

## 推荐看板面板

- 总请求吞吐，
- 全局命中率，
- 本地命中率，
- 远程命中率，
- query fail 速率，
- miss 量最高的 cache 名称。

建议同时保留 5 分钟趋势视图和 1 分钟突发视图。

## 推荐告警规则

- 严重：命中率连续 N 分钟低于阈值。
- 警告：query fail 持续高于基线。
- 警告：remote miss 上升且 local hit 下降。

告警阈值建议基于业务基线，不建议全局固定。

## 自动刷新相关监控

启用自动刷新（`WithRefreshDuration` + `Refresh(true)`）后建议增加：

- `TaskSize()` 趋势监控，
- 开启刷新后 query 负载变化，
- key 过期窗口是否触发后端 QPS 峰值。

## 上线检查清单

- 每个缓存实例显式设置 `WithName(...)`。
- 生产环境至少保留一个统计处理器（默认统计链或自定义 `WithStatsHandler(...)`）。
- 发布前验证 Prometheus 抓取与标签。
- 为核心缓存命名空间准备对应 runbook。

## 范围说明

本页是 `jetcache-go` 监控文档的主入口。
