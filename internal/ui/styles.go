package ui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#96D0FF")).
			PaddingLeft(1)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#96D0FF")).
			Bold(true).
			PaddingLeft(2).
			PaddingRight(1)

	statusEnabledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#04B575")).
				Bold(true)

	statusDisabledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5555")).
				Bold(true)

	statLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#999999"))

	statValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#96D0FF")).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			PaddingLeft(1).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			PaddingLeft(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Padding(0, 1)

	barFilledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	barEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))
)
