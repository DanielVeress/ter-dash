package components

import (
	"terminal-dashboard/theme"

	"github.com/charmbracelet/lipgloss"
)

type binding struct {
	key  string
	desc string
}

var bindings = []binding{
	{"j / ↓", "Move cursor down"},
	{"k / ↑", "Move cursor up"},
	{"enter", "Mark selected task as done"},
	{"p", "Start / pause / resume Pomodoro"},
	{"s", "Stop Pomodoro"},
	{"?", "Toggle this help"},
	{"q / ctrl+c", "Quit"},
}

func RenderHelp(screenWidth, screenHeight int) string {
	keyStyle := lipgloss.NewStyle().
		Foreground(theme.GlobalTheme.Accent1).
		Bold(true).
		Width(16)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.GlobalTheme.TextPrimary)

	rows := ""
	for _, b := range bindings {
		rows += "\n  " + keyStyle.Render(b.key) + descStyle.Render(b.desc)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.GlobalTheme.Accent2).
		Padding(1, 3).
		Render(
			theme.TitleStyle.Render("  Keybindings  ") + "\n" + rows + "\n",
		)

	return lipgloss.Place(screenWidth, screenHeight, lipgloss.Center, lipgloss.Center, box)
}
