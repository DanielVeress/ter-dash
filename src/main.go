package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/beeep"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"terminal-dashboard/components"
	"terminal-dashboard/theme"
)

//go:embed assets/announcer_victory.wav
var pomodoroSound []byte

//go:embed assets/damit.wav
var breakSound []byte

func playSound(data []byte) {
	for _, player := range []string{"paplay", "aplay"} {
		cmd := exec.Command(player, "-")
		cmd.Stdin = bytes.NewReader(data)
		if err := cmd.Run(); err == nil {
			return
		}
	}
}

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
	tasks               []components.NotionTask
	taskCacheTop        []components.NotionTask
	taskCacheHigh       []components.NotionTask
	taskCacheNoPriority []components.NotionTask
	currentPriority     string
	cursor              int
	lastTasks           time.Time
	notionKey           string
	notionDB            string
	pomodoroActive             bool
	pomodoroPaused             bool
	pomodoroStart              time.Time
	pomodoroElapsedBeforePause time.Duration
	pomodoroCount              int
	pomodoroDate               string
	pomodoroHistory            map[string]int
	breakActive                bool
	breakStart                 time.Time
	showHelp    bool
	hideAux     bool
	err         error
}

type tickMsg time.Time

func sliceCopy(tasks []components.NotionTask) []components.NotionTask {
	result := make([]components.NotionTask, len(tasks))
	copy(result, tasks)
	return result
}

