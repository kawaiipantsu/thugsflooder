package generator

import (
	"context"
	"net/http"
	"time"

	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
)

// replayWorker cycles through the attached recording, waiting each entry's
// (speed-scaled) delay, checking the allowlist, and sending its payload.
// Every entry's target must be in the allowlist just like the junk
// generators — a recording cannot be used to reach an unauthorized host.
func (m *Manager) replayWorker(ctx context.Context) {
	if len(m.replay) == 0 {
		return
	}
	client := &http.Client{Timeout: httpTimeout}
	var lastLogged string

	for i := 0; ; i = (i + 1) % len(m.replay) {
		if ctx.Err() != nil {
			return
		}
		e := m.replay[i]

		delay := time.Duration(float64(e.Delay()) / m.replaySpeed)
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		if err := m.limiter.Wait(ctx); err != nil {
			return
		}

		proto := config.Protocol(e.Proto)
		if !m.allow.Allowed(e.Host, e.Port, proto) {
			m.sinks.Stats.RecordBlocked()
			m.sinks.Audit.Record(audit.Entry{Proto: e.Proto, Host: e.Host, Port: e.Port, Result: audit.ResultBlocked, Detail: "replay entry not in allowlist"})
			logOnce(m.sinks, &lastLogged, "replay %s %s:%d: blocked (not in allowlist)", e.Proto, e.Host, e.Port)
			continue
		}

		payload, err := e.Payload()
		if err != nil {
			continue
		}

		var n int
		var sendErr error
		switch proto {
		case config.ProtoUDP:
			n, sendErr = sendUDPOnce(e.Host, e.Port, payload)
		case config.ProtoTCP:
			n, sendErr = sendTCPOnce(e.Host, e.Port, payload)
		case config.ProtoHTTP:
			n, sendErr = sendHTTPOnce(client, e.Host, e.Port, payload, m.session)
		default:
			continue
		}

		if sendErr != nil {
			m.sinks.Stats.RecordDropped()
			m.sinks.Audit.Record(audit.Entry{Proto: e.Proto, Host: e.Host, Port: e.Port, Result: audit.ResultDropped, Detail: sendErr.Error()})
			logOnce(m.sinks, &lastLogged, "replay %s %s:%d: send error: %v", e.Proto, e.Host, e.Port, sendErr)
			continue
		}
		lastLogged = ""
		m.sinks.Stats.RecordSent(e.Proto, e.Host, n)
		m.sinks.Audit.Record(audit.Entry{Proto: e.Proto, Host: e.Host, Port: e.Port, Bytes: n, Result: audit.ResultSent})
	}
}
