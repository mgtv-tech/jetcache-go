# Microservice Pattern Example

This example shows a common microservice setup:

- two-level cache,
- cache-aside read,
- write-after-delete invalidation,
- optional cross-node local cache invalidation event hook.

Version note: cross-process local invalidation hook (`WithSyncLocal`) is available since `v1.1.1+`.

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
	// 1) update DB first
	// 2) invalidate cache
	return c.Delete(ctx, "user:profile:"+strconv.FormatInt(id, 10))
}
```

Implementation note:

- `SourceID` should be unique per process instance (for example: `user-svc-prod-<podUID>-<bootNonce>`).
- Subscribe `cache:invalidate` in each node and call `DeleteFromLocalCache` only for keys from other `SourceID`s.
