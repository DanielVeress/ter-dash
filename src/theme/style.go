package theme

import "github.com/charmbracelet/lipgloss"


type globalTheme struct {
	Border     lipgloss.Color
	TitleFg    lipgloss.Color
	TitleBg    lipgloss.Color
	AsciiArt   lipgloss.Color
	Error      lipgloss.Color
	StatBarBg  lipgloss.Color
	StatBarFg  lipgloss.Color
}

var GlobalTheme = globalTheme{
    Border:    lipgloss.Color("#4C4F69"),
    TitleFg:   lipgloss.Color("#EFF1F5"),
    TitleBg:   lipgloss.Color("#1E66F5"),
    AsciiArt:  lipgloss.Color("#8839EF"),
    Error:     lipgloss.Color("#D20F39"),
	StatBarBg: lipgloss.Color("#444444"),
	StatBarFg: lipgloss.Color("#A6E3A1"),
}

var (
	AppStyle = lipgloss.NewStyle().Margin(1, 2)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GlobalTheme.Border).
			Padding(1, 2).
			Align(lipgloss.Left)

	TitleStyle = lipgloss.NewStyle().
			Foreground(GlobalTheme.TitleFg).
			Background(GlobalTheme.TitleBg).
			Padding(0, 1).
			Bold(true)

	AsciiStyle = lipgloss.NewStyle().
			Foreground(GlobalTheme.AsciiArt).
			Bold(true).
			MarginRight(4)

	SelectedTaskStyle = lipgloss.NewStyle().
				Foreground(GlobalTheme.TitleFg).
				Background(GlobalTheme.Border).
				Bold(true)
)