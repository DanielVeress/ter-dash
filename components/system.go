package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type StatsMsg struct {
	Cpu  float64
	Ram  float64
	Disk float64
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