package generator

import (
	"context"
	"net"
	"strconv"

	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
)

const udpPayloadSize = 512

// udpWorker holds one UDP socket open and writes marked junk datagrams to
// it as fast as the shared rate limiter allows, until ctx is cancelled.
func (m *Manager) udpWorker(ctx context.Context, host string, port int) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.Dial("udp", addr)
	if err != nil {
		m.sinks.Stats.RecordDropped()
		m.sinks.Audit.Record(audit.Entry{Proto: "udp", Host: host, Port: port, Result: audit.ResultDropped, Detail: err.Error()})
		m.sinks.Log("udp %s: dial error: %v", addr, err)
		return
	}
	defer conn.Close()

	for {
		if ctx.Err() != nil {
			return
		}
		if err := m.limiter.Wait(ctx); err != nil {
			return
		}
		if !m.allow.Allowed(host, port, config.ProtoUDP) {
			m.sinks.Stats.RecordBlocked()
			m.sinks.Audit.Record(audit.Entry{Proto: "udp", Host: host, Port: port, Result: audit.ResultBlocked})
			continue
		}

		payload := markedPayload(m.session, udpPayloadSize)
		n, err := conn.Write(payload)
		if err != nil {
			m.sinks.Stats.RecordDropped()
			m.sinks.Audit.Record(audit.Entry{Proto: "udp", Host: host, Port: port, Result: audit.ResultDropped, Detail: err.Error()})
			continue
		}
		m.sinks.Stats.RecordSent("udp", host, n)
		m.sinks.Audit.Record(audit.Entry{Proto: "udp", Host: host, Port: port, Bytes: n, Result: audit.ResultSent})
	}
}
