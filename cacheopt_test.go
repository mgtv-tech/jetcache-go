package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jetcache-go/encoding/json"
)

func TestCacheOptions(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		o := newOptions()
		assert.Equal(t, defaultNotFoundExpiry, o.notFoundExpiry)
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
	})

	t.Run("with not found expiry", func(t *testing.T) {
		o := newOptions(WithNotFoundExpiry(time.Second))
		assert.Equal(t, defaultRefreshConcurrency, o.refreshConcurrency)
		assert.Equal(t, defaultCodec, o.codec)
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

	t.Run("with stats disabled", func(t *testing.T) {
		o := newOptions(WithStatsDisabled(true))
		assert.Equal(t, true, o.statsDisabled)
	})
}
