package cache

import (
	"context"
	"time"

	"github.com/daoshenzzg/jetcache-go/logger"
)

type (
	Item struct {
		Ctx       context.Context
		Key       string
		Value     interface{}                      // Value gets the value for the given key and fills into value.
		TTL       time.Duration                    // TTL is the cache expiration time. Default TTL is 1 hour.
		Do        func(*Item) (interface{}, error) // Do returns value to be cached.
		SetXX     bool                             // SetXX only sets the key if it already exists.
		SetNX     bool                             // SetNX only sets the key if it does not already exist.
		SkipLocal bool                             // SkipLocal skips local cache as if it is not set.
		Refresh   bool                             // Refresh open cache async refresh.
	}

	RefreshTask struct {
		Key            string
		TTL            time.Duration
		Do             func(*Item) (interface{}, error)
		SetXX          bool
		SetNX          bool
		SkipLocal      bool
		LastAccessTime time.Time
	}
)

func (item *Item) Context() context.Context {
	if item.Ctx == nil {
		return context.Background()
	}
	return item.Ctx
}

func (item *Item) value() (interface{}, error) {
	if item.Do != nil {
		return item.Do(item)
	}
	if item.Value != nil {
		return item.Value, nil
	}
	return nil, nil
}

func (item *Item) ttl() time.Duration {
	const defaultTTL = time.Hour

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

func (item *Item) toRefreshTask() *RefreshTask {
	return &RefreshTask{
		Key:            item.Key,
		TTL:            item.TTL,
		Do:             item.Do,
		SkipLocal:      item.SkipLocal,
		LastAccessTime: time.Now(),
	}
}

func (rt *RefreshTask) toItem(ctx context.Context) *Item {
	return &Item{
		Ctx:       ctx,
		Key:       rt.Key,
		TTL:       rt.TTL,
		Do:        rt.Do,
		SetXX:     rt.SetXX,
		SetNX:     rt.SetNX,
		SkipLocal: rt.SkipLocal,
	}
}
