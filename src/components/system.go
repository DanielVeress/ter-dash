package components

import (
	"fmt"
	"strings"
	"terminal-dashboard/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type StatsMsg struct {
	Cpu  float64
	Ram  float64
	Disk float64
}

func drawProgressBar(percent float64, width int, color lipgloss.Color) string {
	filledWidth := int((percent / 100.0) * float64(width))
	if filledWidth > width {
		filledWidth = width
	}
	emptyWidth := width - filledWidth

	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", emptyWidth)

	bar := lipgloss.NewStyle().Foreground(color).Render(filled) +
		lipgloss.NewStyle().Foreground(theme.GlobalTheme.StatBarBg).Render(empty)

	return fmt.Sprintf("%5.1f%% %s", percent, bar)
}

func FetchStats() tea.Cmd {
	return func() tea.Msg {
		c, _ := cpu.Percent(0, false)
		v, _ := mem.VirtualMemory()
		d, _ := disk.Usage("/")

		cpuVal := 0.0
		if len(c) > 0 {
			cpuVal = c[0]
		}

		return StatsMsg{
			Cpu:  cpuVal,
			Ram:  v.UsedPercent,
			Disk: d.UsedPercent,
		}
	}
}

func RenderStats(sized lipgloss.Style, cpu float64, ram float64, disk float64, innerWidth int) string {
	labelStyle := lipgloss.NewStyle().Foreground(theme.GlobalTheme.TextMuted)

	statsContent := "\n"
	statsContent += "\n" + labelStyle.Render("CPU:  ") + drawProgressBar(cpu, innerWidth, theme.GlobalTheme.StatBarFg)
	statsContent += "\n" + labelStyle.Render("RAM:  ") + drawProgressBar(ram, innerWidth, theme.GlobalTheme.StatBarFg)
	statsContent += "\n" + labelStyle.Render("Disk: ") + drawProgressBar(disk, innerWidth, theme.GlobalTheme.StatBarFg)
	statsBox := sized.Render(
		theme.TitleStyle.Render("💻 System Stats") + statsContent,
	)

	return statsBox
}
