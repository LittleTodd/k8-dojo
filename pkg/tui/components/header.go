// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// HeaderModel represents the top header bar.
type HeaderModel struct {
	title     string
	version   string
	startTime time.Time
	width     int
	styles    HeaderStyles
}

// HeaderStyles contains styles for the header.
type HeaderStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Version   lipgloss.Style
	Timer     lipgloss.Style
}

// NewHeaderStyles creates adaptive header styles.
func NewHeaderStyles() HeaderStyles {
	border := lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}
	primary := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	secondary := lipgloss.AdaptiveColor{Light: "#209fb5", Dark: "#74c7ec"}
	accent := lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}
	textBold := lipgloss.AdaptiveColor{Light: "#eff1f5", Dark: "#1e1e2e"}

	return HeaderStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(border),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		Version: lipgloss.NewStyle().
			Bold(true).
			Foreground(textBold).
			Background(secondary).
			Padding(0, 1),

		Timer: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),
	}
}

// NewHeaderModel creates a new header model.
func NewHeaderModel() HeaderModel {
	return HeaderModel{
		title:  "🥋 K8s-Dojo",
		styles: NewHeaderStyles(),
	}
}

// SetTitle sets the header title.
func (m *HeaderModel) SetTitle(title string) {
	m.title = title
}

// SetVersion sets the version display.
func (m *HeaderModel) SetVersion(version string) {
	m.version = version
}

// SetWidth sets the header width.
func (m *HeaderModel) SetWidth(width int) {
	m.width = width
}

// StartTimer starts the elapsed time timer.
func (m *HeaderModel) StartTimer() {
	m.startTime = time.Now()
}

// ResetTimer resets the timer.
func (m *HeaderModel) ResetTimer() {
	m.startTime = time.Time{}
}

// ElapsedTime returns the elapsed time since timer started.
func (m HeaderModel) ElapsedTime() time.Duration {
	if m.startTime.IsZero() {
		return 0
	}
	return time.Since(m.startTime)
}

// View renders the header.
func (m HeaderModel) View() string {
	// Left: Title
	left := m.styles.Title.Render(m.title)

	// Right: Version badge + Timer
	var right string
	if m.version != "" {
		right = m.styles.Version.Render(m.version)
	}
	if !m.startTime.IsZero() {
		elapsed := m.ElapsedTime().Round(time.Second)
		timer := m.styles.Timer.Render(fmt.Sprintf("⏱ %s", elapsed))
		if right != "" {
			right = right + "  " + timer
		} else {
			right = timer
		}
	}

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := m.width - leftWidth - rightWidth - 4 // -4 for padding

	if spacerWidth < 0 {
		spacerWidth = 1
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)

	return m.styles.Container.Width(m.width - 2).Render(content)
}
