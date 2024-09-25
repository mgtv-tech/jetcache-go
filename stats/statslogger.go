package stats

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultStatsInterval = time.Minute

var (
	once  sync.Once
	inner *innerStats
	_     Handler = (*Stats)(nil)
)

type (
	Stats struct {
		Options
		Name       string
		Hit        uint64
		Miss       uint64
		LocalHit   uint64
		LocalMiss  uint64
		RemoteHit  uint64
		RemoteMiss uint64
		Query      uint64
		QueryFail  uint64
	}

	Options struct {
		statsInterval time.Duration
	}

	// Option defines the method to customize an Options.
	Option func(o *Options)

	innerStats struct {
		statsInterval time.Duration
		stats         []*Stats
	}
)

func WithStatsInterval(statsInterval time.Duration) Option {
	return func(o *Options) {
		o.statsInterval = statsInterval
	}
}

func NewStatsLogger(name string, opts ...Option) Handler {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	if o.statsInterval <= 0 {
		o.statsInterval = defaultStatsInterval
	}
	once.Do(func() {
		inner = &innerStats{
			statsInterval: o.statsInterval,
			stats:         make([]*Stats, 0),
		}

		go func() {
			ticker := time.NewTicker(o.statsInterval)
			defer ticker.Stop()

			inner.statLoop(ticker)
		}()
	})

	stat := &Stats{
		Name:    name,
		Options: o,
	}

	inner.stats = append(inner.stats, stat)

	return stat
}

func (s *Stats) IncrHit() {
	atomic.AddUint64(&s.Hit, 1)
}

func (s *Stats) IncrMiss() {
	atomic.AddUint64(&s.Miss, 1)
}

func (s *Stats) IncrLocalHit() {
	atomic.AddUint64(&s.LocalHit, 1)
}

func (s *Stats) IncrLocalMiss() {
	atomic.AddUint64(&s.LocalMiss, 1)
}

func (s *Stats) IncrRemoteHit() {
	atomic.AddUint64(&s.RemoteHit, 1)
}

func (s *Stats) IncrRemoteMiss() {
	atomic.AddUint64(&s.RemoteMiss, 1)
}

func (s *Stats) IncrQuery() {
	atomic.AddUint64(&s.Query, 1)
}

func (s *Stats) IncrQueryFail(err error) {
	atomic.AddUint64(&s.QueryFail, 1)
}

func (inner *innerStats) statLoop(ticker *time.Ticker) {
	for range ticker.C {
		inner.logStatSummary()
	}
}

func (inner *innerStats) logStatSummary() {
	stats := make([]Stats, len(inner.stats))
	var maxNameLen int
	for i, s := range inner.stats {
		stats[i] = Stats{
			Name:       s.Name,
			Hit:        atomic.SwapUint64(&s.Hit, 0),
			Miss:       atomic.SwapUint64(&s.Miss, 0),
			RemoteHit:  atomic.SwapUint64(&s.RemoteHit, 0),
			RemoteMiss: atomic.SwapUint64(&s.RemoteMiss, 0),
			LocalHit:   atomic.SwapUint64(&s.LocalHit, 0),
			LocalMiss:  atomic.SwapUint64(&s.LocalMiss, 0),
			Query:      atomic.SwapUint64(&s.Query, 0),
			QueryFail:  atomic.SwapUint64(&s.QueryFail, 0),
		}
		if len(s.Name) > maxNameLen {
			maxNameLen = len(s.Name)
		}
	}
	maxLenStr := strconv.Itoa(maxNameLen + 7)
	rows := formatRows(stats, maxLenStr)
	if len(rows) > 0 {
		var sb strings.Builder
		header := formatHeader(maxLenStr)
		sb.WriteString(fmt.Sprintf("jetcache-go stats last %s.\n", inner.statsInterval))
		sb.WriteString(header)
		sb.WriteString(formatSepLine(header))
		sb.WriteString(rows)
		sb.WriteString(formatSepLine(header))
		log.Println(sb.String())
	}
}

func formatHeader(maxLenStr string) string {
	return fmt.Sprintf("%-"+maxLenStr+"s|%12s|%12s|%12s|%12s|%12s|%12s", "cache", "qpm", "hit_ratio", "hit", "miss", "query", "query_fail\n")
}

func formatRows(stats []Stats, maxLenStr string) string {
	var rows strings.Builder
	for _, s := range stats {
		total := s.Hit + s.Miss
		remoteTotal := s.RemoteHit + s.RemoteMiss
		localTotal := s.LocalHit + s.LocalMiss
		if total == 0 && s.Query == 0 && s.QueryFail == 0 {
			continue
		}
		// All
		rows.WriteString(fmt.Sprintf("%-"+maxLenStr+"s|", s.Name))
		rows.WriteString(fmt.Sprintf("%12d|", total))
		rows.WriteString(fmt.Sprintf("%11s", rate(s.Hit, total)))
		rows.WriteString("%|")
		rows.WriteString(fmt.Sprintf("%12d|", s.Hit))
		rows.WriteString(fmt.Sprintf("%12d|", s.Miss))
		rows.WriteString(fmt.Sprintf("%12d|", s.Query))
		rows.WriteString(fmt.Sprintf("%12d", s.QueryFail))
		rows.WriteString("\n")
		// Local
		if localTotal > 0 {
			rows.WriteString(fmt.Sprintf("%-"+maxLenStr+"s|", getName(s.Name, "local")))
			rows.WriteString(fmt.Sprintf("%12d|", localTotal))
			rows.WriteString(fmt.Sprintf("%11s", rate(s.LocalHit, localTotal)))
			rows.WriteString("%|")
			rows.WriteString(fmt.Sprintf("%12d|", s.LocalHit))
			rows.WriteString(fmt.Sprintf("%12d|", s.LocalMiss))
			rows.WriteString(fmt.Sprintf("%12s|", "-"))
			rows.WriteString(fmt.Sprintf("%12s", "-"))
			rows.WriteString("\n")
		}
		// Remote
		if remoteTotal > 0 {
			rows.WriteString(fmt.Sprintf("%-"+maxLenStr+"s|", getName(s.Name, "remote")))
			rows.WriteString(fmt.Sprintf("%12d|", remoteTotal))
			rows.WriteString(fmt.Sprintf("%11s", rate(s.RemoteHit, remoteTotal)))
			rows.WriteString("%|")
			rows.WriteString(fmt.Sprintf("%12d|", s.RemoteHit))
			rows.WriteString(fmt.Sprintf("%12d|", s.RemoteMiss))
			rows.WriteString(fmt.Sprintf("%12s|", "-"))
			rows.WriteString(fmt.Sprintf("%12s", "-"))
			rows.WriteString("\n")
		}
	}

	return rows.String()
}

func formatSepLine(header string) string {
	var b bytes.Buffer
	for _, c := range header {
		if c == '|' {
			b.WriteString("+")
		} else {
			b.WriteString("-")
		}
	}
	b.WriteString("\n")
	return b.String()
}

func rate(count, total uint64) string {
	if total == 0 {
		return "0.00"
	}

	return fmt.Sprintf("%2.2f", float64(count*100)/float64(total))
}

func getName(name, typ string) string {
	return fmt.Sprintf("%s_%s", name, typ)
}
