package components

import (
	"bufio"
	"fmt"
	"math"
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

func LoadAllPomodoroData() map[string]int {
	path, err := pomodoroLogPath()
	if err != nil {
		return map[string]int{}
	}
	lines, err := readPomodoroLog(path)
	if err != nil {
		return map[string]int{}
	}
	data := make(map[string]int)
	for _, line := range lines {
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 2 {
			count, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			data[parts[0]] = count
		}
	}
	return data
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

func pomodoroAverage(data map[string]int, days int) float64 {
	var total int
	for i := 1; i <= days; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		total += data[date]
	}
	return float64(total) / float64(days)
}

var heatmapColors = []lipgloss.Color{
	"#313244", // 0: empty   (Catppuccin Surface0)
	"#1A4731", // 1: 1-2     (dark green)
	"#2A7A50", // 2: 3-4     (medium green)
	"#A6E3A1", // 3: 5-7     (Catppuccin Green)
	"#94E2D5", // 4: 8+      (Catppuccin Teal)
}

func heatLevel(count int) int {
	switch {
	case count <= 0:
		return 0
	case count <= 2:
		return 1
	case count <= 4:
		return 2
	case count <= 7:
		return 3
	default:
		return 4
	}
}

func renderHeatmap(data map[string]int) string {
	today := time.Now()
	dayLabels := []string{"M", "T", "W", "T", "F", "S", "S"}

	todayWeekday := int(today.Weekday())       // 0=Sun … 6=Sat
	mondayOffset := (todayWeekday - 1 + 7) % 7 // days since last Monday
	thisWeekMonday := today.AddDate(0, 0, -mondayOffset)

	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))

	rows := make([]string, 7)
	for row := range 7 {
		rows[row] = mutedStyle.Render(dayLabels[row]) + " "
		for col := range 4 {
			weeksAgo := 3 - col
			target := thisWeekMonday.AddDate(0, 0, row-weeksAgo*7)

			var block string
			if target.After(today) {
				block = lipgloss.NewStyle().Foreground(heatmapColors[0]).Render("█")
			} else {
				count := data[target.Format("2006-01-02")]
				block = lipgloss.NewStyle().Foreground(heatmapColors[heatLevel(count)]).Render("█")
			}

			if col < 3 {
				rows[row] += block + " "
			} else {
				rows[row] += block
			}
		}
	}
	return strings.Join(rows, "\n")
}

func pomodorProgressBar(progress float64, width int) string {
	if width < 2 {
		return ""
	}
	filled := min(int(progress*float64(width)), width)
	empty := width - filled
	bar := lipgloss.NewStyle().Foreground(theme.GlobalTheme.Active).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(theme.GlobalTheme.StatBarBg).Render(strings.Repeat("░", empty))
	return bar
}

func RenderPomodoro(box lipgloss.Style, pomodoroActive bool, pomodoroPaused bool, remaining time.Duration, count int, elapsed time.Duration, width int, history map[string]int) string {
	title := theme.TitleStyle.Render("🍅 Pomodoro")

	var statusStyle lipgloss.Style
	var statusText string
	if pomodoroActive {
		statusStyle = lipgloss.NewStyle().Foreground(theme.GlobalTheme.Active).Bold(true)
		statusText = "RUNNING"
	} else if pomodoroPaused {
		statusStyle = lipgloss.NewStyle().Foreground(theme.GlobalTheme.Accent1).Bold(true)
		statusText = "PAUSED"
	} else {
		statusStyle = lipgloss.NewStyle().Foreground(theme.GlobalTheme.Border).Bold(true)
		statusText = "READY"
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

	avg := pomodoroAverage(history, 14)
	borderColor := theme.GlobalTheme.Border
	if count > 0 || avg > 0 {
		rounded := math.Round(avg)
		if float64(count) > rounded {
			borderColor = theme.GlobalTheme.Warning // gold: beating your average
		} else if float64(count) >= rounded {
			borderColor = theme.GlobalTheme.Success // green: meeting your average
		}
	}
	box = box.BorderForeground(borderColor)

	heatLabel := lipgloss.NewStyle().Foreground(theme.GlobalTheme.TextMuted).Render("28 Days")
	heatmap := renderHeatmap(history)

	rightBlock := lipgloss.JoinVertical(lipgloss.Left,
		heatLabel,
		heatmap,
	)

	infoContent := lipgloss.JoinVertical(lipgloss.Center,
		statusStyle.Render(statusText),
		"", // Spacer between status and time
		timer,
	)

	rightWidth := lipgloss.Width(rightBlock)
	rightHeight := lipgloss.Height(rightBlock)
	titleHeight := lipgloss.Height(title)

	leftSpaceWidth := width - rightWidth
	if leftSpaceWidth < 0 {
		leftSpaceWidth = 0
	}

	infoContainerHeight := rightHeight - titleHeight
	if infoContainerHeight < 1 {
		infoContainerHeight = 5 // Fallback safeguard
	}

	centeredInfo := lipgloss.Place(
		leftSpaceWidth,
		infoContainerHeight,
		lipgloss.Center, // Horizontal Center
		lipgloss.Center, // Vertical Center
		infoContent,
	)

	leftBlock := lipgloss.JoinVertical(lipgloss.Left,
		title,
		centeredInfo,
	)

	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock)

	content := lipgloss.JoinVertical(lipgloss.Left,
		topSection,
		"", // Spacer before the progress bar
		bar,
	)

	return box.Render(content)
}