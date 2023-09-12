package stats

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jetcache-go/logger"
)

const defaultStatsInterval = time.Minute

var _ Handler = (*Stats)(nil)

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

	ret := &Stats{
		Name:    name,
		Options: o,
	}

	go func() {
		ticker := time.NewTicker(o.statsInterval)
		defer ticker.Stop()

		ret.statLoop(ticker)
	}()

	return ret
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

func (s *Stats) statLoop(ticker *time.Ticker) {
	for range ticker.C {
		s.logStatSummary()
	}
}

func (s *Stats) logStatSummary() {
	var (
		hit         = atomic.SwapUint64(&s.Hit, 0)
		miss        = atomic.SwapUint64(&s.Miss, 0)
		remoteHit   = atomic.SwapUint64(&s.RemoteHit, 0)
		remoteMiss  = atomic.SwapUint64(&s.RemoteMiss, 0)
		localHit    = atomic.SwapUint64(&s.LocalHit, 0)
		localMiss   = atomic.SwapUint64(&s.LocalMiss, 0)
		query       = atomic.SwapUint64(&s.Query, 0)
		queryFail   = atomic.SwapUint64(&s.QueryFail, 0)
		total       = hit + miss
		remoteTotal = remoteHit + remoteMiss
		localTotal  = localHit + localMiss
	)

	if total == 0 {
		return
	}

	var (
		b      strings.Builder
		length = len(s.Name) + 7
		lenStr = strconv.Itoa(length)
	)

	b.WriteString(fmt.Sprintf("jetcache-go stats last %v.\n", s.statsInterval))
	// -----Header start------
	title := fmt.Sprintf("%-"+lenStr+"s|%12s|%12s|%12s|%12s|%12s|%12s", "cache", "qpm", "hit_ratio", "hit", "miss", "query", "query_fail")
	b.WriteString(title)
	b.WriteString("\n")
	// -----Header end------

	// -----Rows start------
	printSepLine(&b, title)
	// All
	b.WriteString(fmt.Sprintf("%-"+lenStr+"s|", s.Name))
	b.WriteString(fmt.Sprintf("%12d|", total))
	b.WriteString(fmt.Sprintf("%11s", rate(hit, total)))
	b.WriteString("%%|")
	b.WriteString(fmt.Sprintf("%12d|", hit))
	b.WriteString(fmt.Sprintf("%12d|", miss))
	b.WriteString(fmt.Sprintf("%12d|", query))
	b.WriteString(fmt.Sprintf("%12d|", queryFail))
	b.WriteString("\n")
	// Local
	if localTotal > 0 {
		b.WriteString(fmt.Sprintf("%-"+lenStr+"s|", getName(s.Name, "local")))
		b.WriteString(fmt.Sprintf("%12d|", localTotal))
		b.WriteString(fmt.Sprintf("%11s", rate(localHit, localTotal)))
		b.WriteString("%%|")
		b.WriteString(fmt.Sprintf("%12d|", localHit))
		b.WriteString(fmt.Sprintf("%12d|", localMiss))
		b.WriteString(fmt.Sprintf("%12s|", "-"))
		b.WriteString(fmt.Sprintf("%12s|", "-"))
		b.WriteString("\n")
	}
	// Remote
	if remoteTotal > 0 {
		b.WriteString(fmt.Sprintf("%-"+lenStr+"s|", getName(s.Name, "remote")))
		b.WriteString(fmt.Sprintf("%12d|", remoteTotal))
		b.WriteString(fmt.Sprintf("%11s", rate(remoteHit, remoteTotal)))
		b.WriteString("%%|")
		b.WriteString(fmt.Sprintf("%12d|", remoteHit))
		b.WriteString(fmt.Sprintf("%12d|", remoteMiss))
		b.WriteString(fmt.Sprintf("%12s|", "-"))
		b.WriteString(fmt.Sprintf("%12s|", "-"))
		b.WriteString("\n")
	}
	printSepLine(&b, title)
	// -----Rows end------

	logger.Info(b.String())
}

func printSepLine(b *strings.Builder, title string) {
	for _, c := range title {
		if c == '|' {
			b.WriteString("+")
		} else {
			b.WriteString("-")
		}
	}
	b.WriteString("\n")
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
