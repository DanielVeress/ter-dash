package components

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"terminal-dashboard/theme"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NewsItem struct {
	Title string `xml:"title"`
}

type Rss struct {
	Items []NewsItem `xml:"channel>item"`
}

type NewsMsg []string

func FetchNews() tea.Cmd {
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
		return NewsMsg(allHeadlines)
	}
}

func RenderNews(sized lipgloss.Style, news []string, innerWidth int) string {
	bulletStyle := lipgloss.NewStyle().Foreground(theme.GlobalTheme.Accent2)
	textStyle := lipgloss.NewStyle().Foreground(theme.GlobalTheme.TextPrimary)

	newsContent := ""
	if len(news) == 0 {
		newsContent = "\n\n" + lipgloss.NewStyle().Foreground(theme.GlobalTheme.TextMuted).Render("Loading news...")
	} else {
		for _, headline := range news {
			line := lipgloss.NewStyle().Width(innerWidth).Render(
				bulletStyle.Render("• ") + textStyle.Render(headline),
			)
			newsContent += "\n" + line
		}
	}
	newsBox := sized.Render(
		theme.TitleStyle.Render("📰 Latest News") + "\n" + newsContent,
	)

	return newsBox
}
