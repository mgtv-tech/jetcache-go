<!-- TOC -->
* [Cache 配置项说明](#cache-配置项说明)
* [Cache 缓存实例创建](#cache-缓存实例创建)
  * [示例1：创建二级缓存实例（Both）](#示例1创建二级缓存实例both)
  * [示例2：创建仅本地缓存实例（Local）](#示例2创建仅本地缓存实例local)
  * [示例3：创建仅远程缓存实例（Remote）](#示例3创建仅远程缓存实例remote)
  * [示例4：创建缓存实例，并配置jetcache-go-plugin Prometheus 统计插件](#示例4创建缓存实例并配置jetcache-go-plugin-prometheus-统计插件)
  * [示例5：创建缓存实例，并配置 `errNotFound` 防止缓存穿透](#示例5创建缓存实例并配置-errnotfound-防止缓存穿透)
<!-- TOC -->

# Cache 配置项说明

| 配置项名称                      | 配置项类型                | 缺省值                  | 说明                                                                                                                                                |
|----------------------------|----------------------|----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------|
| name                       | string               | default              | 缓存名称，用于日志标识和指标报告                                                                                                                                  |
| remote                     | `remote.Remote` 接口   | nil                  | remote 是分布式缓存，例如 Redis。也可以自定义，实现`remote.Remote`接口即可                                                                                               |
| local                      | `local.Local` 接口     | nil                  | local 是内存缓存，例如 FreeCache、TinyLFU。也可以自定义，实现`local.Local`接口即可                                                                                       |
| codec                      | string               | msgpack              | value的编码和解码方法。默认为 "msgpack"。可选：`json` \| `msgpack` \| `sonic`，也可以自定义，实现`encoding.Codec`接口并注册即可                                                    | 
| errNotFound                | error                | nil                  | 缓存未命中时返回的错误，例：`gorm.ErrRecordNotFound`。用于防止缓存穿透（即缓存空对象）                                                                                           |
| remoteExpiry               | `time.Duration`      | 1小时                  | 远程缓存 TTL，默认为 1 小时                                                                                                                                 |
| notFoundExpiry             | `time.Duration`      | 1分钟                  | 缓存未命中时占位符缓存的过期时间。默认为 1 分钟                                                                                                                         |
| offset                     | `time.Duration`      | (0,10]秒              | 缓存未命中时的过期时间抖动因子                                                                                                                                   |
| refreshDuration            | `time.Duration`      | 0                    | 异步缓存刷新的间隔。默认为 0（禁用刷新）                                                                                                                             |
| stopRefreshAfterLastAccess | `time.Duration`      | refreshDuration + 1秒 | 缓存停止刷新之前的持续时间（上次访问后）                                                                                                                              |
| refreshConcurrency         | int                  | 4                    | 刷新缓存任务池的并发刷新的最大数量                                                                                                                                 |
| statsDisabled              | bool                 | false                | 禁用缓存统计的标志                                                                                                                                         |
| statsHandler               | `stats.Handler` 接口   | stats.NewStatsLogger | 指标统计收集器。默认内嵌实现了`log`统计，也可以使用[jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) 的`Prometheus` 插件。或自定义实现，只要实现`stats.Handler`接口即可 |
| sourceID                   | string               | 16位随机字符串             | 【缓存事件广播】缓存实例的唯一标识符                                                                                                                                |
| syncLocal                  | bool                 | false                | 【缓存事件广播】启用同步本地缓存的事件（仅适用于 "Both" 缓存类型）                                                                                                             |
| eventChBufSize             | int                  | 100                  | 【缓存事件广播】事件通道的缓冲区大小（默认为 100）                                                                                                                       |
| eventHandler               | `func(event *Event)` | nil                  | 【缓存事件广播】处理本地缓存失效事件的函数                                                                                                                             |
| separatorDisabled          | bool                 | false                | 禁用缓存键的分隔符。默认为false。如果为true，则缓存键不会使用分隔符。目前主要用于泛型接口的缓存key和ID拼接                                                                                      |
| separator                  | string               | :                    | 缓存键的分隔符。默认为 ":"。目前主要用于泛型接口的缓存key和ID拼接                                                                                                             |

# Cache 缓存实例创建

## 示例1：创建二级缓存实例（Both）

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

// 创建二级缓存实例
mycache := cache.New(cache.WithName("any"),
    cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
    cache.WithLocal(local.NewTinyLFU(10000, time.Minute)), // 本地缓存过期时间统一为 1 分钟
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}
// 设置缓存，其中远程缓存过期时间 TTL 为 1 小时
err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
    // 错误处理
}
```

## 示例2：创建仅本地缓存实例（Local）

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

// 创建仅本地缓存实例
mycache := cache.New(cache.WithName("any"),
    cache.WithLocal(local.NewTinyLFU(10000, time.Minute)),
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj))
if err != nil {
    // 错误处理
}
```

## 示例3：创建仅远程缓存实例（Remote）

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

// 创建仅远程缓存实例
mycache := cache.New(cache.WithName("any"),
    cache.WithRemote(remote.NewGoRedisV9Adapter(ring)),
    cache.WithErrNotFound(gorm.ErrRecordNotFound))

obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}

err := mycache.Set(context.Background(), "mykey", cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
    // 错误处理
}
```

## 示例4：创建缓存实例，并配置[jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) Prometheus 统计插件

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
    // 错误处理
}
```
> 示例4 同时集成了 `Log` 和 `Prometheus` 统计。效果见：[Stat](/docs/CN/Stat.md)

## 示例5：创建缓存实例，并配置 `errNotFound` 防止缓存穿透

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

// 创建缓存实例，并配置 errNotFound 防止缓存穿透
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

`jetcache-go` 采取轻量级的 \[缓存空对象\] 方式来解决缓存穿透问题：

- 创建cache实例时，指定未找到错误。例如：gorm.ErrRecordNotFound、redis.Nil
- 查询如果遇到未找到错误，直接用*号作为缓存值缓存
- 返回的时候，判断缓存值是否为*号，如果是，则返回对应的未找到错误
