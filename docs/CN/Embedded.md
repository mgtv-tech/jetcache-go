<!-- TOC -->
* [介绍](#介绍)
* [多种序列化方式](#多种序列化方式)
* [多种本地缓存、远程缓存](#多种本地缓存远程缓存)
* [指标采集统计](#指标采集统计)
* [自定义接管日志](#自定义接管日志)
<!-- TOC -->

# 介绍

`jetcache-go` 通过通用接口实现了许多内嵌组件，方便开发者集成，以及参考实现自己的组件。


# 多种序列化方式

| codec方式                                           | 说明                        | 优势         |
|---------------------------------------------------|---------------------------|------------|
| native json                                       | golang自带的序列化工具            | 兼容性好       |
| [msgpack](https://github.com/vmihailenco/msgpack) | msgpack+snappy压缩（内容>64字节) | 性能较强，内存占用小 |
| [sonic](https://github.com/go-sonic/sonic)        | 字节开源的高性能json序列化工具         | 性能强        |

你也可以通过实现 `encoding.Codec` 接口来自定义自己的序列化，并通过 `encoding.RegisterCodec` 注册进来。

```go

import (
    _ "github.com/mgtv-tech/jetcache-go/encoding/yourCodec"
)

// Register your codec
encoding.RegisterCodec(yourCodec.Name)

mycache := cache.New(cache.WithName("any"),
    cache.WithRemote(...),
    cache.WithCodec(yourCodec.Name))
```


# 多种本地缓存、远程缓存

| 名称                                                  | 类型     | 特点                |
|-----------------------------------------------------|--------|-------------------|
| [ristretto](https://github.com/dgraph-io/ristretto) | Local  | 高性能、高命中率          |
| [freecache](https://github.com/coocood/freecache)   | Local  | 零垃圾收集负荷、严格限制内存使用  |
| [go-redis](https://github.com/redis/go-redis)       | Remote | 最流行的 GO Redis 客户端 |

你也可以通过实现 `remote.Remote`、`local.Local` 接口来实现自己的本地、远程缓存。

> FreeCache 使用注意事项：
>
> 缓存key的大小需要小于65535，否则无法存入到本地缓存中（The key is larger than 65535）  
> 缓存value的大小需要小于缓存总容量的1/1024，否则无法存入到本地缓存中（The entry size need less than 1/1024 of cache size）  
> 内嵌的FreeCache实例内部共享了一个 `innerCache` 实例，防止当多个缓存实例都使用 FreeCache 时内存占用过多。因此，共享 `innerCache` 会以第一次创建的配置的内存容量和过期时间为准。

# 指标采集统计

| 名称              | 类型 | 说明                                                                            |
|-----------------|----|-------------------------------------------------------------------------------|
| logStats        | 内嵌 | 默认的指标采集统计器，统计信息打印到日志                                                          |
| PrometheusStats | 插件 | [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) 提供的统计插件 |

你也可以通过实现 `stats.Handler` 接口来自定义自己的指标采集器。

示例：同时使用多种指标采集器

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

# 自定义接管日志

```go
import "github.com/mgtv-tech/jetcache-go/logger"

// Set your Logger
logger.SetDefaultLogger(l logger.Logger)
```



