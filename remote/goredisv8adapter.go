package remote

import (
	"context"
	"errors"
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

func (r *GoRedisV8Adaptor) SetEX(ctx context.Context, key string, value any, expire time.Duration) error {
	return r.client.SetEX(ctx, key, value, expire).Err()
}

func (r *GoRedisV8Adaptor) SetNX(ctx context.Context, key string, value any, expire time.Duration) (val bool, err error) {
	return r.client.SetNX(ctx, key, value, expire).Result()
}

func (r *GoRedisV8Adaptor) SetXX(ctx context.Context, key string, value any, expire time.Duration) (val bool, err error) {
	return r.client.SetXX(ctx, key, value, expire).Result()
}

func (r *GoRedisV8Adaptor) Get(ctx context.Context, key string) (val string, err error) {
	return r.client.Get(ctx, key).Result()
}

func (r *GoRedisV8Adaptor) Del(ctx context.Context, key string) (val int64, err error) {
	return r.client.Del(ctx, key).Result()
}

func (r *GoRedisV8Adaptor) MGet(ctx context.Context, keys ...string) (map[string]any, error) {
	pipeline := r.client.Pipeline()
	keyIdxMap := make(map[int]string, len(keys))
	ret := make(map[string]any, len(keys))

	for idx, key := range keys {
		keyIdxMap[idx] = key
		pipeline.Get(ctx, key)
	}

	cmder, err := pipeline.Exec(ctx)
	if err != nil && !errors.Is(err, r.Nil()) {
		return nil, err
	}

	for idx, cmd := range cmder {
		if strCmd, ok := cmd.(*redis.StringCmd); ok {
			key := keyIdxMap[idx]
			if val, _ := strCmd.Result(); len(val) > 0 {
				ret[key] = val
			}
		}
	}

	return ret, nil
}

func (r *GoRedisV8Adaptor) MSet(ctx context.Context, value map[string]any, expire time.Duration) error {
	pipeline := r.client.Pipeline()

	for key, val := range value {
		pipeline.SetEX(ctx, key, val, expire)
	}
	_, err := pipeline.Exec(ctx)

	return err
}

func (r *GoRedisV8Adaptor) Nil() error {
	return redis.Nil
}
