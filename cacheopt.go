package cache

import (
	"fmt"
	"time"

	"github.com/mgtv-tech/jetcache-go/encoding"
	_ "github.com/mgtv-tech/jetcache-go/encoding/json"
	"github.com/mgtv-tech/jetcache-go/encoding/msgpack"
	_ "github.com/mgtv-tech/jetcache-go/encoding/sonic"
	"github.com/mgtv-tech/jetcache-go/local"
	"github.com/mgtv-tech/jetcache-go/remote"
	"github.com/mgtv-tech/jetcache-go/stats"
	"github.com/mgtv-tech/jetcache-go/util"
)

const (
	defaultName               = "default"
	defaultRefreshConcurrency = 4
	defaultRemoteExpiry       = time.Hour
	defaultNotFoundExpiry     = time.Minute
	defaultCodec              = msgpack.Name
	defaultRandSourceIdLen    = 16
	defaultEventChBufSize     = 100
	defaultSeparator          = ":"
	minEffectRefreshDuration  = time.Second
	maxOffset                 = 10 * time.Second
)

const (
	EventTypeSet          EventType = 1
	EventTypeSetByOnce    EventType = 2
	EventTypeSetByRefresh EventType = 3
	EventTypeSetByMGet    EventType = 4
	EventTypeDelete       EventType = 5
)

type (
	// Options are used to store cache options.
	Options struct {
		name                       string             // Cache name, used for log identification and metric reporting
		remote                     remote.Remote      // Remote is distributed cache, such as Redis.
		local                      local.Local        // Local is memory cache, such as FreeCache.
		codec                      string             // Value encoding and decoding method. Default is "msgpack.Name". You can also customize it.
		errNotFound                error              // Error to return for cache miss. Used to prevent cache penetration.
		remoteExpiry               time.Duration      // Remote cache ttl, Default is 1 hour.
		notFoundExpiry             time.Duration      // Duration for placeholder cache when there is a cache miss. Default is 1 minute.
		offset                     time.Duration      // Expiration time jitter factor for cache misses.
		refreshDuration            time.Duration      // Interval for asynchronous cache refresh. Default is 0 (refresh is disabled).
		stopRefreshAfterLastAccess time.Duration      // Duration for cache to stop refreshing after no access. Default is refreshDuration + 1 second.
		refreshConcurrency         int                // Maximum number of concurrent cache refreshes. Default is 4.
		statsDisabled              bool               // Flag to disable cache statistics.
		statsHandler               stats.Handler      // Metrics statsHandler collector.
		sourceID                   string             // Unique identifier for cache instance.
		syncLocal                  bool               // Enable events for syncing local cache (only for "Both" cache type).
		eventChBufSize             int                // Buffer size for event channel (default: 100).
		eventHandler               func(event *Event) // Function to handle local cache invalidation events.
		separatorDisabled          bool               // Disable separator for cache key. Default is false. If true, the cache key will not be split into multiple parts.
		separator                  string             // Separator for cache key. Default is ":".
	}

	// Option defines the method to customize an Options.
	Option func(o *Options)

	EventType int

	Event struct {
		CacheName string
		SourceID  string
		EventType EventType
		Keys      []string
	}
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
	if o.refreshDuration > 0 && o.refreshDuration < minEffectRefreshDuration {
		o.refreshDuration = minEffectRefreshDuration
	}
	if o.stopRefreshAfterLastAccess <= 0 {
		o.stopRefreshAfterLastAccess = o.refreshDuration + time.Second
	}
	if o.statsHandler == nil {
		o.statsHandler = stats.NewHandles(o.statsDisabled, stats.NewStatsLogger(o.name))
	}
	if o.sourceID == "" {
		o.sourceID = util.NewSafeRand().RandN(defaultRandSourceIdLen)
	}
	if o.eventChBufSize <= 0 {
		o.eventChBufSize = defaultEventChBufSize
	}
	if o.separator == "" && !o.separatorDisabled {
		o.separator = defaultSeparator
	}
	if encoding.GetCodec(o.codec) == nil {
		panic(fmt.Sprintf("encoding %s is not registered, please register it first", o.codec))
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

func WithSourceId(sourceId string) Option {
	return func(o *Options) {
		o.sourceID = sourceId
	}
}

func WithSyncLocal(syncLocal bool) Option {
	return func(o *Options) {
		o.syncLocal = syncLocal
	}
}

func WithEventChBufSize(eventChBufSize int) Option {
	return func(o *Options) {
		o.eventChBufSize = eventChBufSize
	}
}

func WithEventHandler(eventHandler func(event *Event)) Option {
	return func(o *Options) {
		o.eventHandler = eventHandler
	}
}

func WithSeparatorDisabled(separatorDisabled bool) Option {
	return func(o *Options) {
		o.separatorDisabled = separatorDisabled
	}
}

func WithSeparator(separator string) Option {
	return func(o *Options) {
		if !o.separatorDisabled {
			o.separator = separator
		}
	}
}
