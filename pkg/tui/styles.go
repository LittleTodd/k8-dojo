// Package tui provides derived styles from the theme.
package tui

import "github.com/charmbracelet/lipgloss"

// Styles provides pre-built styles using adaptive colors.
type Styles struct {
	// Text styles
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Text      lipgloss.Style
	TextMuted lipgloss.Style
	TextBold  lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// Interactive styles
	Highlight     lipgloss.Style
	ActiveItem    lipgloss.Style
	SelectedItem  lipgloss.Style
	FocusedBorder lipgloss.Style
	NormalBorder  lipgloss.Style

	// Layout styles
	Header    lipgloss.Style
	Sidebar   lipgloss.Style
	Content   lipgloss.Style
	StatusBar lipgloss.Style
	Box       lipgloss.Style
	HintBox   lipgloss.Style

	// Special styles
	Help    lipgloss.Style
	Badge   lipgloss.Style
	Timer   lipgloss.Style
	Command lipgloss.Style
}

// NewStyles creates styled components from the theme.
func NewStyles(theme Theme) Styles {
	return Styles{
		// Text styles
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true),

		Text: lipgloss.NewStyle().
			Foreground(theme.Text),

		TextMuted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		TextBold: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBold),

		// Status styles
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Success),

		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Error),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning),

		Info: lipgloss.NewStyle().
			Foreground(theme.Secondary),

		// Interactive styles
		Highlight: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Accent),

		ActiveItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBold).
			Background(theme.Primary).
			Padding(0, 1),

		FocusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderActive),

		NormalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		// Layout styles
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(theme.Border),

		Sidebar: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		Content: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		StatusBar: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(theme.Border),

		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderActive).
			Padding(1, 2),

		HintBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Warning).
			Padding(1, 2).
			MarginTop(1),

		// Special styles
		Help: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		Badge: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.TextBold).
			Background(theme.Primary).
			Padding(0, 1),

		Timer: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true),

		Command: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Background(theme.BackgroundAlt).
			Padding(0, 1),
	}
}

// CategoryIcon returns the icon for a category.
func CategoryIcon(category string) string {
	icons := map[string]string{
		"Networking": "🌐",
		"Lifecycle":  "🔄",
		"Scheduling": "📅",
		"Security":   "🔒",
		"Storage":    "💾",
		"Ops":        "⚙️",
		"Resources":  "📊",
		"Kernel":     "🐧",
	}
	if icon, ok := icons[category]; ok {
		return icon
	}
	return "📁"
}

// StatusIndicator returns the status indicator character.
func StatusIndicator(completed bool) string {
	if completed {
		return "●" // Filled circle for completed
	}
	return "○" // Empty circle for not completed
}
