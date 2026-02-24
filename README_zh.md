# jetcache-go

![banner](docs/images/banner.png)

<p>
<a href="https://github.com/mgtv-tech/jetcache-go/actions"><img src="https://github.com/mgtv-tech/jetcache-go/workflows/Go/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/mgtv-tech/jetcache-go"><img src="https://codecov.io/gh/mgtv-tech/jetcache-go/master/graph/badge.svg" alt="codeCov"></a>
<a href="https://goreportcard.com/report/github.com/mgtv-tech/jetcache-go"><img src="https://goreportcard.com/badge/github.com/mgtv-tech/jetcache-go" alt="Go Report Card"></a>
<a href="https://github.com/mgtv-tech/jetcache-go/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
<a href="https://github.com/mgtv-tech/jetcache-go/releases"><img src="https://img.shields.io/github/release/mgtv-tech/jetcache-go" alt="Release"></a>
</p>

语言： [English](README.md)

## 项目简介

`jetcache-go` 是面向生产环境的 Go 缓存框架。它参考 Java JetCache 思路，并在 `go-redis/cache` 模型上扩展了两级缓存、singleflight 防击穿、泛型批量查询和运维能力。

## 核心能力

- 两级缓存：本地（`FreeCache`/`TinyLFU`）+ 远程（`Redis`）
- singleflight 合并回源，可选自动刷新
- 泛型 `MGet` 与 pipeline 优化
- 通过 not-found 占位符策略防缓存穿透
- 内置统计与 Prometheus 插件集成
- 基于接口设计，便于扩展 local/remote/codec/stats

## 功能版本说明

- 泛型 `MGet` + 回源函数 + pipeline 优化：`v1.1.0+`
- 更新后跨进程本地缓存失效：`v1.1.1+`

详见 [版本与功能可用性](docs/CN/Versioning.md)。

## 快速开始

安装：

```bash
go get github.com/mgtv-tech/jetcache-go
```

最小示例：

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

查看更多：

- [快速上手](docs/CN/QuickStart.md)
- [场景示例](docs/CN/Examples/README.md)

## 文档导航

入门：

- [快速上手](docs/CN/QuickStart.md)
- [架构设计](docs/CN/Architecture.md)
- [场景示例](docs/CN/Examples/README.md)

配置与 API：

- [配置项参考](docs/CN/Config.md)
- [API 参考](docs/CN/CacheAPI.md)
- [版本与功能可用性](docs/CN/Versioning.md)
- [术语说明](docs/CN/Terminology.md)
- [内嵌组件](docs/CN/Embedded.md)
- [插件生态](docs/CN/Plugin.md)

生产实践：

- [最佳实践](docs/CN/BestPractices.md)
- [监控与可观测性](docs/CN/Monitoring.md)
- [故障排查](docs/CN/Troubleshooting.md)
- [常见问题](docs/CN/FAQ.md)

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 许可证

MIT，见 [许可证](LICENSE)。

## 联系方式

- 邮箱：`daoshenzzg@gmail.com`
- Issues：<https://github.com/mgtv-tech/jetcache-go/issues>
