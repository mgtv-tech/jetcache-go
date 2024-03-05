package cache

import (
	"time"

	"github.com/daoshenzzg/jetcache-go/encoding/sonic"
	"github.com/daoshenzzg/jetcache-go/local"
	"github.com/daoshenzzg/jetcache-go/remote"
	"github.com/daoshenzzg/jetcache-go/stats"
)

const (
	defaultName               = "default"
	defaultRefreshConcurrency = 4
	defaultRemoteExpiry       = time.Hour
	defaultNotFoundExpiry     = time.Minute
	defaultCodec              = sonic.Name
	maxOffset                 = 10 * time.Second
)

type (
	// Options are used to store cache options.
	Options struct {
		name                       string        // Cache name, used for log identification and metric reporting
		remote                     remote.Remote // Remote is distributed cache, such as Redis.
		local                      local.Local   // Local is memory cache, such as FreeCache.
		codec                      string        // Value encoding and decoding method. Default is "sonic.Name". You can also customize it.
		errNotFound                error         // Error to return for cache miss. Used to prevent cache penetration.
		remoteExpiry               time.Duration // Remote cache ttl, Default is 1 hour.
		notFoundExpiry             time.Duration // Duration for placeholder cache when there is a cache miss. Default is 1 minute.
		offset                     time.Duration // Expiration time jitter factor for cache misses.
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
	if o.name == "" {
		o.name = defaultName
	}
	if o.codec == "" {
		o.codec = defaultCodec
	}
	if o.remoteExpiry <= 0 {
		o.remoteExpiry = defaultRemoteExpiry
	}
	if o.notFoundExpiry <= 0 {
		o.notFoundExpiry = defaultNotFoundExpiry
	}
	if o.offset <= 0 {
		o.offset = o.notFoundExpiry / 10
	}
	if o.offset > maxOffset {
		o.offset = maxOffset
	}
	if o.refreshConcurrency <= 0 {
		o.refreshConcurrency = defaultRefreshConcurrency
	}
	if o.stopRefreshAfterLastAccess <= 0 {
		o.stopRefreshAfterLastAccess = o.refreshDuration + time.Second
	}
	if o.statsHandler == nil {
		o.statsHandler = stats.NewHandles(o.statsDisabled, stats.NewStatsLogger(o.name))
	}

	return o
}

func WithName(name string) Option {
	return func(o *Options) {
		o.name = name
	}
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

func WithRemoteExpiry(remoteExpiry time.Duration) Option {
	return func(o *Options) {
		o.remoteExpiry = remoteExpiry
	}
}

func WithNotFoundExpiry(notFoundExpiry time.Duration) Option {
	return func(o *Options) {
		o.notFoundExpiry = notFoundExpiry
	}
}

func WithOffset(offset time.Duration) Option {
	return func(o *Options) {
		o.offset = offset
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

func WithStatsDisabled(statsDisabled bool) Option {
	return func(o *Options) {
		o.statsDisabled = statsDisabled
	}
}
