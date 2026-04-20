package components

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"terminal-dashboard/theme"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WeatherMsg string
type WeatherErrMsg error

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

func FetchWeather() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}

		res, err := client.Get("https://wttr.in/?format=3")
		if err != nil {
			return WeatherErrMsg(err)
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return WeatherErrMsg(err)
		}
		weatherStr := strings.TrimSpace(string(body))

		if strings.Contains(weatherStr, "HTML") || weatherStr == "" {
			return WeatherErrMsg(fmt.Errorf("API unavailable"))
		}

		weatherStr = formatWeatherLocation(weatherStr)

		return WeatherMsg(weatherStr)
	}
}

const asciiArt = `
  _|_|_|_|_|                    _|_|_|                          _|
      _|      _|_|    _|  _|_|  _|    _|    _|_|_|    _|_|_|    _|_|_|
      _|    _|_|_|_|  _|_|      _|    _|  _|    _|  _|_|        _|    _|
      _|    _|        _|        _|    _|  _|    _|      _|_|    _|    _|
      _|      _|_|_|  _|        _|_|_|      _|_|_|  _|_|_|      _|    _|`

func RenderHeader(t time.Time, weather string) string {
    infoBoxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()). // Adds a nice card-like container
        BorderForeground(lipgloss.Color("62")). // Subtle purple/blue border
        Padding(0, 1).
		Width(28)

    timeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")) // Pinkish
    seasonStyle := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("241")) // Gray
    weatherHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // Light blue
    weatherStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

    currentDate := timeStyle.Render(t.Format("Monday, 02 Jan 2006"))
    currentTimeText := timeStyle.Render(t.Format("15:04:05"))
    currentSeason := seasonStyle.Render(getSeason(t))
    
    weatherBlock := lipgloss.JoinVertical(lipgloss.Center,
        weatherHeaderStyle.Render("⛅ Local Weather"),
        weatherStyle.Render(weather),
    )

    infoColumn := lipgloss.JoinVertical(lipgloss.Center,
        currentDate,    
        currentTimeText,
        currentSeason,  
        "",             
        weatherBlock,   
    )

    infoBox := infoBoxStyle.Render(infoColumn)
    leftArt := theme.AsciiStyle.Render(asciiArt) 
    header := lipgloss.JoinHorizontal(lipgloss.Center, leftArt, infoBox)

    return header
}