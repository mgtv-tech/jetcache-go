package cache

import (
	"github.com/daoshenzzg/jetcache-go/stats"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/daoshenzzg/jetcache-go/encoding/json"
)

func TestCacheOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		o := newOptions()
		assert.Equal(t, defaultName, o.name)
		assert.Equal(t, defaultNotFoundExpiry, o.notFoundExpiry)
		assert.Equal(t, defaultNotFoundExpiry/10, o.offset)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
		assert.NotNil(t, o.statsHandler)
	})

	t.Run("with name", func(t *testing.T) {
		o := newOptions(WithName("any"))
		assert.Equal(t, "any", o.name)
		assert.Equal(t, defaultNotFoundExpiry/10, o.offset)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
	})

	t.Run("with not found expiry", func(t *testing.T) {
		o := newOptions(WithNotFoundExpiry(time.Second))
		assert.Equal(t, time.Second/10, o.offset)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
	})

	t.Run("with offset", func(t *testing.T) {
		o := newOptions(WithOffset(time.Second))
		assert.Equal(t, time.Second, o.offset)
	})

	t.Run("with max offset", func(t *testing.T) {
		o := newOptions(WithOffset(30 * time.Second))
		assert.Equal(t, maxOffset, o.offset)
	})

	t.Run("with refresh concurrency", func(t *testing.T) {
		o := newOptions(WithRefreshConcurrency(16))
		assert.Equal(t, defaultNotFoundExpiry, o.notFoundExpiry)
		assert.Equal(t, 16, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
	})

	t.Run("with codec", func(t *testing.T) {
		o := newOptions(WithCodec(json.Name))
		assert.Equal(t, defaultNotFoundExpiry, o.notFoundExpiry)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, json.Name, o.codec)
	})

	t.Run("with stats handler", func(t *testing.T) {
		stat := stats.NewStatsLogger("any")
		o := newOptions(WithStatsHandler(stat))
		assert.Equal(t, stat, o.statsHandler)
	})

	t.Run("with stats disabled", func(t *testing.T) {
		o := newOptions(WithStatsDisabled(true))
		assert.Equal(t, true, o.statsDisabled)
	})
}