func removeTaskByID(tasks []components.NotionTask, id string) []components.NotionTask {
	for i := range tasks {
		if tasks[i].ID == id {
			return append(tasks[:i], tasks[i+1:]...)
		}
	}
	return tasks
}

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
		components.FetchTasksByPriority(m.notionKey, m.notionDB, "Top"),
		components.FetchTasksByPriority(m.notionKey, m.notionDB, "High"),
		components.FetchTasksByPriority(m.notionKey, m.notionDB, "No Priority"),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "h":
			m.hideAux = !m.hideAux
			return m, nil
		case "j", "down":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "p":
			if m.pomodoroPaused {
				m.pomodoroStart = time.Now()
				m.pomodoroPaused = false
				m.pomodoroActive = true
			} else if m.pomodoroActive {
				m.pomodoroElapsedBeforePause = time.Since(m.pomodoroStart)
				m.pomodoroActive = false
				m.pomodoroPaused = true
			} else {
				m.pomodoroElapsedBeforePause = 0
				m.pomodoroStart = time.Now()
				m.pomodoroActive = true
			}
		case "b":
			if !m.pomodoroActive && !m.pomodoroPaused && !m.breakActive {
				m.breakActive = true
				m.breakStart = time.Now()
			}
		case "s":
			m.pomodoroActive = false
			m.pomodoroPaused = false
			m.pomodoroElapsedBeforePause = 0
			m.breakActive = false
		case "tab":
			switch m.currentPriority {
			case "Top":
				m.currentPriority = "High"
				m.tasks = components.FilterUrgentOrAll(m.taskCacheHigh)
			case "High":
				m.currentPriority = "No Priority"
				m.tasks = components.FilterUrgentOrAll(m.taskCacheNoPriority)
			default:
				m.currentPriority = "Top"
				m.tasks = components.FilterUrgentOrAll(m.taskCacheTop)
			}
			m.cursor = 0
			if len(m.tasks) == 0 {
				m.tasks = []components.NotionTask{{Label: "No tasks in this priority."}}
			}
		case "enter":
			if len(m.tasks) > 0 && m.tasks[m.cursor].ID != "" {
				m.tasks[m.cursor].IsPending = true
				return m, components.MarkTaskDone(m.tasks[m.cursor].ID, m.notionKey)
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
			cmds = append(cmds,
				components.FetchTasksByPriority(m.notionKey, m.notionDB, "Top"),
				components.FetchTasksByPriority(m.notionKey, m.notionDB, "High"),
				components.FetchTasksByPriority(m.notionKey, m.notionDB, "No Priority"),
			)
		}

		// Midnight reset
		today := time.Now().Format("2006-01-02")
		if today != m.pomodoroDate {
			m.pomodoroDate = today
			m.pomodoroCount = 0
			m.pomodoroActive = false
			m.pomodoroPaused = false
			m.pomodoroElapsedBeforePause = 0
			m.breakActive = false
		}

		// Pomodoro
		elapsed := m.pomodoroElapsedBeforePause
		if m.pomodoroActive {
			elapsed += time.Since(m.pomodoroStart)
		}
		if m.pomodoroActive && elapsed > 25*time.Minute {
			m.pomodoroActive = false
			m.pomodoroPaused = false
			m.pomodoroElapsedBeforePause = 0
			m.pomodoroCount++
			components.SavePomodoroCount(m.pomodoroCount)
			m.pomodoroHistory[today] = m.pomodoroCount
			go func() {
				playSound(pomodoroSound)
				beeep.Notify("Pomodoro done!", "Press 'b' to start your break. 🍅", "")
			}()
		}

		if m.breakActive && time.Since(m.breakStart) > 5*time.Minute {
			m.breakActive = false
			m.breakStart = time.Time{}
			go func() {
				playSound(breakSound)
				beeep.Notify("Break over!", "Press 'p' to start a new pomodoro. 🍅", "")
			}()
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

	case components.PriorityTasksMsg:
		switch msg.Priority {
		case "Top":
			m.taskCacheTop = msg.Tasks
		case "High":
			m.taskCacheHigh = msg.Tasks
		case "No Priority":
			m.taskCacheNoPriority = msg.Tasks
		}
		m.err = nil

		if msg.Priority == m.currentPriority {
			var activeTaskID string
			hasActiveTask := false
			if len(m.tasks) > 0 && m.cursor >= 0 && m.cursor < len(m.tasks) {
				activeTaskID = m.tasks[m.cursor].ID
				hasActiveTask = true
			}

			m.tasks = components.FilterUrgentOrAll(msg.Tasks)
			if len(m.tasks) == 0 {
				m.tasks = []components.NotionTask{{Label: "No tasks in this priority."}}
			}
			m.cursor = 0
			if hasActiveTask {
				for i, task := range m.tasks {
					if task.ID == activeTaskID {
						m.cursor = i
						break
					}
				}
			}
		}
		return m, nil

	case components.TaskDoneMsg:
		for i := range m.tasks {
			if m.tasks[i].ID == msg.ID {
				m.tasks[i].IsPending = false
				m.tasks[i].IsJustFinished = true
				break
			}
		}
		return m, func() tea.Msg {
			time.Sleep(800 * time.Millisecond)
			return components.ClearFlashMsg{ID: msg.ID}
		}

	case components.ClearFlashMsg:
		switch m.currentPriority {
		case "Top":
			m.taskCacheTop = removeTaskByID(m.taskCacheTop, msg.ID)
		case "High":
			m.taskCacheHigh = removeTaskByID(m.taskCacheHigh, msg.ID)
		case "No Priority":
			m.taskCacheNoPriority = removeTaskByID(m.taskCacheNoPriority, msg.ID)
		}
		var updatedCache []components.NotionTask
		switch m.currentPriority {
		case "Top":
			updatedCache = m.taskCacheTop
		case "High":
			updatedCache = m.taskCacheHigh
		default:
			updatedCache = m.taskCacheNoPriority
		}
		m.tasks = components.FilterUrgentOrAll(updatedCache)
		if len(m.tasks) == 0 {
			m.tasks = []components.NotionTask{{Label: "No tasks in this priority."}}
		}
		if m.cursor >= len(m.tasks) && m.cursor > 0 {
			m.cursor--
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

type layoutDims struct {
	colWidth   int
	innerWidth int
	halfCol    int
	halfInner  int
	sized      lipgloss.Style
	halfSized  lipgloss.Style
}

func computeDims(totalWidth int) layoutDims {
	colWidth := (totalWidth - 6) / 2
	innerWidth := max(colWidth-6, 10)
	halfCol := (colWidth - 2) / 2
	halfInner := max(halfCol-6, 6)
	return layoutDims{
		colWidth:   colWidth,
		innerWidth: innerWidth,
		halfCol:    halfCol,
		halfInner:  halfInner,
		sized:      theme.BoxStyle.Width(colWidth),
		halfSized:  theme.BoxStyle.Width(halfCol).Height(12),
	}
}

func renderLeftColumn(m model, dims layoutDims, elapsed, remaining time.Duration, breakElapsed time.Duration) string {
	pomStyle := theme.BoxStyle.Width(dims.halfCol)
	statsBox := components.RenderStats(dims.halfSized, m.cpu, m.ram, m.disk, dims.halfInner)
	pomodoroBox := components.RenderPomodoro(pomStyle, m.pomodoroActive, m.pomodoroPaused, remaining, m.pomodoroCount, elapsed, dims.halfInner, m.pomodoroHistory, m.breakActive, breakElapsed)
	newsBox := components.RenderNews(dims.sized, m.news, dims.innerWidth)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, statsBox, pomodoroBox)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, newsBox)
}

