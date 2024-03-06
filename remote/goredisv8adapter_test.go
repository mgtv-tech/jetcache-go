package remote

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestGoRedisV8Adaptor_MGet(t *testing.T) {
	client := NewGoRedisV8Adaptor(newRdb())

	if err := client.SetEX(context.Background(), "key1", "value1", time.Minute); err != nil {
		t.Fatal(err)
	}

	val, err := client.Get(context.Background(), "key1")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "value1", val)

	result, err := client.MGet(context.Background(), "key1", "key2")
	assert.Nil(t, err)
	assert.Equal(t, map[string]any{"key1": "value1"}, result)
}

func TestGoRedisV8Adaptor_MSet(t *testing.T) {
	client := NewGoRedisV8Adaptor(newRdb())

	err := client.MSet(context.Background(), map[string]any{"key1": "value1", "key2": 2}, time.Minute)
	assert.Nil(t, err)

	val, err := client.Get(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, "value1", val)

	val, err = client.Get(context.Background(), "key2")
	assert.Nil(t, err)
	assert.Equal(t, "2", val)

	result, err := client.MGet(context.Background(), "key1", "key2")
	assert.Nil(t, err)
	assert.Equal(t, map[string]any{"key1": "value1", "key2": "2"}, result)
}

func newRdb() *redis.Client {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	return redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
}
