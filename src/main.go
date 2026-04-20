package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminal-dashboard/components"
	"terminal-dashboard/theme"
)

var (
	GlobalTheme = theme.GlobalTheme
)

type Config struct {
	NotionAPIKey   string `json:"notion_api_key"`
	NotionDatabase string `json:"notion_database_id"`
}

type model struct {
	width       int
	height      int
	time        time.Time
	cpu         float64
	ram         float64
	disk        float64
	weather     string
	lastWeather time.Time
	news        []string
	lastNews    time.Time
	tasks       []components.NotionTask
	cursor      int
	lastTasks   time.Time
	notionKey   string
	notionDB    string
	err         error
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		components.FetchStats(),
		components.FetchWeather(),
		components.FetchNews(),
		components.FetchTasks(m.notionKey, m.notionDB),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if len(m.tasks) > 0 && m.tasks[m.cursor].ID != "" {
				return m, components.MarkTaskDone(m.tasks[m.cursor].ID, m.notionKey, m.cursor)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.time = time.Time(msg)

		var cmds []tea.Cmd
		cmds = append(cmds, tick(), components.FetchStats())

		if time.Since(m.lastNews) > time.Hour {
			m.lastNews = time.Now()
			cmds = append(cmds, components.FetchNews())
		}

		if time.Since(m.lastWeather) > 2*time.Hour {
			m.lastWeather = time.Now()
			cmds = append(cmds, components.FetchWeather())
		}

		if time.Since(m.lastTasks) > 5*time.Minute {
			m.lastTasks = time.Now()
			cmds = append(cmds, components.FetchTasks(m.notionKey, m.notionDB))
		}

		return m, tea.Batch(cmds...)

	case components.StatsMsg:
		m.cpu = msg.Cpu
		m.ram = msg.Ram
		m.disk = msg.Disk
		return m, nil

	case components.WeatherMsg:
		m.weather = string(msg)
		return m, nil

	case components.NewsMsg:
		m.news = msg
		m.lastNews = time.Now()
		return m, nil

	case components.TasksMsg:
		m.tasks = msg
		m.cursor = 0
		m.err = nil
		m.lastTasks = time.Now()
		return m, nil

	case components.TaskDoneMsg:
		if msg.Index >= 0 && msg.Index < len(m.tasks) {
			m.tasks = append(m.tasks[:msg.Index], m.tasks[msg.Index+1:]...)
			if m.cursor >= len(m.tasks) && m.cursor > 0 {
				m.cursor--
			}
			if len(m.tasks) == 0 {
				m.tasks = []components.NotionTask{{Label: "No upcoming tasks!"}}
			}
		}
		return m, nil

	case components.TaskErrMsg:
		m.err = msg.Err
		return m, nil

	case components.WeatherErrMsg:
		m.weather = "Weather unavailable right now"
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	header := components.RenderHeader(m.time, m.weather)

	colWidth := (m.width - 6) / 2
	innerWidth := colWidth - 6
	innerWidth = max(innerWidth, 10)
	sized := theme.BoxStyle.Width(colWidth)

	statsBox := components.RenderStats(sized, m.cpu, m.ram, m.disk, innerWidth)
	newsBox := components.RenderNews(sized, m.news, innerWidth)
	tasksBox := components.RenderTasks(sized, m.tasks, m.cursor, innerWidth)

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, statsBox, newsBox)
	grid := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, tasksBox)

	statusBar := ""
	if m.err != nil {
		statusBar = lipgloss.NewStyle().
			Foreground(GlobalTheme.Error).
			Width(m.width).
			Render("  ✗ " + m.err.Error())
	}

	finalUI := lipgloss.JoinVertical(lipgloss.Left, header, grid, statusBar)

	return theme.AppStyle.Render(finalUI)
}

func loadOrSetupConfig() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("✗ Error getting home directory:", err)
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".config", "ter_dash")
	configPath := filepath.Join(configDir, "config.json")

	var cfg Config

	// Try to read existing config
	data, err := os.ReadFile(configPath)
	if err == nil {
		if json.Unmarshal(data, &cfg) == nil && cfg.NotionAPIKey != "" && cfg.NotionDatabase != "" {
			return cfg // Config exists and is valid
		}
	}

	// If we reach here, we need to run the initial setup
	fmt.Println("✨ First time setup: Please enter your Notion credentials.")
	fmt.Println("These will be saved securely to:", configPath)
	fmt.Println(strings.Repeat("-", 50))

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Notion API Key: ")
	apiKey, _ := reader.ReadString('\n')
	cfg.NotionAPIKey = strings.TrimSpace(apiKey)

	fmt.Print("Notion Database ID: ")
	dbID, _ := reader.ReadString('\n')
	cfg.NotionDatabase = strings.TrimSpace(dbID)

	// Create directory and save file
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Println("✗ Error creating config directory:", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(cfg, "", "  ")
	// Using 0600 permissions so only your user account can read the secrets
	if err := os.WriteFile(configPath, out, 0600); err != nil {
		fmt.Println("✗ Error saving config file:", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration saved! Starting dashboard...")
	time.Sleep(1 * time.Second) // Brief pause so the user sees the success message

	return cfg
}

func main() {
	cfg := loadOrSetupConfig()
	initialModel := model{
		time:      time.Now(),
		weather:   "Fetching weather...",
		notionKey: cfg.NotionAPIKey,
        notionDB:  cfg.NotionDatabase,
    }

	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
