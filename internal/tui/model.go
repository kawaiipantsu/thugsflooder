// Package tui implements the live flood dashboard: a bandwidth/rate bar
// graph, a scrolling log/status pane, and a summary stats box.
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thugsred/thugsflooder/internal/logbuf"
	"github.com/thugsred/thugsflooder/internal/stats"
)

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// Model is the bubbletea root model for the live flood dashboard.
type Model struct {
	stats     *stats.Stats
	logs      *logbuf.Buffer
	auditPath string
	cancel    context.CancelFunc

	width, height int
	logVP         viewport.Model
	final         stats.Snapshot
}

// New builds the dashboard model. cancel is called when the user quits
// (q/ctrl+c/esc), which stops the generator Manager driving stats/logs.
func New(s *stats.Stats, logs *logbuf.Buffer, auditPath string, cancel context.CancelFunc) Model {
	return Model{
		stats:     s,
		logs:      logs,
		auditPath: auditPath,
		cancel:    cancel,
		logVP:     newLogViewport(80, 10),
	}
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.final = m.stats.Snapshot()
			m.cancel()
			return m, tea.Quit
		}

	case tickMsg:
		// Content/sizing of the log viewport is (re)computed in View()
		// every frame; this tick just drives the redraw.
		return m, tick()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "starting thugsflooder dashboard...\n"
	}

	snap := m.stats.Snapshot()
	boxWidth := m.width - 4
	if boxWidth < 10 {
		boxWidth = 10
	}

	header := titleStyle.Render("thugsflooder") + "  " + subtitleStyle.Render("FLOODING — live status")

	graphBox := boxStyle.Width(boxWidth).Render(
		boxTitleStyle.Render("Bandwidth / rate") + "\n" + renderSparkline(snap.RateSamples, boxWidth-6, snap.CurrentPPS),
	)

	statsBoxRendered := boxStyle.Width(boxWidth).Render(
		boxTitleStyle.Render("Summary") + "\n" + renderStats(snap, m.auditPath),
	)

	footer := footerStyle.Render("q / ctrl+c: stop and print final summary")

	logBoxChrome := boxStyle.Width(boxWidth).Render(boxTitleStyle.Render("Log / status") + "\n")

	// Size the log viewport to exactly fill whatever height is left after
	// every other fixed element, measured from their real rendered height
	// rather than a hardcoded guess (which drifts whenever any box's
	// content changes).
	usedHeight := lipgloss.Height(header) + 1 /* blank line */ +
		lipgloss.Height(graphBox) + 1 +
		lipgloss.Height(logBoxChrome) +
		lipgloss.Height(statsBoxRendered) + 1 +
		lipgloss.Height(footer)
	logHeight := m.height - usedHeight
	if logHeight < 3 {
		logHeight = 3
	}
	vpWidth := boxWidth - 4
	if vpWidth < 10 {
		vpWidth = 10
	}
	m.logVP.Width = vpWidth
	m.logVP.Height = logHeight
	m.logVP.SetContent(renderLogLines(m.logs.Snapshot()))
	m.logVP.GotoBottom()

	logBox := boxStyle.Width(boxWidth).Render(
		boxTitleStyle.Render("Log / status") + "\n" + m.logVP.View(),
	)

	return header + "\n\n" + graphBox + "\n" + logBox + "\n" + statsBoxRendered + "\n" + footer + "\n"
}

// Final returns the stats snapshot taken at the moment the user quit, so
// main can print a final summary after the TUI exits.
func (m Model) Final() stats.Snapshot { return m.final }
