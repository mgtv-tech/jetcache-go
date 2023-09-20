package stats

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("stat loop query not 0", func(t *testing.T) {
		stat := NewStatsLogger("any", WithStatsInterval(time.Millisecond))
		stat.IncrQuery()
		time.Sleep(10 * time.Millisecond)
	})
}

func TestStatLogger_race(t *testing.T) {
	assert.Equal(t, "0.00", rate(0, 0))
	assert.Equal(t, "1.00", rate(1, 100))
}
