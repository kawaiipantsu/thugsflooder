package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent   = lipgloss.Color("205")
	colorMuted    = lipgloss.Color("241")
	colorGood     = lipgloss.Color("42")
	colorWarn     = lipgloss.Color("214")
	colorBad      = lipgloss.Color("203")
	colorBorder   = lipgloss.Color("238")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	boxTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	sentStyle    = lipgloss.NewStyle().Foreground(colorGood)
	blockedStyle = lipgloss.NewStyle().Foreground(colorWarn)
	droppedStyle = lipgloss.NewStyle().Foreground(colorBad)
)
