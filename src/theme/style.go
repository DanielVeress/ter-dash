package theme

import "github.com/charmbracelet/lipgloss"

type globalTheme struct {
	Border      lipgloss.Color
	TitleFg     lipgloss.Color
	TitleBg     lipgloss.Color
	AsciiArt    lipgloss.Color
	Error       lipgloss.Color
	StatBarBg   lipgloss.Color
	StatBarFg   lipgloss.Color
	TextMuted   lipgloss.Color
	TextPrimary lipgloss.Color
	Accent1     lipgloss.Color
	Accent2     lipgloss.Color
	Active      lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
}

var GlobalTheme = globalTheme{
	Border:      lipgloss.Color("#89B4FA"), // Catppuccin Blue
	TitleFg:     lipgloss.Color("#1E1E2E"), // Catppuccin Base (dark)
	TitleBg:     lipgloss.Color("#89B4FA"), // Catppuccin Blue
	AsciiArt:    lipgloss.Color("#CBA6F7"), // Catppuccin Mauve
	Error:       lipgloss.Color("#F38BA8"), // Catppuccin Red
	StatBarBg:   lipgloss.Color("#45475A"), // Catppuccin Surface1
	StatBarFg:   lipgloss.Color("#A6E3A1"), // Catppuccin Green
	TextMuted:   lipgloss.Color("#6C7086"), // Catppuccin Overlay0
	TextPrimary: lipgloss.Color("#CDD6F4"), // Catppuccin Text
	Accent1:     lipgloss.Color("#F5C2E7"), // Catppuccin Pink
	Accent2:     lipgloss.Color("#94E2D5"), // Catppuccin Teal
	Active:      lipgloss.Color("#FAB387"), // Catppuccin Peach
	Success:     lipgloss.Color("#A6E3A1"), // Catppuccin Green
	Warning:     lipgloss.Color("#F9E2AF"), // Catppuccin Yellow
}

var (
	AppStyle = lipgloss.NewStyle()

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GlobalTheme.Border).
			Padding(1, 2)

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
				Background(GlobalTheme.Active).
				Bold(true)
)
