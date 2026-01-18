// Package components provides reusable TUI components.
package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel represents the bottom status bar with keybindings.
type StatusBarModel struct {
	keys   []key.Binding
	width  int
	styles StatusBarStyles
}

// StatusBarStyles contains styles for the status bar.
type StatusBarStyles struct {
	Container lipgloss.Style
	Key       lipgloss.Style
	Separator lipgloss.Style
}

// NewStatusBarStyles creates adaptive status bar styles.
func NewStatusBarStyles() StatusBarStyles {
	border := lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}
	textMuted := lipgloss.AdaptiveColor{Light: "#8c8fa1", Dark: "#6c7086"}
	accent := lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}

	return StatusBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(border).
			Foreground(textMuted),

		Key: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		Separator: lipgloss.NewStyle().
			Foreground(border),
	}
}

// NewStatusBarModel creates a new status bar model.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		styles: NewStatusBarStyles(),
	}
}

// SetKeys sets the keybindings to display.
func (m *StatusBarModel) SetKeys(keys []key.Binding) {
	m.keys = keys
}

// SetWidth sets the status bar width.
func (m *StatusBarModel) SetWidth(width int) {
	m.width = width
}

// View renders the status bar.
func (m StatusBarModel) View() string {
	var parts []string

	for _, k := range m.keys {
		if !k.Enabled() {
			continue
		}
		help := k.Help()
		keyStr := m.styles.Key.Render(help.Key)
		desc := help.Desc
		parts = append(parts, keyStr+":"+desc)
	}

	sep := m.styles.Separator.Render("  ")
	content := strings.Join(parts, sep)

	return m.styles.Container.
		Width(m.width - 2).
		Render(content)
}

// ContextualStatusBar returns keybindings text for a specific context.
func ContextualStatusBar(context string) []key.Binding {
	switch context {
	case "version-select":
		return []key.Binding{
			key.NewBinding(key.WithKeys("↑/k"), key.WithHelp("↑/k", "up")),
			key.NewBinding(key.WithKeys("↓/j"), key.WithHelp("↓/j", "down")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	case "category-select":
		return []key.Binding{
			key.NewBinding(key.WithKeys("↑/k"), key.WithHelp("↑/k", "up")),
			key.NewBinding(key.WithKeys("↓/j"), key.WithHelp("↓/j", "down")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	case "scenario-select":
		return []key.Binding{
			key.NewBinding(key.WithKeys("↑/k"), key.WithHelp("↑/k", "up")),
			key.NewBinding(key.WithKeys("↓/j"), key.WithHelp("↓/j", "down")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "start")),
			key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	case "scenario-running":
		return []key.Binding{
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check")),
			key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hints")),
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "focus")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	case "success":
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
			key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "menu")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	default:
		return []key.Binding{
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		}
	}
}
