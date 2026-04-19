package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Theme struct {
	Border     lipgloss.Color
	TitleFg    lipgloss.Color
	TitleBg    lipgloss.Color
	AsciiArt   lipgloss.Color
	Error      lipgloss.Color
	StatBarBg  lipgloss.Color
	StatBarFg  lipgloss.Color
}

var currentTheme = Theme{
    Border:    lipgloss.Color("#4C4F69"),
    TitleFg:   lipgloss.Color("#EFF1F5"),
    TitleBg:   lipgloss.Color("#1E66F5"),
    AsciiArt:  lipgloss.Color("#8839EF"),
    Error:     lipgloss.Color("#D20F39"),
	StatBarBg: lipgloss.Color("#444444"),
	StatBarFg: lipgloss.Color("#A6E3A1"),
}

type Config struct {
	NotionAPIKey   string `json:"notion_api_key"`
	NotionDatabase string `json:"notion_database_id"`
}

var (
	appStyle = lipgloss.NewStyle().Margin(1, 2)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(currentTheme.Border).
			Padding(1, 2).
			Align(lipgloss.Left)

	titleStyle = lipgloss.NewStyle().
			Foreground(currentTheme.TitleFg).
			Background(currentTheme.TitleBg).
			Padding(0, 1).
			Bold(true)

	asciiStyle = lipgloss.NewStyle().
			Foreground(currentTheme.AsciiArt).
			Bold(true).
			MarginRight(4)

	selectedTaskStyle = lipgloss.NewStyle().
				Foreground(currentTheme.TitleFg).
				Background(currentTheme.Border).
				Bold(true)
)

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
	tasks       []notionTask
	cursor      int
	lastTasks   time.Time
	notionKey   string
	notionDB    string
	err         error
}

type tickMsg time.Time
type statsMsg struct {
	cpu  float64
	ram  float64
	disk float64
}
type weatherMsg string
type errMsg error
type NewsItem struct {
	Title string `xml:"title"`
}
type Rss struct {
	Items []NewsItem `xml:"channel>item"`
}
type newsMsg []string

type notionTask struct {
	id    string
	label string
}

type NotionResponse struct {
	Results []NotionPage `json:"results"`
}
type NotionProperty struct {
	Type  string `json:"type"`
	Title []struct {
		PlainText string `json:"plain_text"`
	} `json:"title"`
	Date *struct {
		Start string `json:"start"`
	} `json:"date"`
}
type NotionPage struct {
	ID         string                     `json:"id"`
	Properties map[string]NotionProperty  `json:"properties"`
}
type tasksMsg []notionTask
type taskErrMsg struct{ err error }
type taskDoneMsg struct{ index int }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchStats() tea.Cmd {
	return func() tea.Msg {
		c, _ := cpu.Percent(0, false)
		v, _ := mem.VirtualMemory()
		d, _ := disk.Usage("/")

		cpuVal := 0.0
		if len(c) > 0 {
			cpuVal = c[0]
		}

		return statsMsg{
			cpu:  cpuVal,
			ram:  v.UsedPercent,
			disk: d.UsedPercent,
		}
	}
}

func formatWeatherLocation(raw string) string {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return raw
	}
	locs := strings.Split(parts[0], ",")

	if len(locs) >= 3 {
		city := strings.TrimSpace(locs[0])
		country := strings.TrimSpace(locs[len(locs)-1])

		if len(city) > 0 {
			city = strings.ToUpper(string(city[0])) + city[1:]
		}
		country = strings.ToUpper(country)

		return fmt.Sprintf("%s, %s:%s", city, country, parts[1])
	}

	return raw
}

func getSeason(t time.Time) string {
	if t.IsZero() {
		return "..."
	}

	switch t.Month() {
	case time.March, time.April, time.May:
		return "🌸 Spring"
	case time.June, time.July, time.August:
		return "☀️ Summer"
	case time.September, time.October, time.November:
		return "🍂 Autumn"
	case time.December, time.January, time.February:
		return "❄️ Winter"
	default:
		return "🌍 Unknown"
	}
}

func fetchWeather() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}

		res, err := client.Get("https://wttr.in/?format=3")
		if err != nil {
			return errMsg(err)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return errMsg(err)
		}
		weatherStr := strings.TrimSpace(string(body))

		if strings.Contains(weatherStr, "HTML") || weatherStr == "" {
			return errMsg(fmt.Errorf("API unavailable"))
		}

		weatherStr = formatWeatherLocation(weatherStr)

		return weatherMsg(weatherStr)
	}
}

