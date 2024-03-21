package remote

import (
	"context"
	"time"
)

type Remote interface {
	// SetEX sets the expiration value for a key.
	SetEX(ctx context.Context, key string, value any, expire time.Duration) error

	// SetNX sets the value of a key if it does not already exist.
	SetNX(ctx context.Context, key string, value any, expire time.Duration) (val bool, err error)

	// SetXX sets the value of a key if it already exists.
	SetXX(ctx context.Context, key string, value any, expire time.Duration) (val bool, err error)

	// Get retrieves the value of a key. It returns errNotFound (e.g., redis.Nil) when the key does not exist.
	Get(ctx context.Context, key string) (val string, err error)

	// Del deletes the cached value associated with a key.
	Del(ctx context.Context, key string) (val int64, err error)

	// MGet retrieves the values of multiple keys.
	MGet(ctx context.Context, keys ...string) (map[string]any, error)

	// MSet sets multiple key-value pairs in the cache.
	MSet(ctx context.Context, value map[string]any, expire time.Duration) error

	// Nil returns an error indicating that the key does not exist.
	Nil() error
}
