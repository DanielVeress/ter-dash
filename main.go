package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
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
	cpu  float64
	ram  float64
	disk float64
	weather string
	news        []string
	lastNews    time.Time
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
		return raw // Fallback if the string format is unexpected
	}
	locs := strings.Split(parts[0], ",")
	
	// If we have at least city, county, and country
	if len(locs) >= 3 {
		city := strings.TrimSpace(locs[0])
		country := strings.TrimSpace(locs[len(locs)-1])

		// Capitalize
		if len(city) > 0 {
			city = strings.ToUpper(string(city[0])) + city[1:]
		}
		country = strings.ToUpper(country)

		return fmt.Sprintf("%s, %s:%s", city, country, parts[1])
	}

	return raw
}

func getSeason(t time.Time) string {
	// If the time hasn't been set yet, return a default
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
		
		// If the API sends back an HTML error page by accident, catch it
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
				continue // Skip failed sources
			}
			defer resp.Body.Close()

			var rss Rss
			if err := xml.NewDecoder(resp.Body).Decode(&rss); err != nil {
				continue
			}

			// Just take the top 2 headlines per source to keep it "minimal"
			count := 0
			for _, item := range rss.Items {
				if count >= 2 { break }
				allHeadlines = append(allHeadlines, fmt.Sprintf("[%s] %s", name, item.Title))
				count++
			}
		}
		return newsMsg(allHeadlines)
	}
}

func (m model) Init() tea.Cmd {	
	return tea.Batch(
		tick(), 
		fetchStats(), 
		fetchWeather(),
		fetchNews(),
	)
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

			// Every hour check
			var newsCmd tea.Cmd
			if time.Since(m.lastNews) > time.Hour {
				m.lastNews = time.Now()
				newsCmd = fetchNews()
			}
			
			return m, tea.Batch(tick(), fetchStats(), newsCmd)
		
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
	
	// Time
	currentTime := m.time.Format("Monday, 02 Jan 2006 | 15:04:05")
	season := getSeason(m.time)
	
	headerInfo := fmt.Sprintf("%s\n%s\n\nPress 'q' to quit", currentTime, season)
	
	// Join the ASCII art and the Time info side-by-side
	header := lipgloss.JoinHorizontal(lipgloss.Top, asciiStyle.Render(asciiArt), headerInfo)

	// -- Dashboard Quadrants --
	weatherBox := boxStyle.Render(
		titleStyle.Render("⛅ Local Weather") + "\n\n" + m.weather,
	)
    
	statsContent := fmt.Sprintf("\n\nCPU:  %5.1f%%\nRAM:  %5.1f%%\nDisk: %5.1f%%", m.cpu, m.ram, m.disk)
	statsBox := boxStyle.Render(
		titleStyle.Render("💻 System Stats") + statsContent,
	)
	
	newsContent := "\n\n"
	if len(m.news) == 0 {
		newsContent += "Loading news..."
	} else {
		for _, headline := range m.news {
			// Basic wrapping: if headline is too long, truncate it
			if len(headline) > 35 {
				headline = headline[:32] + "..."
			}
			newsContent += "• " + headline + "\n"
		}
	}
	newsBox := boxStyle.Render(
		titleStyle.Render("📰 Latest News") + newsContent,
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
		weather: "Fetching weather...",
    }		

	p := tea.NewProgram(initialModel, tea.WithAltScreen()) 
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}