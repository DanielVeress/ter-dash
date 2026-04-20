package components

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

func GetSeason(t time.Time) string {
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