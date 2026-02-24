# 5-Minute Quick Start

This guide covers core `jetcache-go` usage in five steps.

## Prerequisites

- Go 1.20+
- Redis (for remote/both examples)

Install:

```bash
go get github.com/mgtv-tech/jetcache-go
```

## Step 1: Local Cache

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

## Step 2: Remote Cache (Redis)

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

## Step 3: Two-Level Cache + `Once`

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

## Step 4: Prevent Cache Penetration

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

## Step 5: Auto-Refresh for Hot Keys

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

## Next

- [Architecture](Architecture.md)
- [Config](Config.md)
- [CacheAPI](CacheAPI.md)
- [Versioning](Versioning.md)
- [Examples](Examples/README.md)
- [Monitoring](Monitoring.md)
- [BestPractices](BestPractices.md)
