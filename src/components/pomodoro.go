package components

import (
	"fmt"
	"strings"
	"time"

	"terminal-dashboard/theme"

	"github.com/charmbracelet/lipgloss"
)

func SavePomodoroCount(count int) {}

func pomodorProgressBar(progress float64, width int) string {
	if width < 2 {
		return ""
	}
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	bar := lipgloss.NewStyle().Foreground(theme.GlobalTheme.Active).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(theme.GlobalTheme.StatBarBg).Render(strings.Repeat("░", empty))
	return bar
}

func RenderPomodoro(box lipgloss.Style, pomodoroActive bool, pomodoroPaused bool, remaining time.Duration, count int, elapsed time.Duration, width int) string {
	title := theme.TitleStyle.Render("🍅 Pomodoro")

	var statusStyle lipgloss.Style
	var statusText string
	if pomodoroActive {
		statusStyle = lipgloss.NewStyle().Foreground(theme.GlobalTheme.Active).Bold(true)
		statusText = "● RUNNING"
	} else if pomodoroPaused {
		statusStyle = lipgloss.NewStyle().Foreground(theme.GlobalTheme.Accent1).Bold(true)
		statusText = "⏸ PAUSED"
	} else {
		statusStyle = lipgloss.NewStyle().Foreground(theme.GlobalTheme.Border)
		statusText = "◎  READY"
	}

	timerColor := theme.GlobalTheme.Border
	if pomodoroActive {
		timerColor = theme.GlobalTheme.Active
	} else if pomodoroPaused {
		timerColor = theme.GlobalTheme.Accent1
	}
	timerStr := fmt.Sprintf("%02d:%02d", int(remaining.Minutes()), int(remaining.Seconds())%60)
	timer := lipgloss.NewStyle().Foreground(timerColor).Bold(true).Render(timerStr)

	progress := 0.0
	if pomodoroActive || pomodoroPaused {
		progress = elapsed.Seconds() / (25 * 60)
		if progress > 1 {
			progress = 1
		}
	}

	bar := pomodorProgressBar(progress, width)

	content := title + "\n\n" +
		statusStyle.Render(statusText) + "\n\n" +
		timer + "\n\n" +
		bar
		

	return box.Render(content)
}