func renderRightColumn(m model, dims layoutDims) string {
	return components.RenderTasks(dims.sized, m.tasks, m.cursor, dims.innerWidth, m.currentPriority)
}

func renderFocused(m model, dims layoutDims, elapsed, remaining time.Duration, breakElapsed time.Duration) string {
	pomStyle := theme.BoxStyle.Width(dims.colWidth)
	pomodoroBox := components.RenderPomodoro(pomStyle, m.pomodoroActive, m.pomodoroPaused, remaining, m.pomodoroCount, elapsed, dims.innerWidth, m.pomodoroHistory, m.breakActive, breakElapsed)
	tasksBox := components.RenderTasks(dims.sized, m.tasks, m.cursor, dims.innerWidth, m.currentPriority)
	return lipgloss.JoinHorizontal(lipgloss.Top, pomodoroBox, tasksBox)
}

func renderStatusBar(m model, width int) string {
	if m.err == nil {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(GlobalTheme.Error).
		Width(width).
		Render("  ✗ " + m.err.Error())
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	if m.showHelp {
		return components.RenderHelp(m.width, m.height)
	}
	dims := computeDims(m.width)

	var elapsed, remaining time.Duration
	if m.pomodoroActive {
		elapsed = m.pomodoroElapsedBeforePause + time.Since(m.pomodoroStart)
		remaining = max(25*time.Minute-elapsed, 0)
	} else if m.pomodoroPaused {
		elapsed = m.pomodoroElapsedBeforePause
		remaining = max(25*time.Minute-elapsed, 0)
	} else {
		remaining = 25 * time.Minute
	}

	var breakElapsed time.Duration
	if m.breakActive {
		breakElapsed = time.Since(m.breakStart)
	}

	header := components.RenderHeader(m.time, m.weather)
	var grid string
	if m.hideAux {
		grid = renderFocused(m, dims, elapsed, remaining, breakElapsed)
	} else {
		grid = lipgloss.JoinHorizontal(lipgloss.Top,
			renderLeftColumn(m, dims, elapsed, remaining, breakElapsed),
			renderRightColumn(m, dims),
		)
	}

	rows := []string{header, grid}
	if m.err != nil {
		rows = append(rows, renderStatusBar(m, m.width))
	}

	content := lipgloss.JoinVertical(lipgloss.Center, rows...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top,
		lipgloss.NewStyle().MarginTop(1).Render(content))
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
		time:            time.Now(),
		weather:         "Fetching weather...",
		notionKey:       cfg.NotionAPIKey,
		notionDB:        cfg.NotionDatabase,
		currentPriority: "Top",
		pomodoroCount:   components.LoadTodayPomodoroCount(),
		pomodoroDate:    time.Now().Format("2006-01-02"),
		pomodoroHistory: components.LoadAllPomodoroData(),
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
