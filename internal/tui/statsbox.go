package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thugsred/thugsflooder/internal/stats"
)

func renderStats(s stats.Snapshot, auditPath string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s %s\n", subtitleStyle.Render("Elapsed:"), s.Elapsed.Round(1e9))
	fmt.Fprintf(&b, "%s %s   %s %s   %s %s\n",
		subtitleStyle.Render("Sent:"), sentStyle.Render(fmt.Sprintf("%d", s.Sent)),
		subtitleStyle.Render("Blocked:"), blockedStyle.Render(fmt.Sprintf("%d", s.Blocked)),
		subtitleStyle.Render("Dropped:"), droppedStyle.Render(fmt.Sprintf("%d", s.Dropped)),
	)
	fmt.Fprintf(&b, "%s %s   %s %d\n",
		subtitleStyle.Render("Bytes:"), formatBytes(s.Bytes),
		subtitleStyle.Render("Active hosts:"), s.ActiveHosts,
	)

	if len(s.PerProto) > 0 {
		protos := make([]string, 0, len(s.PerProto))
		for p := range s.PerProto {
			protos = append(protos, p)
		}
		sort.Strings(protos)
		parts := make([]string, 0, len(protos))
		for _, p := range protos {
			parts = append(parts, fmt.Sprintf("%s=%d", p, s.PerProto[p]))
		}
		fmt.Fprintf(&b, "%s %s\n", subtitleStyle.Render("Per-protocol:"), strings.Join(parts, "  "))
	}

	fmt.Fprintf(&b, "%s %s\n", subtitleStyle.Render("Audit log:"), auditPath)

	return b.String()
}

func formatBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
