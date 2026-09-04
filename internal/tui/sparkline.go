package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var blocks = []rune("▁▂▃▄▅▆▇█")

// renderSparkline draws samples as a block-character bar graph scaled to
// the max value seen, plus a current-rate readout.
func renderSparkline(samples []float64, width int, currentPPS float64) string {
	if width < 10 {
		width = 10
	}
	graphWidth := width - 2
	if graphWidth < 1 {
		graphWidth = 1
	}

	var trimmed []float64
	if len(samples) > graphWidth {
		trimmed = samples[len(samples)-graphWidth:]
	} else {
		trimmed = samples
	}

	max := 0.0
	for _, v := range trimmed {
		if v > max {
			max = v
		}
	}

	var b strings.Builder
	pad := graphWidth - len(trimmed)
	for i := 0; i < pad; i++ {
		b.WriteRune(' ')
	}
	for _, v := range trimmed {
		if max <= 0 {
			b.WriteRune(blocks[0])
			continue
		}
		idx := int((v / max) * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}

	graph := lipgloss.NewStyle().Foreground(colorAccent).Render(b.String())
	readout := subtitleStyle.Render(fmt.Sprintf("%.0f pps (peak %.0f)", currentPPS, max))
	return graph + "\n" + readout
}
