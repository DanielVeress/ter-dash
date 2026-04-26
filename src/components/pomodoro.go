package components

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"terminal-dashboard/theme"

	"github.com/charmbracelet/lipgloss"
)

func pomodoroLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ter_dash", "pomodoro_log.csv"), nil
}

func readPomodoroLog(path string) ([]string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func LoadTodayPomodoroCount() int {
	path, err := pomodoroLogPath()
	if err != nil {
		return 0
	}
	today := time.Now().Format("2006-01-02")
	lines, err := readPomodoroLog(path)
	if err != nil {
		return 0
	}
	for _, line := range lines {
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 2 && parts[0] == today {
			count, _ := strconv.Atoi(parts[1])
			return count
		}
	}
	return 0
}

func SavePomodoroCount(count int) {
	path, err := pomodoroLogPath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}

	today := time.Now().Format("2006-01-02")
	lines, _ := readPomodoroLog(path)

	updated := false
	for i, line := range lines {
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 2 && parts[0] == today {
			lines[i] = fmt.Sprintf("%s,%d", today, count)
			updated = true
			break
		}
	}
	if !updated {
		lines = append(lines, fmt.Sprintf("%s,%d", today, count))
	}

	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	w.Flush()
}

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
