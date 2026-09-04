// Package generator implements the traffic-generation workers. Every
// worker in this package checks the allowlist immediately before each
// send (defense in depth on top of the config-load-time validation) and
// reports every attempt to both stats and the mandatory audit log.
package generator

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
	"github.com/thugsred/thugsflooder/internal/recording"
	"github.com/thugsred/thugsflooder/internal/stats"
)

// Marker is prepended to every synthetic payload thugsflooder sends, so
// anyone inspecting captured traffic (or the audit log) can immediately
// identify it as this tool's own test traffic.
const Marker = "THUGSFLOODER-TEST-TRAFFIC"

// Sinks bundles everything a worker needs to report its activity.
type Sinks struct {
	Stats *stats.Stats
	Audit *audit.Logger
	Log   func(format string, args ...any)
}

// RateLimiter caps the combined send rate across all workers at maxPPS.
type RateLimiter struct {
	tokens chan struct{}
	stop   chan struct{}
}

// NewRateLimiter starts a token bucket that admits at most maxPPS sends/sec.
func NewRateLimiter(maxPPS int) *RateLimiter {
	if maxPPS < 1 {
		maxPPS = 1
	}
	rl := &RateLimiter{
		tokens: make(chan struct{}, maxPPS),
		stop:   make(chan struct{}),
	}
	go rl.refill(maxPPS)
	return rl
}

func (rl *RateLimiter) refill(maxPPS int) {
	interval := time.Second / time.Duration(maxPPS)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
			}
		case <-rl.stop:
			return
		}
	}
}

// Wait blocks until a token is available or ctx is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the limiter's refill goroutine.
func (rl *RateLimiter) Close() { close(rl.stop) }

// markedPayload builds a synthetic payload of approximately size bytes,
// always beginning with Marker plus a per-run session id.
func markedPayload(sessionID string, size int) []byte {
	prefix := []byte(fmt.Sprintf("%s|session=%s|", Marker, sessionID))
	if size <= len(prefix) {
		return prefix[:size]
	}
	buf := make([]byte, size)
	copy(buf, prefix)
	_, _ = rand.Read(buf[len(prefix):])
	return buf
}

// logOnce logs the formatted message only if it differs from the last
// message logged through lastLogged, so a sustained run of identical
// failures (e.g. a target that's gone down) doesn't flood the log pane
// with one line per send — the audit log still records every attempt
// regardless.
func logOnce(sinks *Sinks, lastLogged *string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if msg == *lastLogged {
		return
	}
	*lastLogged = msg
	sinks.Log("%s", msg)
}

func newSessionID() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return "fallback"
	}
	return fmt.Sprintf("%x", n)
}

// Manager owns and runs all configured generators until its context is cancelled.
type Manager struct {
	cfg     *config.Config
	allow   *config.Allowlist
	sinks   *Sinks
	limiter *RateLimiter
	session string

	replay      []recording.Entry
	replaySpeed float64
}

// NewManager builds a Manager from a validated config/allowlist and sinks.
func NewManager(cfg *config.Config, allow *config.Allowlist, sinks *Sinks) *Manager {
	return &Manager{
		cfg:     cfg,
		allow:   allow,
		sinks:   sinks,
		limiter: NewRateLimiter(cfg.RateLimit.MaxPPS),
		session: newSessionID(),
	}
}

// WithReplay attaches a loaded recording to be replayed alongside the
// configured junk generators, at the given speed multiplier.
func (m *Manager) WithReplay(entries []recording.Entry, speed float64) *Manager {
	m.replay = entries
	m.replaySpeed = speed
	if m.replaySpeed <= 0 {
		m.replaySpeed = 1
	}
	return m
}

// Session returns this run's session id (also embedded in every payload).
func (m *Manager) Session() string { return m.session }

// Run starts one worker per configured target/port/protocol (plus a
// replay worker if a recording was attached) and blocks until ctx is
// cancelled and every worker has exited.
func (m *Manager) Run(ctx context.Context) {
	defer m.limiter.Close()
	var wg sync.WaitGroup

	workers := m.cfg.RateLimit.WorkersPerTarget
	if workers < 1 {
		workers = 1
	}

	for _, t := range m.cfg.Targets {
		for _, p := range t.Protocols {
			proto := config.Protocol(p)
			for _, port := range t.Ports {
				host, port, proto := t.Host, port, proto
				// A pool of concurrent workers per target:port:protocol —
				// throughput against TCP/HTTP is round-trip-time bound, not
				// CPU bound, so one goroutine per target can't push much
				// volume; N concurrent goroutines can.
				for i := 0; i < workers; i++ {
					switch proto {
					case config.ProtoUDP:
						wg.Add(1)
						go func() { defer wg.Done(); m.udpWorker(ctx, host, port) }()
					case config.ProtoTCP:
						wg.Add(1)
						go func() { defer wg.Done(); m.tcpWorker(ctx, host, port) }()
					case config.ProtoHTTP:
						wg.Add(1)
						go func() { defer wg.Done(); m.httpWorker(ctx, host, port) }()
					}
				}
			}
		}
	}

	if len(m.replay) > 0 {
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); m.replayWorker(ctx) }()
		}
	}

	wg.Wait()
}
