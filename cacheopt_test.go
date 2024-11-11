package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mgtv-tech/jetcache-go/encoding/json"
	"github.com/mgtv-tech/jetcache-go/stats"
)

func TestCacheOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		o := newOptions()
		assert.Equal(t, defaultName, o.name)
		assert.Equal(t, defaultRemoteExpiry, o.remoteExpiry)
		assert.Equal(t, defaultNotFoundExpiry, o.notFoundExpiry)
		assert.Equal(t, defaultNotFoundExpiry/10, o.offset)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
		assert.NotNil(t, o.statsHandler)
		assert.Equal(t, defaultRandSourceIdLen, len(o.sourceID))
		assert.False(t, o.syncLocal)
		assert.Equal(t, defaultEventChBufSize, o.eventChBufSize)
		assert.Nil(t, o.eventHandler)
		assert.Equal(t, defaultSeparator, o.separator)
		assert.Equal(t, false, o.separatorDisabled)
	})

	t.Run("with name", func(t *testing.T) {
		o := newOptions(WithName("any"))
		assert.Equal(t, "any", o.name)
		assert.Equal(t, defaultNotFoundExpiry/10, o.offset)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
	})

	t.Run("with remote expiry", func(t *testing.T) {
		o := newOptions(WithRemoteExpiry(time.Second))
		assert.Equal(t, time.Second, o.remoteExpiry)
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

	t.Run("with refresh duration", func(t *testing.T) {
		o := newOptions(WithRefreshDuration(time.Second))
		assert.Equal(t, time.Second, o.refreshDuration)
	})

	t.Run("with refresh concurrency", func(t *testing.T) {
		o := newOptions(WithRefreshConcurrency(16))
		assert.Equal(t, defaultNotFoundExpiry, o.notFoundExpiry)
		assert.Equal(t, 16, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
	})

	t.Run("with mock decode", func(t *testing.T) {
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

	t.Run("with source id", func(t *testing.T) {
		sourceId := "12345678"
		o := newOptions(WithSourceId(sourceId))
		assert.Equal(t, sourceId, o.sourceID)
	})

	t.Run("with sync local", func(t *testing.T) {
		o := newOptions(WithSyncLocal(true))
		assert.True(t, o.syncLocal)
	})

	t.Run("with event chan buffer size", func(t *testing.T) {
		o := newOptions(WithEventChBufSize(10))
		assert.Equal(t, o.eventChBufSize, 10)
	})

	t.Run("with event handler", func(t *testing.T) {
		o := newOptions(WithEventHandler(func(event *Event) {
		}))
		assert.NotNil(t, o.eventHandler)
	})

	t.Run("with separator", func(t *testing.T) {
		o := newOptions(WithSeparator(":"))
		assert.Equal(t, defaultSeparator, o.separator)
	})

	t.Run("with disable default separator", func(t *testing.T) {
		o := newOptions(WithSeparatorDisable(true))
		assert.Equal(t, "", o.separator)
	})

	t.Run("with disable customized separator", func(t *testing.T) {
		o := newOptions(WithSeparatorDisable(true), WithSeparator(":"))
		assert.Equal(t, "", o.separator)
	})
}

func TestCacheOptionsRefreshDuration(t *testing.T) {
	tests := []struct {
		input  time.Duration
		expect time.Duration
	}{
		{
			input:  0,
			expect: 0,
		},
		{
			input:  time.Millisecond,
			expect: minEffectRefreshDuration,
		},
		{
			input:  time.Minute,
			expect: time.Minute,
		},
	}

	for _, v := range tests {
		o := newOptions(WithRefreshDuration(v.input))
		assert.Equal(t, v.expect, o.refreshDuration)
	}
}
