package cache

import (
	"context"
	"time"

	"github.com/mgtv-tech/jetcache-go/logger"
)

type (
	// ItemOption defines the method to customize an Options.
	ItemOption func(o *item)

	// DoFunc returns getValue to be cached.
	DoFunc func(ctx context.Context) (any, error)

	item struct {
		ctx       context.Context
		key       string
		value     any           // value gets the value for the given key and fills into value.
		ttl       time.Duration // ttl is the remote cache expiration time. Default ttl is 1 hour.
		do        DoFunc        // do is DoFunc
		setXX     bool          // setXX only sets the key if it already exists.
		setNX     bool          // setNX only sets the key if it does not already exist.
		skipLocal bool          // skipLocal skips local cache as if it is not set.
		refresh   bool          // refresh open cache async refresh.
	}

	refreshTask struct {
		key            string
		ttl            time.Duration
		do             DoFunc
		setXX          bool
		setNX          bool
		skipLocal      bool
		lastAccessTime time.Time
	}
)

func newItemOptions(ctx context.Context, key string, opts ...ItemOption) *item {
	var item = item{ctx: ctx, key: key}
	for _, opt := range opts {
		opt(&item)
	}

	return &item
}

func Value(value any) ItemOption {
	return func(o *item) {
		o.value = value
	}
}

func TTL(ttl time.Duration) ItemOption {
	return func(o *item) {
		o.ttl = ttl
	}
}

func Do(do DoFunc) ItemOption {
	return func(o *item) {
		o.do = do
	}
}

func SetXX(setXx bool) ItemOption {
	return func(o *item) {
		o.setXX = setXx
	}
}

func SetNX(setNx bool) ItemOption {
	return func(o *item) {
		o.setNX = setNx
	}
}

func SkipLocal(skipLocal bool) ItemOption {
	return func(o *item) {
		o.skipLocal = skipLocal
	}
}

func Refresh(refresh bool) ItemOption {
	return func(o *item) {
		o.refresh = refresh
	}
}

func (item *item) Context() context.Context {
	if item.ctx == nil {
		return context.Background()
	}
	return item.ctx
}

func (item *item) getValue() (any, error) {
	if item.do != nil {
		return item.do(item.Context())
	}
	if item.value != nil {
		return item.value, nil
	}
	return nil, nil
}

func (item *item) getTtl(defaultTTL time.Duration) time.Duration {
	if item.ttl < 0 {
		return 0
	}

	if item.ttl != 0 {
		if item.ttl < time.Second {
			logger.Warn("too short ttl for key=%q: %s", item.key, item.ttl)
			return defaultTTL
		}
		return item.ttl
	}

	return defaultTTL
}

func (item *item) toRefreshTask() *refreshTask {
	return &refreshTask{
		key:            item.key,
		ttl:            item.ttl,
		do:             item.do,
		skipLocal:      item.skipLocal,
		lastAccessTime: time.Now(),
	}
}
