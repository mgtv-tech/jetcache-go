package remote

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var _ Remote = (*GoRedisV8Adaptor)(nil)

type GoRedisV8Adaptor struct {
	client redis.Cmdable
}

// NewGoRedisV8Adaptor is
func NewGoRedisV8Adaptor(client redis.Cmdable) Remote {
	return &GoRedisV8Adaptor{
		client: client,
	}
}

func (r *GoRedisV8Adaptor) SetEX(ctx context.Context, key string, value interface{}, expire time.Duration) error {
	return r.client.SetEX(ctx, key, value, expire).Err()
}

func (r *GoRedisV8Adaptor) SetNX(ctx context.Context, key string, value interface{}, expire time.Duration) (val bool, err error) {
	return r.client.SetNX(ctx, key, value, expire).Result()
}

func (r *GoRedisV8Adaptor) SetXX(ctx context.Context, key string, value interface{}, expire time.Duration) (val bool, err error) {
	return r.client.SetXX(ctx, key, value, expire).Result()
}

func (r *GoRedisV8Adaptor) Get(ctx context.Context, key string) (val string, err error) {
	return r.client.Get(ctx, key).Result()
}

func (r *GoRedisV8Adaptor) Del(ctx context.Context, key string) (val int64, err error) {
	return r.client.Del(ctx, key).Result()
}

func (r *GoRedisV8Adaptor) Nil() error {
	return redis.Nil
}
