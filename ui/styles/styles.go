package styles

import "github.com/charmbracelet/lipgloss"

var (
	DocStyle = lipgloss.NewStyle().Margin(1, 2)

	TitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginLeft(2)

	PaginationStyle = lipgloss.NewStyle().Padding(0, 1)

	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ViewportStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)
)
