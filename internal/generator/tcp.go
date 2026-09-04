package generator

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
)

const (
	tcpPayloadSize = 512
	tcpDialTimeout = 2 * time.Second
)

// tcpWorker repeatedly opens a fresh TCP connection, writes a marked junk
// payload, and closes it — simulating connection-flood-style noise —
// until ctx is cancelled.
func (m *Manager) tcpWorker(ctx context.Context, host string, port int) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	for {
		if ctx.Err() != nil {
			return
		}
		if err := m.limiter.Wait(ctx); err != nil {
			return
		}
		if !m.allow.Allowed(host, port, config.ProtoTCP) {
			m.sinks.Stats.RecordBlocked()
			m.sinks.Audit.Record(audit.Entry{Proto: "tcp", Host: host, Port: port, Result: audit.ResultBlocked})
			continue
		}

		conn, err := net.DialTimeout("tcp", addr, tcpDialTimeout)
		if err != nil {
			m.sinks.Stats.RecordDropped()
			m.sinks.Audit.Record(audit.Entry{Proto: "tcp", Host: host, Port: port, Result: audit.ResultDropped, Detail: err.Error()})
			continue
		}

		payload := markedPayload(m.session, tcpPayloadSize)
		n, err := conn.Write(payload)
		conn.Close()
		if err != nil {
			m.sinks.Stats.RecordDropped()
			m.sinks.Audit.Record(audit.Entry{Proto: "tcp", Host: host, Port: port, Result: audit.ResultDropped, Detail: err.Error()})
			continue
		}
		m.sinks.Stats.RecordSent("tcp", host, n)
		m.sinks.Audit.Record(audit.Entry{Proto: "tcp", Host: host, Port: port, Bytes: n, Result: audit.ResultSent})
	}
}
