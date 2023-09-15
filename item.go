package cache

import (
	"context"
	"time"

	"github.com/daoshenzzg/jetcache-go/logger"
)

const defaultTTL = time.Hour

type (
	// ItemOption defines the method to customize an Options.
	ItemOption func(o *item)
	// DoFunc returns value to be cached.
	DoFunc func() (interface{}, error)

	item struct {
		Ctx       context.Context
		Key       string
		Value     interface{}   // Value gets the value for the given key and fills into value.
		TTL       time.Duration // TTL is the remote cache expiration time. Default TTL is 1 hour.
		Do        DoFunc        // Do is DoFunc
		SetXX     bool          // SetXX only sets the key if it already exists.
		SetNX     bool          // SetNX only sets the key if it does not already exist.
		SkipLocal bool          // SkipLocal skips local cache as if it is not set.
		Refresh   bool          // Refresh open cache async refresh.
	}

	refreshTask struct {
		Key            string
		TTL            time.Duration
		Do             DoFunc
		SetXX          bool
		SetNX          bool
		SkipLocal      bool
		LastAccessTime time.Time
	}
)

func newItemOptions(ctx context.Context, key string, opts ...ItemOption) *item {
	var item = item{Ctx: ctx, Key: key}
	for _, opt := range opts {
		opt(&item)
	}

	return &item
}

func Value(value interface{}) ItemOption {
	return func(o *item) {
		o.Value = value
	}
}

func TTL(ttl time.Duration) ItemOption {
	return func(o *item) {
		o.TTL = ttl
	}
}

func Do(do DoFunc) ItemOption {
	return func(o *item) {
		o.Do = do
	}
}

func SetXX(setXx bool) ItemOption {
	return func(o *item) {
		o.SetXX = setXx
	}
}

func SetNX(setNx bool) ItemOption {
	return func(o *item) {
		o.SetNX = setNx
	}
}

func SkipLocal(skipLocal bool) ItemOption {
	return func(o *item) {
		o.SkipLocal = skipLocal
	}
}

func Refresh(refresh bool) ItemOption {
	return func(o *item) {
		o.Refresh = refresh
	}
}

func (item *item) Context() context.Context {
	if item.Ctx == nil {
		return context.Background()
	}
	return item.Ctx
}

func (item *item) value() (interface{}, error) {
	if item.Do != nil {
		return item.Do()
	}
	if item.Value != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (item *item) ttl() time.Duration {
	if item.TTL < 0 {
		return 0
	}

	if item.TTL != 0 {
		if item.TTL < time.Second {
			logger.Warn("too short TTL for key=%q: %s", item.Key, item.TTL)
			return defaultTTL
		}
		return item.TTL
	}

	return defaultTTL
}

func (item *item) toRefreshTask() *refreshTask {
	return &refreshTask{
		Key:            item.Key,
		TTL:            item.TTL,
		Do:             item.Do,
		SkipLocal:      item.SkipLocal,
		LastAccessTime: time.Now(),
	}
}
