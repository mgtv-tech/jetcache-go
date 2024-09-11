package remote

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestGoRedisV9Adaptor_MGet(t *testing.T) {
	client := NewGoRedisV9Adapter(newRdb())

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

func TestGoRedisV9Adaptor_MSet(t *testing.T) {
	client := NewGoRedisV9Adapter(newRdb())

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

func TestGoRedisV9Adaptor_Del(t *testing.T) {
	client := NewGoRedisV9Adapter(newRdb())

	err := client.SetEX(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)
	val, err := client.Get(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, "value1", val)
	_, err = client.Del(context.Background(), "key1")
	assert.Nil(t, err)
	_, err = client.Get(context.Background(), "key1")
	assert.NotNil(t, err)
	assert.Equal(t, err, client.Nil())
}

func TestGoRedisV9Adaptor_SetXxNx(t *testing.T) {
	client := NewGoRedisV9Adapter(newRdb())

	_, err := client.SetXX(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)
	_, err = client.Get(context.Background(), "key1")
	assert.NotNil(t, err)
	assert.Equal(t, err, client.Nil())

	_, err = client.SetNX(context.Background(), "key1", "value1", time.Minute)
	assert.Nil(t, err)
	val, err := client.Get(context.Background(), "key1")
	assert.Nil(t, err)
	assert.Equal(t, "value1", val)
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
