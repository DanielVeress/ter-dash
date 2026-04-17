package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	cursor int
	items  []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
				case "ctrl+c", "q":
					return m, tea.Quit
				case "up", "k":
					if m.cursor > 0 {
						m.cursor--
					}
				case "down", "j":
					if m.cursor < len(m.items)-1 {
						m.cursor++
					}
			}
	}
	return m, nil
}

func (m model) View() string {
	s := "|TerDash|\n\n"

	for i, item := range m.items {
		cursorMarker := " "
		if m.cursor == i {
			cursorMarker = ">"
		}

		s += fmt.Sprintf("%s %s\n", cursorMarker, item)
	}

	s += "\nPress q to quit.\n"
	return s
}

func main() {
	initialModel := model{
		items: []string{"System Stats", "Weather", "Notion Tasks"},
	}

	p := tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}