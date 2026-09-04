package generator

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
)

const (
	httpPayloadSize = 1024
	httpTimeout     = 3 * time.Second
)

// httpWorker repeatedly issues marked synthetic HTTP requests against
// host:port to generate application-layer log noise (access logs, WAF
// logs), until ctx is cancelled.
func (m *Manager) httpWorker(ctx context.Context, host string, port int) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	url := "http://" + addr + "/thugsflooder-test/" + m.session
	client := &http.Client{Timeout: httpTimeout}

	var lastLogged string
	for {
		if ctx.Err() != nil {
			return
		}
		if err := m.limiter.Wait(ctx); err != nil {
			return
		}
		if !m.allow.Allowed(host, port, config.ProtoHTTP) {
			m.sinks.Stats.RecordBlocked()
			m.sinks.Audit.Record(audit.Entry{Proto: "http", Host: host, Port: port, Result: audit.ResultBlocked})
			logOnce(m.sinks, &lastLogged, "http %s: blocked (not in allowlist)", addr)
			continue
		}

		body := markedPayload(m.session, httpPayloadSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			m.sinks.Stats.RecordDropped()
			continue
		}
		req.Header.Set("User-Agent", "thugsflooder-synthetic-test-traffic")
		req.Header.Set("X-Thugsflooder", Marker+"|session="+m.session)

		resp, err := client.Do(req)
		if err != nil {
			m.sinks.Stats.RecordDropped()
			m.sinks.Audit.Record(audit.Entry{Proto: "http", Host: host, Port: port, Result: audit.ResultDropped, Detail: err.Error()})
			logOnce(m.sinks, &lastLogged, "http %s: request error: %v", addr, err)
			continue
		}
		resp.Body.Close()
		lastLogged = ""

		m.sinks.Stats.RecordSent("http", host, len(body))
		m.sinks.Audit.Record(audit.Entry{Proto: "http", Host: host, Port: port, Bytes: len(body), Result: audit.ResultSent})
	}
}
