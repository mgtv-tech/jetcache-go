<p align="center">
<a href="https://github.com/daoshenzzg/jetcache-go/actions"><img src="https://github.com/daoshenzzg/jetcache-go/workflows/Go/badge.svg" alt="Build Status"></a>
<a href="https://codecov.io/gh/daoshenzzg/jetcache-go"><img src="https://codecov.io/gh/daoshenzzg/jetcache-go/master/graph/badge.svg" alt="codeCov"></a>
<a href="https://github.com/daoshenzzg/jetcache-go/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
</p>

Translations: [English](README.md) | [简体中文](README_zh.md)

# 介绍
[jetcache-go](https://github.com/daoshenzzg/jetcache-go)是基于[go-redis/cache](https://github.com/go-redis/cache)拓展的通用缓存访问框架。
实现了类似Java版[JetCache](https://github.com/alibaba/jetcache)的核心功能，包括：

- ✅ 二级缓存自由组合：本地缓存、集中缓存、本地缓存+集中缓存
- ✅ Once接口采用单飞(`singleflight`)模式，高并发且线程安全
- ✅ 默认采用[MsgPack](https://github.com/vmihailenco/msgpack)来编解码Value，你也可以自定义实现
- ✅ 本地缓存默认实现了[TinyLFU](https://github.com/dgryski/go-tinylfu)和[FreeCache](https://github.com/coocood/freecache)，你也可以自定义实现
- ✅ 集中缓存默认实现了[go-redis/v8](https://github.com/redis/go-redis)的适配器，你也可以自定义实现
- ✅ 可以自定义`errNotFound`，通过占位符替换，缓存空结果防止缓存穿透
- ✅ 支持开启分布式缓存异步刷新
- ✅ 指标采集，默认实现了通过日志打印各级缓存的统计指标（QPM、Hit、Miss、Query、QueryFail），你也可以自定义实现

# 安装
使用最新版本的jetcache-go，您可以在项目中导入该库：
```shell
go get https://github.com/daoshenzzg/jetcache-go
```


## 快速开始

### 
```go
package cache_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/jetcache-go"
	"github.com/jetcache-go/local"
	"github.com/jetcache-go/logger"
	"github.com/jetcache-go/remote"
	"github.com/jetcache-go/util"
)

var errRecordNotFound = errors.New("mock gorm.errRecordNotFound")

type object struct {
	Str string
	Num int
}

func mockDBGetObject(id int) (*object, error) {
	if id > 100 {
		return nil, errRecordNotFound
	}
	return &object{Str: "mystring", Num: 42}, nil
}

func Example_basicUsage() {
	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": ":6379",
			"server2": ":6380",
		},
	})

	mycache := cache.New("basicUsage",
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound))

	ctx := context.TODO()
	key := util.JoinAny(":", "mykey", 1)
	obj, _ := mockDBGetObject(1)

	if err := mycache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: obj,
		TTL:   time.Hour,
	}); err != nil {
		panic(err)
	}

	var wanted object
	if err := mycache.Get(ctx, key, &wanted); err == nil {
		fmt.Println(wanted)
	}
	// Output: {mystring 42}

	mycache.Close()
}

func Example_advancedUsage() {
	logger.SetLevel(logger.LevelInfo)

	ring := redis.NewRing(&redis.RingOptions{
		Addrs: map[string]string{
			"server1": ":6379",
			"server2": ":6380",
		},
	})

	mycache := cache.New("advancedUsage",
		cache.WithRemote(remote.NewGoRedisV8Adaptor(ring)),
		cache.WithLocal(local.NewFreeCache(256*local.MB, time.Minute)),
		cache.WithErrNotFound(errRecordNotFound),
		cache.WithRefreshDuration(time.Minute))

	obj := new(object)
	err := mycache.Once(&cache.Item{
		Key:   util.JoinAny(":", "mykey", 1),
		Value: obj, // destination
		Do: func(*cache.Item) (interface{}, error) {
			return mockDBGetObject(1)
		},
		Refresh: true, // auto refreshment
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(obj)
	//Output: &{mystring 42}

	mycache.Close()
}
```

### Configure settings
```go
// Options are used to store cache options.
type Options struct {
    remote                     remote.Remote // Remote cache.
    local                      local.Local   // Local cache.
    codec                      string        // Value encoding and decoding method. Default is "json.Name" or "msgpack.Name". You can also customize it.
    errNotFound                error         // Error to return for cache miss. Used to prevent cache penetration.
    notFoundExpiry             time.Duration // Duration for placeholder cache when there is a cache miss. Default is 1 minute.
    refreshDuration            time.Duration // Interval for asynchronous cache refresh. Default is 0 (refresh is disabled).
    stopRefreshAfterLastAccess time.Duration // Duration for cache to stop refreshing after no access. Default is refreshDuration + 1 second.
    refreshConcurrency         int           // Maximum number of concurrent cache refreshes. Default is 4.
    statsDisabled              bool          // Flag to disable cache statistics.
    statsHandler               stats.Handler // Metrics statsHandler collector.
}
```

### 缓存指标收集和统计
您可以实现`stats.Handler`接口并注册到Cache组件。我们默认实现了通过日志打印统计指标，如下所示：
```shell
2023/09/11 16:42:30.695294 statslogger.go:178: [INFO] jetcache-go stats last 1 minute.
cache       |         qpm|   hit_ratio|         hit|        miss|       query|  query_fail
------------+------------+------------+------------+------------+------------+------------
bench       |   216440123|     100.00%|   216439867|         256|         256|           0|
bench_local |   216440123|     100.00%|   216434970|        5153|           -|           -|
bench_remote|        5153|      95.03%|        4897|         256|           -|           -|
------------+------------+------------+------------+------------+------------+------------
```

