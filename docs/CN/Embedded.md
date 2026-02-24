# 内嵌组件指南

`jetcache-go` 采用接口驱动设计。你既可以使用内置组件，也可以替换为自定义实现。

## 组件总览

| 层级 | 接口 | 内置实现 |
| --- | --- | --- |
| 本地缓存 | `local.Local` | `local.NewTinyLFU`、`local.NewFreeCache` |
| 远程缓存 | `remote.Remote` | `remote.NewGoRedisV9Adapter` |
| 编解码 | `encoding.Codec` | `msgpack`（默认）、`json`、`sonic` |
| 指标统计 | `stats.Handler` | `stats.NewStatsLogger`、多处理器组合 |
| 日志 | `logger.Logger` | 默认实现，可替换 |

## 本地缓存

`local.Local` 接口：

```go
type Local interface {
	Set(key string, data []byte)
	Get(key string) ([]byte, bool)
	Del(key string)
}
```

内置本地缓存实现：

- `local.NewTinyLFU(size, ttl)`
- `local.NewFreeCache(size, ttl, innerKeyPrefix...)`

### TinyLFU 说明

- 基于 Ristretto。
- 适合高命中率场景，通常可作为默认选择。
- 内部支持 TTL 与随机抖动。

### FreeCache 说明

- 内存边界严格。
- 进程内多个实例共享一个底层 `innerCache`（`once.Do(...)`）。
- FreeCache 限制：
  - key 长度必须小于 65535 字节。
  - value 不能超过缓存总大小的 1/1024。

## 远程缓存

`remote.Remote` 接口：

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

内置远程适配器（可运行）：

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

## 编解码

`encoding.Codec` 接口：

```go
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
	Name() string
}
```

默认已注册 codec：

- `msgpack`（默认）
- `json`
- `sonic`

选择 codec（可运行）：

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

注册自定义 codec（可运行）：

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

## 指标统计

`stats.Handler` 接口：

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

多处理器组合（可运行）：

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

## 日志

替换全局日志实现（可运行）：

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

## 自定义组件检查清单

- 实现需线程安全。
- 远程接口要正确处理 context 超时/取消。
- `Nil()` 必须稳定表示“key 不存在”。
- 尽量减少热点路径（`Get`、`MGet`）中的额外分配。
- 增加边界测试：空值、TTL、超时、序列化失败。
