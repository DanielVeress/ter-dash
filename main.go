package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().Margin(1, 2)
	
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 2).
		Width(40).
		Height(10)
		
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(lipgloss.Color("#25A065")).
		Padding(0, 1).
		Bold(true)
		
	asciiStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF7CCB")).
		Bold(true).
		MarginRight(4)
)

type model struct{
	time time.Time
}
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {	
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
				case "ctrl+c", "q":
					return m, tea.Quit
				}	
		case tickMsg:
			m.time = time.Time(msg)
			return m, tick()
	}
	return m, nil
}

func (m model) View() string {
	asciiArt := `
  _|_|_|_|_|                    _|_|_|                          _|      
      _|      _|_|    _|  _|_|  _|    _|    _|_|_|    _|_|_|    _|_|_|  
      _|    _|_|_|_|  _|_|      _|    _|  _|    _|  _|_|        _|    _|
      _|    _|        _|        _|    _|  _|    _|      _|_|    _|    _|
      _|      _|_|_|  _|        _|_|_|      _|_|_|  _|_|_|      _|    _|`
	
	// Time
	currentTime := m.time.Format("Monday, 02 Jan 2006 | 15:04")
	season := "🌸 Spring" 
	
	headerInfo := fmt.Sprintf("%s\n%s\n\nPress 'q' to quit", currentTime, season)
	
	// Join the ASCII art and the Time info side-by-side
	header := lipgloss.JoinHorizontal(lipgloss.Top, asciiStyle.Render(asciiArt), headerInfo)

	// -- Dashboard Quadrants --
	weatherBox := boxStyle.Render(
		titleStyle.Render("⛅ Weather (Uppsala)") + "\n\nLoading wttr.in...",
	)
	statsBox := boxStyle.Render(
		titleStyle.Render("💻 System Stats") + "\n\nCPU: --%\nRAM: --%\nDisk: --%",
	)
	newsBox := boxStyle.Render(
		titleStyle.Render("📰 News") + "\n\nFetching articles...",
	)
	tasksBox := boxStyle.Render(
		titleStyle.Render("✅ Notion Tasks") + "\n\nLoading tasks...",
	)

	// -- Layout Construction --
	// Stack the boxes vertically to create columns
	leftColumn 	:= lipgloss.JoinVertical(lipgloss.Left, statsBox, newsBox)
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, weatherBox, tasksBox)
	grid 		:= lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
	finalUI 	:= lipgloss.JoinVertical(lipgloss.Left, header, grid)
	
	// Return the whole thing wrapped in our app-level margins
	return appStyle.Render(finalUI)
}

func main() {
	initialModel := model{
        time: time.Now(),
    }		

	p := tea.NewProgram(initialModel, tea.WithAltScreen()) 
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}