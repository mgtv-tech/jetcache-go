package stats

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStatsLogger(t *testing.T) {
	t.Run("default stats interval", func(t *testing.T) {
		logger := NewStatsLogger("any")
		stats := logger.(*Stats)
		assert.Equal(t, defaultStatsInterval, stats.statsInterval)
	})

	t.Run("custom stats interval", func(t *testing.T) {
		logger := NewStatsLogger("any", WithStatsInterval(time.Millisecond))
		stats := logger.(*Stats)
		assert.Equal(t, stats.statsInterval, time.Millisecond, stats.statsInterval)
	})
}

func TestStatLogger_statLoop(t *testing.T) {
	t.Run("stat loop stats 0", func(t *testing.T) {
		_ = NewStatsLogger("any", WithStatsInterval(time.Millisecond))
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("stat loop total not 0", func(t *testing.T) {
		stat := NewStatsLogger("any", WithStatsInterval(time.Millisecond))
		stat.IncrHit()
		stat.IncrMiss()
		stat.IncrLocalHit()
		stat.IncrLocalMiss()
		stat.IncrRemoteHit()
		stat.IncrRemoteMiss()
		stat.IncrQuery()
		stat.IncrQueryFail(errors.New("any"))

		for i := 0; i < 3; i++ {
			stat := NewStatsLogger(fmt.Sprintf("test_lang_cache_%d", i), WithStatsInterval(time.Millisecond))
			stat.IncrHit()
			stat.IncrMiss()
			stat.IncrLocalHit()
			stat.IncrLocalMiss()
			stat.IncrRemoteHit()
			stat.IncrRemoteMiss()
			stat.IncrQuery()
			stat.IncrQueryFail(errors.New("any"))
		}
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("stat loop query not 0", func(t *testing.T) {
		stat := NewStatsLogger("any", WithStatsInterval(time.Millisecond))
		stat.IncrQuery()
		time.Sleep(10 * time.Millisecond)
	})
}

func TestStatLogger_race(t *testing.T) {
	testCases := []struct {
		count  uint64
		total  uint64
		expect string
	}{
		{10, 100, "10.00"},
		{0, 100, "0.00"},
		{10, 0, "0.00"},
		{50, 50, "100.00"},
	}
	for _, tc := range testCases {
		actual := rate(tc.count, tc.total)
		assert.Equal(t, tc.expect, actual)
	}
}

func TestStatLogger_getName(t *testing.T) {
	assert.Equal(t, "cache_local", getName("cache", "local"))
}
