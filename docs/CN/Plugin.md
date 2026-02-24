# 插件生态

`jetcache-go-plugin` 提供由 jetcache 团队维护的可选集成组件。

- 仓库地址：<https://github.com/mgtv-tech/jetcache-go-plugin>

## 当前可用集成

| 领域 | 包路径 | 状态 | 说明 |
| --- | --- | --- | --- |
| 远程适配器 | `github.com/mgtv-tech/jetcache-go-plugin/remote` | 可用 | `go-redis/v8` 适配器。 |
| 指标统计 | `github.com/mgtv-tech/jetcache-go-plugin/stats` | 可用 | Prometheus 指标处理器。 |
| 本地缓存 | - | 使用主仓接口扩展 | 建议使用内置 `local`，或自定义 `local.Local`。 |
| 编解码 | - | 使用主仓接口扩展 | 建议使用内置 codec，或自定义 `encoding.Codec`。 |

## 安装

```bash
go get github.com/mgtv-tech/jetcache-go-plugin@latest
```

## 远程适配器：go-redis v8

当项目仍使用 `github.com/go-redis/redis/v8` 时可用。

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

如果你使用 `go-redis/v9`，请优先使用主仓内置 `remote.NewGoRedisV9Adapter(...)`。

## 指标插件：Prometheus

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

该方式会在保留日志统计的同时，把指标暴露给 Prometheus。

## 官方插件之外的扩展方式

如果你需要非官方集成，可直接实现主仓接口：

- 实现 `remote.Remote` 接入自定义远程缓存。
- 实现 `local.Local` 接入自定义本地缓存引擎。
- 实现 `encoding.Codec` 并通过 `encoding.RegisterCodec(...)` 注册。
- 实现 `stats.Handler` 接入自定义观测系统。

可配合阅读：

- [内嵌组件](Embedded.md)
- [配置项参考](Config.md)
