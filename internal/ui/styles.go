package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("2")
	colorRed    = lipgloss.Color("1")
	colorYellow = lipgloss.Color("3")
	colorGray   = lipgloss.Color("8")
	colorWhite  = lipgloss.Color("15")

	styleActive   = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	styleInactive = lipgloss.NewStyle().Foreground(colorGray)
	styleError    = lipgloss.NewStyle().Foreground(colorRed)
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorWhite)
	styleHelp     = lipgloss.NewStyle().Foreground(colorGray)
	styleBorder   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorGray).Padding(0, 1)
	styleSelected = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
)
