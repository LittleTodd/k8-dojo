// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ContentModel represents the main content panel.
type ContentModel struct {
	title       string
	description string
	namespace   string
	status      string
	statusOK    bool
	commands    []string
	hints       []string
	currentHint int
	showHints   bool

	viewport viewport.Model
	width    int
	height   int
	focused  bool
	styles   ContentStyles
}

// ContentStyles contains styles for the content panel.
type ContentStyles struct {
	Container     lipgloss.Style
	FocusedBorder lipgloss.Style
	Title         lipgloss.Style
	Subtitle      lipgloss.Style
	Label         lipgloss.Style
	Text          lipgloss.Style
	StatusOK      lipgloss.Style
	StatusError   lipgloss.Style
	CommandBox    lipgloss.Style
	Command       lipgloss.Style
	HintBox       lipgloss.Style
	HintLabel     lipgloss.Style
	Muted         lipgloss.Style
}

// NewContentStyles creates adaptive content styles.
func NewContentStyles() ContentStyles {
	// Use AdaptiveColor for automatic light/dark mode detection
	border := lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}
	activeBorder := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	text := lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}
	textMuted := lipgloss.AdaptiveColor{Light: "#8c8fa1", Dark: "#6c7086"}
	primary := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	secondary := lipgloss.AdaptiveColor{Light: "#209fb5", Dark: "#74c7ec"}
	accent := lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}
	success := lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	errorColor := lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}
	warning := lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}

	return ContentStyles{
		Container: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border),

		FocusedBorder: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeBorder),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		Subtitle: lipgloss.NewStyle().
			Foreground(secondary),

		Label: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		Text: lipgloss.NewStyle().
			Foreground(text),

		StatusOK: lipgloss.NewStyle().
			Bold(true).
			Foreground(success),

		StatusError: lipgloss.NewStyle().
			Bold(true).
			Foreground(errorColor),

		CommandBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondary).
			Padding(0, 1).
			MarginTop(1),

		Command: lipgloss.NewStyle().
			Foreground(accent),

		HintBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(warning).
			Padding(0, 1).
			MarginTop(1),

		HintLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(warning),

		Muted: lipgloss.NewStyle().
			Foreground(textMuted),
	}
}

// NewContentModel creates a new content model.
func NewContentModel() ContentModel {
	return ContentModel{
		styles:   NewContentStyles(),
		viewport: viewport.New(0, 0),
	}
}

// SetScenario sets the scenario content.
func (m *ContentModel) SetScenario(title, description, namespace string) {
	m.title = title
	m.description = description
	m.namespace = namespace
	m.status = ""
	m.statusOK = false
	m.currentHint = 0
}

// SetStatus sets the current status.
func (m *ContentModel) SetStatus(status string, ok bool) {
	m.status = status
	m.statusOK = ok
}

// SetCommands sets the quick commands.
func (m *ContentModel) SetCommands(commands []string) {
	m.commands = commands
}

// SetHints sets the hints.
func (m *ContentModel) SetHints(hints []string) {
	m.hints = hints
	m.currentHint = 0
}

// ToggleHints toggles hint visibility.
func (m *ContentModel) ToggleHints() {
	m.showHints = !m.showHints
}

// NextHint cycles to the next hint.
func (m *ContentModel) NextHint() {
	if len(m.hints) > 0 {
		m.currentHint = (m.currentHint + 1) % len(m.hints)
	}
}

// PrevHint cycles to the previous hint.
func (m *ContentModel) PrevHint() {
	if len(m.hints) > 0 {
		m.currentHint = (m.currentHint - 1 + len(m.hints)) % len(m.hints)
	}
}

// SetSize sets the content dimensions.
func (m *ContentModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width - 6
	m.viewport.Height = height - 10
}

// SetFocus sets the focus state.
func (m *ContentModel) SetFocus(focused bool) {
	m.focused = focused
}

// IsFocused returns the focus state.
func (m ContentModel) IsFocused() bool {
	return m.focused
}

// Update handles input.
func (m ContentModel) Update(msg tea.Msg) (ContentModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the content panel.
func (m ContentModel) View() string {
	var b strings.Builder

	// Title
	if m.title != "" {
		icon := "🔧"
		b.WriteString(m.styles.Title.Render(fmt.Sprintf("%s %s", icon, m.title)))
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render(strings.Repeat("─", m.width-6)))
		b.WriteString("\n\n")
	}

	// Description
	if m.description != "" {
		b.WriteString(m.styles.Label.Render("DESCRIPTION"))
		b.WriteString("\n")
		b.WriteString(m.styles.Text.Render(m.description))
		b.WriteString("\n\n")
	}

	// Namespace
	if m.namespace != "" {
		b.WriteString(m.styles.Label.Render("NAMESPACE: "))
		b.WriteString(m.styles.Subtitle.Render(m.namespace))
		b.WriteString("\n\n")
	}

	// Status
	if m.status != "" {
		b.WriteString(m.styles.Label.Render("STATUS: "))
		var statusStyle lipgloss.Style
		var indicator string
		if m.statusOK {
			statusStyle = m.styles.StatusOK
			indicator = "✓"
		} else {
			statusStyle = m.styles.StatusError
			indicator = "●"
		}
		b.WriteString(statusStyle.Render(indicator))
		b.WriteString(" ")
		b.WriteString(m.styles.Text.Render(m.status))
		b.WriteString("\n")
	}

	// Commands box
	if len(m.commands) > 0 {
		cmdWidth := m.width - 10
		var cmdLines []string
		for _, cmd := range m.commands {
			// Add consistent left padding for alignment
			cmdLines = append(cmdLines, "  "+m.styles.Command.Render(cmd))
		}
		cmdContent := strings.Join(cmdLines, "\n")
		cmdBox := m.styles.CommandBox.Width(cmdWidth).Render(
			m.styles.Muted.Render("Quick Commands") + "\n" + cmdContent,
		)
		b.WriteString(cmdBox)
		b.WriteString("\n")
	}

	// Hints box
	if m.showHints && len(m.hints) > 0 {
		hintWidth := m.width - 10
		hintLabel := m.styles.HintLabel.Render(
			fmt.Sprintf("💡 Hints (%d/%d)", m.currentHint+1, len(m.hints)),
		)
		hintContent := m.styles.Text.Render(m.hints[m.currentHint])
		hintBox := m.styles.HintBox.Width(hintWidth).Render(
			hintLabel + "\n" + hintContent,
		)
		b.WriteString(hintBox)
	}

	// Apply container style
	container := m.styles.Container
	if m.focused {
		container = m.styles.FocusedBorder
	}

	return container.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(b.String())
}
