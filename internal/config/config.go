// Package config loads and validates the mandatory target allowlist.
// thugsflooder never sends traffic to anything outside this allowlist.
package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Protocol identifies a traffic-generator protocol.
type Protocol string

const (
	ProtoUDP    Protocol = "udp"
	ProtoTCP    Protocol = "tcp"
	ProtoHTTP   Protocol = "http"
	ProtoReplay Protocol = "replay"
)

// Target is one allowlisted host, with the ports/protocols permitted on it.
type Target struct {
	Host      string   `yaml:"host"`
	Ports     []int    `yaml:"ports"`
	Protocols []string `yaml:"protocols"`
}

// RateLimit caps total traffic across all generators and controls how many
// concurrent workers hammer each target so throughput isn't limited to one
// goroutine's serial connect/send/round-trip loop per target.
type RateLimit struct {
	MaxPPS           int `yaml:"max_pps"`
	WorkersPerTarget int `yaml:"workers_per_target"`
}

// DefaultWorkersPerTarget is used when workers_per_target is unset/zero.
const DefaultWorkersPerTarget = 8

// Config is the full allowlist configuration.
type Config struct {
	Targets   []Target  `yaml:"targets"`
	RateLimit RateLimit `yaml:"rate_limit"`
	AuditLog  string    `yaml:"audit_log"`
}

// allowKey identifies one authorized host:port:protocol tuple.
type allowKey struct {
	host  string
	port  int
	proto Protocol
}

// Allowlist is the validated, queryable form of Config's targets, used by
// every generator at send-time as a defense-in-depth check.
type Allowlist struct {
	set   map[allowKey]struct{}
	hosts map[string]struct{}
}

// Allowed reports whether host:port:proto is explicitly authorized.
func (a *Allowlist) Allowed(host string, port int, proto Protocol) bool {
	_, ok := a.set[allowKey{host: host, port: port, proto: proto}]
	return ok
}

// Hosts returns the distinct set of allowlisted hosts.
func (a *Allowlist) Hosts() []string {
	out := make([]string, 0, len(a.hosts))
	for h := range a.hosts {
		out = append(out, h)
	}
	return out
}

// Load reads, parses, and validates a YAML allowlist config file.
func Load(path string) (*Config, *Allowlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("parsing config %q: %w", path, err)
	}

	allow, err := cfg.validate()
	if err != nil {
		return nil, nil, err
	}
	return &cfg, allow, nil
}

func (c *Config) validate() (*Allowlist, error) {
	if len(c.Targets) == 0 {
		return nil, fmt.Errorf("config has no targets: thugsflooder refuses to run with an empty allowlist")
	}
	if strings.TrimSpace(c.AuditLog) == "" {
		return nil, fmt.Errorf("audit_log path must be set")
	}
	if isNullSink(c.AuditLog) {
		return nil, fmt.Errorf("audit_log cannot be a null device or discard path (%q): the audit trail is mandatory", c.AuditLog)
	}
	if c.RateLimit.MaxPPS <= 0 {
		return nil, fmt.Errorf("rate_limit.max_pps must be a positive integer")
	}
	if c.RateLimit.WorkersPerTarget <= 0 {
		c.RateLimit.WorkersPerTarget = DefaultWorkersPerTarget
	}

	allow := &Allowlist{
		set:   make(map[allowKey]struct{}),
		hosts: make(map[string]struct{}),
	}

	for i, t := range c.Targets {
		host := strings.TrimSpace(t.Host)
		if host == "" {
			return nil, fmt.Errorf("targets[%d]: host is empty", i)
		}
		if net.ParseIP(host) == nil {
			// Not a literal IP; require it at least look like a hostname.
			if _, err := net.LookupHost(host); err != nil {
				return nil, fmt.Errorf("targets[%d]: host %q is not a valid IP and does not resolve: %w", i, host, err)
			}
		}
		if len(t.Ports) == 0 {
			return nil, fmt.Errorf("targets[%d] (%s): no ports listed", i, host)
		}
		if len(t.Protocols) == 0 {
			return nil, fmt.Errorf("targets[%d] (%s): no protocols listed", i, host)
		}

		var protos []Protocol
		for _, p := range t.Protocols {
			proto := Protocol(strings.ToLower(strings.TrimSpace(p)))
			switch proto {
			case ProtoUDP, ProtoTCP, ProtoHTTP, ProtoReplay:
			default:
				return nil, fmt.Errorf("targets[%d] (%s): unknown protocol %q", i, host, p)
			}
			protos = append(protos, proto)
		}

		for _, port := range t.Ports {
			if port < 1 || port > 65535 {
				return nil, fmt.Errorf("targets[%d] (%s): port %d out of range", i, host, port)
			}
			for _, proto := range protos {
				allow.set[allowKey{host: host, port: port, proto: proto}] = struct{}{}
			}
		}
		allow.hosts[host] = struct{}{}
	}

	return allow, nil
}

// isNullSink rejects attempts to configure the audit log as a no-op.
func isNullSink(path string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	switch clean {
	case "/dev/null", "/dev/zero", "nul", "NUL":
		return true
	}
	return false
}
