# 微服务模式示例

本示例展示微服务常见缓存模式：

- 两级缓存，
- cache-aside 读路径，
- 写后删除失效，
- 可选跨节点本地缓存失效事件。

版本说明：跨进程本地失效同步钩子（`WithSyncLocal`）从 `v1.1.1+` 开始可用。

```go
package main

import (
	"context"
	"encoding/json"
	"strconv"
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

func buildCache(rdb *redis.Client, sourceID string) cache.Cache {
	return cache.New(
		cache.WithName("user-service"),
		cache.WithLocal(local.NewTinyLFU(200_000, time.Minute)),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithSyncLocal(true),
		cache.WithSourceId(sourceID),
		cache.WithEventHandler(func(e *cache.Event) {
			payload, _ := json.Marshal(e)
			_ = rdb.Publish(context.Background(), "cache:invalidate", payload).Err()
		}),
	)
}

func getUser(ctx context.Context, c cache.Cache, id int64) (User, error) {
	key := "user:profile:" + strconv.FormatInt(id, 10)
	var out User
	err := c.Once(ctx, key,
		cache.Value(&out),
		cache.Do(func(context.Context) (any, error) {
			return User{ID: id, Name: "from-db"}, nil
		}),
	)
	return out, err
}

func updateUser(ctx context.Context, c cache.Cache, id int64, name string) error {
	// 1) 先更新 DB
	// 2) 再失效缓存
	return c.Delete(ctx, "user:profile:"+strconv.FormatInt(id, 10))
}
```

实现提示：

- `SourceID` 建议按进程实例唯一生成（例如：`user-svc-prod-<podUID>-<bootNonce>`）。
- 每个节点订阅 `cache:invalidate`，只对来自其他 `SourceID` 的 key 调用 `DeleteFromLocalCache`。
