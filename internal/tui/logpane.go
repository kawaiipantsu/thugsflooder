package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/thugsred/thugsflooder/internal/logbuf"
)

// renderLogLines formats log buffer lines for the viewport, newest last
// (viewport auto-scrolls to bottom on update).
func renderLogLines(lines []logbuf.Line) string {
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(l.Time.Format("15:04:05"))
		b.WriteString("  ")
		b.WriteString(l.Text)
	}
	return b.String()
}

func newLogViewport(width, height int) viewport.Model {
	vp := viewport.New(width, height)
	return vp
}
