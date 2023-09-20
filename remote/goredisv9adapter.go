package remote

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var _ Remote = (*GoRedisV9Adaptor)(nil)

type GoRedisV9Adaptor struct {
	client redis.Cmdable
}

// NewGoRedisV9Adaptor is
func NewGoRedisV9Adaptor(client redis.Cmdable) Remote {
	return &GoRedisV9Adaptor{
		client: client,
	}
}

func (r *GoRedisV9Adaptor) SetEX(ctx context.Context, key string, value interface{}, expire time.Duration) error {
	return r.client.SetEx(ctx, key, value, expire).Err()
}

func (r *GoRedisV9Adaptor) SetNX(ctx context.Context, key string, value interface{}, expire time.Duration) (val bool, err error) {
	return r.client.SetNX(ctx, key, value, expire).Result()
}

func (r *GoRedisV9Adaptor) SetXX(ctx context.Context, key string, value interface{}, expire time.Duration) (val bool, err error) {
	return r.client.SetXX(ctx, key, value, expire).Result()
}

func (r *GoRedisV9Adaptor) Get(ctx context.Context, key string) (val string, err error) {
	return r.client.Get(ctx, key).Result()
}

func (r *GoRedisV9Adaptor) Del(ctx context.Context, key string) (val int64, err error) {
	return r.client.Del(ctx, key).Result()
}

func (r *GoRedisV9Adaptor) Nil() error {
	return redis.Nil
}
