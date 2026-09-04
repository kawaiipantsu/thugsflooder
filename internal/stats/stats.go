// Package stats holds the live counters and rate history the TUI renders.
package stats

import (
	"sync"
	"sync/atomic"
	"time"
)

const rateHistorySize = 120 // ~24s of history at 200ms ticks, or 2min at 1s ticks

// Snapshot is an immutable copy of current stats for one TUI render tick.
type Snapshot struct {
	Sent        uint64
	Bytes       uint64
	Blocked     uint64
	Dropped     uint64
	ActiveHosts int
	PerProto    map[string]uint64
	RateSamples []float64 // packets/sec, oldest first
	CurrentPPS  float64
	Elapsed     time.Duration
}

// Stats aggregates counters across all generator workers.
type Stats struct {
	sent    atomic.Uint64
	bytes   atomic.Uint64
	blocked atomic.Uint64
	dropped atomic.Uint64
	started time.Time

	mu       sync.Mutex
	perProto map[string]uint64
	hosts    map[string]struct{}

	rateMu      sync.Mutex
	rateHistory []float64
	windowCount uint64
	windowStart time.Time
}

// New creates a Stats aggregator and starts its internal elapsed-time clock.
func New() *Stats {
	now := time.Now()
	return &Stats{
		started:     now,
		perProto:    make(map[string]uint64),
		hosts:       make(map[string]struct{}),
		windowStart: now,
	}
}

// RecordSent records one successfully sent packet/request.
func (s *Stats) RecordSent(proto, host string, n int) {
	s.sent.Add(1)
	s.bytes.Add(uint64(n))

	s.mu.Lock()
	s.perProto[proto]++
	s.hosts[host] = struct{}{}
	s.mu.Unlock()

	s.rateMu.Lock()
	s.windowCount++
	s.rateMu.Unlock()
}

// RecordBlocked records an attempt rejected by the allowlist check.
func (s *Stats) RecordBlocked() { s.blocked.Add(1) }

// RecordDropped records a send/connect error.
func (s *Stats) RecordDropped() { s.dropped.Add(1) }

// Tick should be called roughly once per second to roll the rate window
// into history; it returns the pps for the window that just closed.
func (s *Stats) Tick() float64 {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	elapsed := time.Since(s.windowStart).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	pps := float64(s.windowCount) / elapsed
	s.rateHistory = append(s.rateHistory, pps)
	if len(s.rateHistory) > rateHistorySize {
		s.rateHistory = s.rateHistory[len(s.rateHistory)-rateHistorySize:]
	}
	s.windowCount = 0
	s.windowStart = time.Now()
	return pps
}

// Snapshot returns a point-in-time copy of all stats.
func (s *Stats) Snapshot() Snapshot {
	s.mu.Lock()
	proto := make(map[string]uint64, len(s.perProto))
	for k, v := range s.perProto {
		proto[k] = v
	}
	activeHosts := len(s.hosts)
	s.mu.Unlock()

	s.rateMu.Lock()
	samples := make([]float64, len(s.rateHistory))
	copy(samples, s.rateHistory)
	var current float64
	if len(samples) > 0 {
		current = samples[len(samples)-1]
	}
	s.rateMu.Unlock()

	return Snapshot{
		Sent:        s.sent.Load(),
		Bytes:       s.bytes.Load(),
		Blocked:     s.blocked.Load(),
		Dropped:     s.dropped.Load(),
		ActiveHosts: activeHosts,
		PerProto:    proto,
		RateSamples: samples,
		CurrentPPS:  current,
		Elapsed:     time.Since(s.started),
	}
}
