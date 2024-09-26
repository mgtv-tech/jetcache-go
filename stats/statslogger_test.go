package stats

import (
	"bytes"
	"errors"
	"fmt"
	"log"
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

func TestStatLogger_logStatSummary(t *testing.T) {
	var logBuffer = &bytes.Buffer{}
	log.SetOutput(logBuffer)
	stats := []*Stats{
		{Name: "cache1", Hit: 10, Miss: 2, RemoteHit: 5, RemoteMiss: 1, LocalHit: 5, LocalMiss: 1, Query: 100, QueryFail: 5},
		{Name: "cache2", Hit: 5, Miss: 0, Query: 50},
	}
	inner := &innerStats{
		stats:         stats,
		statsInterval: time.Minute,
	}

	inner.logStatSummary()

	expected := `jetcache-go stats last 1m0s.
cache        |         qpm|   hit_ratio|         hit|        miss|       query|  query_fail
-------------+------------+------------+------------+------------+------------+------------
cache1       |          12|      83.33%|          10|           2|         100|           5
cache1_local |           6|      83.33%|           5|           1|           -|           -
cache1_remote|           6|      83.33%|           5|           1|           -|           -
cache2       |           5|     100.00%|           5|           0|          50|           0
-------------+------------+------------+------------+------------+------------+------------`

	assert.Contains(t, logBuffer.String(), expected)
}

func TestFormatHeader(t *testing.T) {
	maxLenStr := "12"
	expected := fmt.Sprintf("%-12s|%12s|%12s|%12s|%12s|%12s|%12s\n", "cache", "qpm", "hit_ratio", "hit", "miss", "query", "query_fail")
	actual := formatHeader(maxLenStr)
	if actual != expected {
		t.Errorf("formatHeader failed. Expected: %s, Actual: %s", expected, actual)
	}
}

func TestFormatSepLine(t *testing.T) {
	header := "cache        |         qpm|   hit_ratio|         hit|        miss|       query| query_fail\n"
	expected := "-------------+------------+------------+------------+------------+------------+-----------\n"
	actual := formatSepLine(header)
	assert.Equal(t, expected, actual)
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
