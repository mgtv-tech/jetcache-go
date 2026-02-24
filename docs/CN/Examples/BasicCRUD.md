# 基础 CRUD 示例

本示例展示两级缓存下的 `Set`、`Get`、`Once`、`Delete`。

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

type User struct {
	ID   int64
	Name string
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

	c := cache.New(
		cache.WithName("user-cache"),
		cache.WithLocal(local.NewTinyLFU(100_000, time.Minute)),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
	)
	defer c.Close()

	// 1) Set
	_ = c.Set(ctx, "user:1001", cache.Value(User{ID: 1001, Name: "Ada"}), cache.TTL(30*time.Minute))

	// 2) Get
	var u User
	_ = c.Get(ctx, "user:1001", &u)
	fmt.Println("get:", u.Name)

	// 3) Once（cache-aside）
	var u2 User
	_ = c.Once(ctx, "user:1002",
		cache.Value(&u2),
		cache.Do(func(context.Context) (any, error) {
			return User{ID: 1002, Name: "Bob"}, nil
		}),
	)
	fmt.Println("once:", u2.Name)

	// 4) Delete
	_ = c.Delete(ctx, "user:1001")
}
```