func fetchNews() tea.Cmd {
	return func() tea.Msg {
		sources := map[string]string{
			"NPR":      "https://feeds.npr.org/1001/rss.xml",
			"Guardian": "https://www.theguardian.com/world/rss",
			"Telex":    "https://telex.hu/rss",
			"HVG":      "https://hvg.hu/rss",
		}

		var allHeadlines []string
		client := &http.Client{Timeout: 5 * time.Second}

		for name, url := range sources {
			resp, err := client.Get(url)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			var rss Rss
			if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
				continue
			}

			count := 0
			for _, item := range rss.Items {
				if count >= 2 {
					break
				}
				allHeadlines = append(allHeadlines, fmt.Sprintf("[%s] %s", name, item.Title))
				count++
			}
		}
		return newsMsg(allHeadlines)
	}
}

func fetchTasks(apiKey string, dbID string) tea.Cmd {
	return func() tea.Msg {
        if apiKey == "" || dbID == "" {
            return taskErrMsg{err: fmt.Errorf("Notion credentials not set")}
        }
        url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", dbID)

		payload := []byte(`{
			"page_size": 10,
			"filter": {
				"property": "Status",
				"status": {
					"does_not_equal": "Done"
				}
			},
			"sorts": [
				{
					"property": "Due Date",
					"direction": "ascending"
				}
			]
		}`)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return taskErrMsg{err: fmt.Errorf("error building request: %w", err)}
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskErrMsg{err: fmt.Errorf("error reaching Notion: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return taskErrMsg{err: fmt.Errorf("Notion API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
		}

		var notionResp NotionResponse
		if err := json.NewDecoder(resp.Body).Decode(&notionResp); err != nil {
			return taskErrMsg{err: fmt.Errorf("error parsing Notion response: %w", err)}
		}

		var tasks []notionTask
		for _, page := range notionResp.Results {
			title := "Untitled"
			for _, prop := range page.Properties {
				if prop.Type == "title" && len(prop.Title) > 0 {
					title = prop.Title[0].PlainText
					break
				}
			}

			dateStr := ""
			if dueProp, ok := page.Properties["Due Date"]; ok && dueProp.Date != nil {
				dateStr = fmt.Sprintf(" (Due: %s)", dueProp.Date.Start)
			}

			tasks = append(tasks, notionTask{id: page.ID, label: title + dateStr})
		}

		if len(tasks) == 0 {
			tasks = append(tasks, notionTask{label: "No upcoming tasks!"})
		}

		return tasksMsg(tasks)
	}
}

func markTaskDone(pageID, apiKey string, index int) tea.Cmd {
	return func() tea.Msg {
		payload := []byte(`{"properties":{"Status":{"status":{"name":"Done"}}}}`)
		url := fmt.Sprintf("https://api.notion.com/v1/pages/%s", pageID)

		req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
		if err != nil {
			return taskErrMsg{err: fmt.Errorf("error building request: %w", err)}
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return taskErrMsg{err: fmt.Errorf("error reaching Notion: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return taskErrMsg{err: fmt.Errorf("Notion API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
		}

		return taskDoneMsg{index: index}
	}
}

func drawProgressBar(percent float64, width int, color lipgloss.Color) string {
    filledWidth := int((percent / 100.0) * float64(width))
    if filledWidth > width { filledWidth = width }
    emptyWidth := width - filledWidth

    // Using block characters
    filled := strings.Repeat("█", filledWidth)
    empty := strings.Repeat("░", emptyWidth)

    bar := lipgloss.NewStyle().Foreground(color).Render(filled) + 
           lipgloss.NewStyle().Foreground(currentTheme.StatBarBg).Render(empty)
    
    return fmt.Sprintf("%5.1f%% %s", percent, bar)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tick(),
		fetchStats(),
		fetchWeather(),
		fetchNews(),
		fetchTasks(m.notionKey, m.notionDB),
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
			if len(m.tasks) > 0 && m.tasks[m.cursor].id != "" {
				return m, markTaskDone(m.tasks[m.cursor].id, m.notionKey, m.cursor)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		m.time = time.Time(msg)

		var cmds []tea.Cmd
		cmds = append(cmds, tick(), fetchStats())

		if time.Since(m.lastNews) > time.Hour {
			m.lastNews = time.Now()
			cmds = append(cmds, fetchNews())
		}

		if time.Since(m.lastWeather) > 2*time.Hour {
			m.lastWeather = time.Now()
			cmds = append(cmds, fetchWeather())
		}

		if time.Since(m.lastTasks) > 5*time.Minute {
			m.lastTasks = time.Now()
			cmds = append(cmds, fetchTasks(m.notionKey, m.notionDB))
		}

		return m, tea.Batch(cmds...)

	case statsMsg:
		m.cpu = msg.cpu
		m.ram = msg.ram
		m.disk = msg.disk
		return m, nil

	case weatherMsg:
		m.weather = string(msg)
		return m, nil

	case newsMsg:
		m.news = msg
		m.lastNews = time.Now()
		return m, nil

	case tasksMsg:
		m.tasks = msg
		m.cursor = 0
		m.err = nil
		m.lastTasks = time.Now()
		return m, nil

	case taskDoneMsg:
		if msg.index >= 0 && msg.index < len(m.tasks) {
			m.tasks = append(m.tasks[:msg.index], m.tasks[msg.index+1:]...)
			if m.cursor >= len(m.tasks) && m.cursor > 0 {
				m.cursor--
			}
			if len(m.tasks) == 0 {
				m.tasks = []notionTask{{label: "No upcoming tasks!"}}
			}
		}
		return m, nil

	case taskErrMsg:
		m.err = msg.err
		return m, nil

	case errMsg:
		m.weather = "Weather unavailable right now"
		return m, nil
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

	currentTime := m.time.Format("Monday, 02 Jan 2006 | 15:04:05")
	season := getSeason(m.time)

	headerInfo := fmt.Sprintf("%s\n%s\n\nPress 'q' to quit | j/k to navigate tasks | Enter to mark done", currentTime, season)

	header := lipgloss.JoinHorizontal(lipgloss.Top, asciiStyle.Render(asciiArt), headerInfo)

	colWidth := (m.width - 6) / 2
	innerWidth := colWidth - 6
	innerWidth = max(innerWidth, 10)
	sized := boxStyle.Width(colWidth)

	weatherBox := sized.Render(
		titleStyle.Render("⛅ Local Weather") + "\n\n" + m.weather,
	)

	statsContent := "\n"
	statsContent += "\nCPU: " + drawProgressBar(m.cpu, innerWidth, currentTheme.StatBarFg)
	statsContent += "\nRAM: " + drawProgressBar(m.ram, innerWidth, currentTheme.StatBarFg)
	statsContent += "\nDisk: " + drawProgressBar(m.disk, innerWidth, currentTheme.StatBarFg)
	statsBox := sized.Render(
		titleStyle.Render("💻 System Stats") + statsContent,
	)

	newsContent := ""
	if len(m.news) == 0 {
		newsContent = "\n\nLoading news..."
	} else {
		for _, headline := range m.news {
			wrappedHeadline := lipgloss.NewStyle().
				Width(innerWidth).
				Render("• " + headline)
			newsContent += "\n" + wrappedHeadline
		}
	}
	newsBox := sized.Render(
		titleStyle.Render("📰 Latest News") + "\n" + newsContent,
	)

	tasksContent := ""
	if len(m.tasks) == 0 {
		tasksContent = "\n\nLoading tasks..."
	} else {
		for i, task := range m.tasks {
			var rendered string
			if i == m.cursor && task.id != "" {
				rendered = selectedTaskStyle.Width(innerWidth).Render("▶ " + task.label)
			} else {
				rendered = lipgloss.NewStyle().Width(innerWidth).Render("☐ " + task.label)
			}
			tasksContent += "\n" + rendered
		}
	}

	tasksBox := sized.Render(
		titleStyle.Render("✅ Notion Tasks") + "\n" + tasksContent,
	)

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, statsBox, newsBox)
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, weatherBox, tasksBox)
	grid := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	statusBar := ""
	if m.err != nil {
		statusBar = lipgloss.NewStyle().
			Foreground(currentTheme.Error).
			Width(m.width).
			Render("  ✗ " + m.err.Error())
	}

	finalUI := lipgloss.JoinVertical(lipgloss.Left, header, grid, statusBar)

	return appStyle.Render(finalUI)
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
