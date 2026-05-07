package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"terminal-dashboard/theme"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NotionTask struct {
	ID      string
	Label   string
	DueDate string // "YYYY-MM-DD", empty if not set

	IsPending      bool
	IsJustFinished bool
}

// FilterUrgentOrAll returns tasks due today or earlier (urgent).
// If no urgent tasks remain, it returns all tasks so future work is still visible.
func FilterUrgentOrAll(tasks []NotionTask) []NotionTask {
	today := time.Now().Format("2006-01-02")
	var urgent []NotionTask
	for _, t := range tasks {
		if t.DueDate != "" && t.DueDate <= today {
			urgent = append(urgent, t)
		}
	}
	if len(urgent) > 0 {
		return urgent
	}
	result := make([]NotionTask, len(tasks))
	copy(result, tasks)
	return result
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
	ID         string                    `json:"id"`
	Properties map[string]NotionProperty `json:"properties"`
}
type TasksMsg []NotionTask
type PriorityTasksMsg struct {
	Priority string
	Tasks    []NotionTask
}
type TaskErrMsg struct{ Err error }
type TaskDoneMsg struct{ ID string }
type ClearFlashMsg struct{ ID string }

func FetchTasks(apiKey string, dbID string) tea.Cmd {
	return func() tea.Msg {
		if apiKey == "" || dbID == "" {
			return TaskErrMsg{Err: fmt.Errorf("Notion credentials not set")}
		}
		url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", dbID)

		today := time.Now().Format("2006-01-02")
		payloadStr := fmt.Sprintf(`{
            "page_size": 10,
            "filter": {
                "and": [
                    {
                        "property": "Status",
                        "status": {
                            "does_not_equal": "Done"
                        }
                    },
                    {
                        "property": "Due Date",
                        "date": {
                            "on_or_before": "%s"
                        }
                    }
                ]
            },
            "sorts": [
                {
                    "property": "Due Date",
                    "direction": "ascending"
                }
            ]
        }`, today)
		payload := []byte(payloadStr)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error building request: %w", err)}
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error reaching Notion: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return TaskErrMsg{Err: fmt.Errorf("Notion API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
		}

		var notionResp NotionResponse
		if err := json.NewDecoder(resp.Body).Decode(&notionResp); err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error parsing Notion response: %w", err)}
		}

		var tasks []NotionTask
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

			tasks = append(tasks, NotionTask{ID: page.ID, Label: title + dateStr})
		}

		if len(tasks) == 0 {
			tasks = append(tasks, NotionTask{Label: "No upcoming tasks!"})
		}

		return TasksMsg(tasks)
	}
}

func FetchTasksByPriority(apiKey, dbID, priority string) tea.Cmd {
	return func() tea.Msg {
		if apiKey == "" || dbID == "" {
			return TaskErrMsg{Err: fmt.Errorf("Notion credentials not set")}
		}
		url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", dbID)

		payloadStr := fmt.Sprintf(`{
            "page_size": 10,
            "filter": {
                "and": [
                    {
                        "property": "Status",
                        "status": {
                            "does_not_equal": "Done"
                        }
                    },
                    {
                        "property": "Priority",
                        "select": {
                            "equals": "%s"
                        }
                    }
                ]
            },
            "sorts": [
                {
                    "property": "Due Date",
                    "direction": "ascending"
                }
            ]
        }`, priority)
		payload := []byte(payloadStr)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
		if err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error building request: %w", err)}
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error reaching Notion: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return TaskErrMsg{Err: fmt.Errorf("Notion API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
		}

		var notionResp NotionResponse
		if err := json.NewDecoder(resp.Body).Decode(&notionResp); err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error parsing Notion response: %w", err)}
		}

		var tasks []NotionTask
		for _, page := range notionResp.Results {
			title := "Untitled"
			for _, prop := range page.Properties {
				if prop.Type == "title" && len(prop.Title) > 0 {
					title = prop.Title[0].PlainText
					break
				}
			}

			dueDate := ""
			dateStr := ""
			if dueProp, ok := page.Properties["Due Date"]; ok && dueProp.Date != nil {
				dueDate = dueProp.Date.Start
				dateStr = fmt.Sprintf(" (Due: %s)", dueDate)
			}

			tasks = append(tasks, NotionTask{ID: page.ID, Label: title + dateStr, DueDate: dueDate})
		}

		return PriorityTasksMsg{Priority: priority, Tasks: tasks}
	}
}

func MarkTaskDone(pageID, apiKey string) tea.Cmd {
	return func() tea.Msg {
		payload := []byte(`{"properties":{"Status":{"status":{"name":"Done"}}}}`)
		url := fmt.Sprintf("https://api.notion.com/v1/pages/%s", pageID)

		req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payload))
		if err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error building request: %w", err)}
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Notion-Version", "2022-06-28")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return TaskErrMsg{Err: fmt.Errorf("error reaching Notion: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return TaskErrMsg{Err: fmt.Errorf("Notion API error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))}
		}

		return TaskDoneMsg{ID: pageID}
	}
}

func RenderTasks(sized lipgloss.Style, tasks []NotionTask, cursor int, innerWidth int, currentPriority string) string {
	pendingStyle := lipgloss.NewStyle().
		Foreground(theme.GlobalTheme.Warning).
		Bold(true)

	successStyle := lipgloss.NewStyle().
		Foreground(theme.GlobalTheme.Success).
		Bold(true)

	mutedStyle := lipgloss.NewStyle().
		Foreground(theme.GlobalTheme.TextPrimary)

	tasksContent := ""
	if len(tasks) == 0 {
		tasksContent = "\n\n" + lipgloss.NewStyle().Foreground(theme.GlobalTheme.TextMuted).Render("Loading tasks...")
	} else {
		for i, task := range tasks {
			var rendered string

			if task.IsJustFinished {
				rendered = successStyle.Width(innerWidth).Render("✓ " + task.Label)
			} else if task.IsPending {
				rendered = pendingStyle.Width(innerWidth).Render("⟳ " + task.Label)
			} else if i == cursor && task.ID != "" {
				rendered = theme.SelectedTaskStyle.Width(innerWidth).Render("▶ " + task.Label)
			} else {
				rendered = mutedStyle.Width(innerWidth).Render("☐ " + task.Label)
			}

			tasksContent += "\n" + rendered
		}
	}

	priorities := []string{"Top", "High", "No Priority"}
	priorityColors := map[string]lipgloss.Color{
		"Top":         theme.GlobalTheme.Error,
		"High":        theme.GlobalTheme.Warning,
		"No Priority": theme.GlobalTheme.TextMuted,
	}

	activeStyle := lipgloss.NewStyle().Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(theme.GlobalTheme.TextMuted)

	var priorityTabs []string
	for _, p := range priorities {
		if p == currentPriority {
			tab := activeStyle.Foreground(priorityColors[p]).Render("▶ " + strings.ToUpper(p))
			priorityTabs = append(priorityTabs, tab)
		} else {
			priorityTabs = append(priorityTabs, inactiveStyle.Render("▷ "+p))
		}
	}
	priorityBar := "  " + strings.Join(priorityTabs, "  ")

	tasksBox := sized.Render(
		theme.TitleStyle.Render("✅ Notion Tasks") + "\n" + priorityBar + "\n" + tasksContent,
	)

	return tasksBox
}
