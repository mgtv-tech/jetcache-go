# High Concurrency Hot-Key Example

This example focuses on hot-key protection with `Once`, singleflight, and optional refresh.

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

type Product struct {
	ID    int64
	Stock int64
}

func newCache(rdb *redis.Client) cache.Cache {
	return cache.New(
		cache.WithName("product-hot"),
		cache.WithLocal(local.NewTinyLFU(300_000, 2*time.Minute)),
		cache.WithRemote(remote.NewGoRedisV9Adapter(rdb)),
		cache.WithRefreshDuration(30*time.Second),
		cache.WithStopRefreshAfterLastAccess(10*time.Minute),
		cache.WithRefreshConcurrency(16),
	)
}

func getHotProduct(ctx context.Context, c cache.Cache, id int64) (Product, error) {
	var p Product
	err := c.Once(ctx, "product:hot:1001",
		cache.Value(&p),
		cache.Refresh(true),
		cache.Do(func(context.Context) (any, error) {
			// Simulate expensive DB query.
			time.Sleep(50 * time.Millisecond)
			return Product{ID: id, Stock: 99}, nil
		}),
	)
	return p, err
}
```

Tips:

- Keep refresh enabled only for a small set of hot keys.
- Monitor `query_fail` and hit ratio during peak traffic.
