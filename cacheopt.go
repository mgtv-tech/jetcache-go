package cache

import (
	"time"

	"github.com/jetcache-go/encoding/msgpack"
	"github.com/jetcache-go/local"
	"github.com/jetcache-go/remote"
	"github.com/jetcache-go/stats"
)

const (
	defaultNotFoundExpiry     = time.Minute
	defaultRefreshConcurrency = 4
	defaultCodec              = msgpack.Name
)

type (
	// Options are used to store cache options.
	Options struct {
		remote                     remote.Remote // Remote cache.
		local                      local.Local   // Local cache.
		codec                      string        // Value encoding and decoding method. Default is "json.Name" or "msgpack.Name". You can also customize it.
		errNotFound                error         // Error to return for cache miss. Used to prevent cache penetration.
		notFoundExpiry             time.Duration // Duration for placeholder cache when there is a cache miss. Default is 1 minute.
		refreshDuration            time.Duration // Interval for asynchronous cache refresh. Default is 0 (refresh is disabled).
		stopRefreshAfterLastAccess time.Duration // Duration for cache to stop refreshing after no access. Default is refreshDuration + 1 second.
		refreshConcurrency         int           // Maximum number of concurrent cache refreshes. Default is 4.
		statsDisabled              bool          // Flag to disable cache statistics.
		statsHandler               stats.Handler // Metrics statsHandler collector.
	}

	// Option defines the method to customize an Options.
	Option func(o *Options)
)

func newOptions(opts ...Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}

	if o.codec == "" {
		o.codec = defaultCodec
	}
	if o.notFoundExpiry <= 0 {
		o.notFoundExpiry = defaultNotFoundExpiry
	}
	if o.refreshConcurrency <= 0 {
		o.refreshConcurrency = defaultRefreshConcurrency
	}
	if o.stopRefreshAfterLastAccess <= 0 {
		o.stopRefreshAfterLastAccess = o.refreshDuration + time.Second
	}
	return o
}

func WithRemote(remote remote.Remote) Option {
	return func(o *Options) {
		o.remote = remote
	}
}

func WithLocal(local local.Local) Option {
	return func(o *Options) {
		o.local = local
	}
}

func WithCodec(codec string) Option {
	return func(o *Options) {
		o.codec = codec
	}
}

func WithErrNotFound(err error) Option {
	return func(o *Options) {
		o.errNotFound = err
	}
}

func WithRefreshDuration(refreshDuration time.Duration) Option {
	return func(o *Options) {
		o.refreshDuration = refreshDuration
	}
}

func WithStopRefreshAfterLastAccess(stopRefreshAfterLastAccess time.Duration) Option {
	return func(o *Options) {
		o.stopRefreshAfterLastAccess = stopRefreshAfterLastAccess
	}
}

func WithRefreshConcurrency(refreshConcurrency int) Option {
	return func(o *Options) {
		o.refreshConcurrency = refreshConcurrency
	}
}

func WithStatsHandler(handler stats.Handler) Option {
	return func(o *Options) {
		o.statsHandler = handler
	}
}

func WithNotFoundExpiry(notFoundExpiry time.Duration) Option {
	return func(o *Options) {
		o.notFoundExpiry = notFoundExpiry
	}
}

func WithStatsDisabled(statsDisabled bool) Option {
	return func(o *Options) {
		o.statsDisabled = statsDisabled
	}
}
