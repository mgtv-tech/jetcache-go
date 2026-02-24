# 5分钟快速上手

本指南用五个步骤覆盖 `jetcache-go` 核心能力。

## 准备

- Go 1.20+
- Redis（远程/两级缓存示例需要）

安装：

```bash
go get github.com/mgtv-tech/jetcache-go
```

## 第一步：本地缓存

```go
package main

import (
	"context"
	"fmt"
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/local"
)

func main() {
	c := cache.New(
		cache.WithName("demo-local"),
		cache.WithLocal(local.NewTinyLFU(100_000, time.Minute)),
	)
	defer c.Close()

	ctx := context.Background()
	_ = c.Set(ctx, "name", cache.Value("jetcache-go"))

	var name string
	_ = c.Get(ctx, "name", &name)
	fmt.Println(name)
}
```

## 第二步：远程缓存（Redis）

```go
package main

import (
	"context"
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithName("demo-remote"),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithRemoteExpiry(time.Hour),
	)
	defer c.Close()

	_ = c.Set(context.Background(), "user:1001", cache.Value("alice"), cache.TTL(time.Hour))
}
```

## 第三步：两级缓存 + `Once`

```go
package main

import (
	"context"
	"fmt"
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/local"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithName("demo-both"),
		cache.WithLocal(local.NewTinyLFU(100_000, time.Minute)),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
	)
	defer c.Close()

	var user string
	err := c.Once(context.Background(), "user:1001",
		cache.Value(&user),
		cache.Do(func(context.Context) (any, error) {
			fmt.Println("load from DB")
			return "alice", nil
		}),
	)
	if err != nil {
		panic(err)
	}
}
```

## 第四步：防缓存穿透

```go
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithErrNotFound(sql.ErrNoRows),
		cache.WithNotFoundExpiry(30*time.Second),
	)
	defer c.Close()

	var out string
	err := c.Once(context.Background(), "user:404",
		cache.Value(&out),
		cache.Do(func(context.Context) (any, error) {
			return nil, sql.ErrNoRows
		}),
	)
	fmt.Println(errors.Is(err, sql.ErrNoRows))
}
```

## 第五步：热点 Key 自动刷新

```go
package main

import (
	"context"
	"time"

	cache "github.com/mgtv-tech/jetcache-go"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	c := cache.New(
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithRefreshDuration(time.Minute),
		cache.WithStopRefreshAfterLastAccess(10*time.Minute),
	)
	defer c.Close()

	var value string
	_ = c.Once(context.Background(), "hot:key",
		cache.Value(&value),
		cache.Refresh(true),
		cache.Do(func(context.Context) (any, error) {
			return "latest", nil
		}),
	)
}
```

## 下一步

- [架构设计](Architecture.md)
- [配置项参考](Config.md)
- [API 参考](CacheAPI.md)
- [版本与功能可用性](Versioning.md)
- [场景示例](Examples/README.md)
- [监控与可观测性](Monitoring.md)
- [最佳实践](BestPractices.md)
