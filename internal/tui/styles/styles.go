// Package styles defines shared TUI styles and colors.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#7C3AED")
	SecondaryColor = lipgloss.Color("#A78BFA")
	SuccessColor   = lipgloss.Color("#10B981")
	WarningColor   = lipgloss.Color("#F59E0B")
	ErrorColor     = lipgloss.Color("#EF4444")
	MutedColor     = lipgloss.Color("#6B7280")
	BgColor        = lipgloss.Color("#1F2937")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginBottom(1)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(PrimaryColor).
			Bold(true).
			Padding(0, 1)

	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Padding(0, 1)

	CountStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginTop(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2)

	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(WarningColor).
			Padding(1, 2)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor)

	SpecialMarkerStyle = lipgloss.NewStyle().
				Foreground(WarningColor).
				Bold(true)
)
