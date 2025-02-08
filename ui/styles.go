package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Existing color definitions (None in this case)

	// Theme colors
	ThemePrimary    = lipgloss.AdaptiveColor{Light: "#1E88E5", Dark: "#64B5F6"}
	ThemeSecondary  = lipgloss.AdaptiveColor{Light: "#FFA000", Dark: "#FFD54F"}
	ThemeAccent     = lipgloss.AdaptiveColor{Light: "#43A047", Dark: "#81C784"}
	ThemeText       = lipgloss.AdaptiveColor{Light: "#212121", Dark: "#FFFFFF"}
	ThemeBackground = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#121212"}
)

var (
	// Existing style definitions (None in this case)

	DocStyle = lipgloss.NewStyle().
			Padding(1, 2, 1, 2).
			Background(ThemeBackground)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ThemePrimary).
			Bold(true).
			Padding(1, 2)

	MenuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ThemeSecondary).
			Padding(1, 1).
			MarginRight(2)

	StatusMessageStyle = lipgloss.NewStyle().
				Foreground(ThemeText)

	ErrorMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	SuccessMessageStyle = lipgloss.NewStyle().
				Foreground(ThemeAccent).
				Bold(true)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(ThemePrimary).
				Bold(true)

	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(ThemeText)
)
