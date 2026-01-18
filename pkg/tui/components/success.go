// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// SuccessModel represents the success/victory screen.
type SuccessModel struct {
	scenarioName string
	message      string
	elapsedTime  time.Duration
	points       int
	width        int
	height       int
	styles       SuccessStyles
}

// SuccessStyles contains styles for the success screen.
type SuccessStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Check     lipgloss.Style
	Stats     lipgloss.Style
	Muted     lipgloss.Style
	Button    lipgloss.Style
	Box       lipgloss.Style
}

// NewSuccessStyles creates adaptive success styles.
func NewSuccessStyles() SuccessStyles {
	text := lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}
	textMuted := lipgloss.AdaptiveColor{Light: "#8c8fa1", Dark: "#6c7086"}
	primary := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	secondary := lipgloss.AdaptiveColor{Light: "#209fb5", Dark: "#74c7ec"}
	success := lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	textBold := lipgloss.AdaptiveColor{Light: "#eff1f5", Dark: "#1e1e2e"}

	return SuccessStyles{
		Container: lipgloss.NewStyle().
			Padding(2, 4).
			Align(lipgloss.Center),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(success).
			MarginBottom(2),

		Subtitle: lipgloss.NewStyle().
			Foreground(text).
			MarginBottom(1),

		Check: lipgloss.NewStyle().
			Foreground(success),

		Stats: lipgloss.NewStyle().
			Foreground(secondary).
			Bold(true),

		Muted: lipgloss.NewStyle().
			Foreground(textMuted),

		Button: lipgloss.NewStyle().
			Foreground(textBold).
			Background(primary).
			Padding(0, 2).
			Bold(true),

		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(success).
			Padding(1, 2).
			Width(40).
			Align(lipgloss.Center),
	}
}

// NewSuccessModel creates a new success model.
func NewSuccessModel() SuccessModel {
	return SuccessModel{
		points: 100,
		styles: NewSuccessStyles(),
	}
}

// SetScenario sets the completed scenario name.
func (m *SuccessModel) SetScenario(name string) {
	m.scenarioName = name
}

// SetMessage sets the success message.
func (m *SuccessModel) SetMessage(message string) {
	m.message = message
}

// SetElapsedTime sets the elapsed time.
func (m *SuccessModel) SetElapsedTime(elapsed time.Duration) {
	m.elapsedTime = elapsed
}

// SetPoints sets the points earned.
func (m *SuccessModel) SetPoints(points int) {
	m.points = points
}

// SetSize sets the dimensions.
func (m *SuccessModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// View renders the success screen.
func (m SuccessModel) View() string {
	var b strings.Builder

	// Big success title
	b.WriteString(m.styles.Title.Render("🎉  S U C C E S S !"))
	b.WriteString("\n\n")

	// Scenario box
	boxContent := m.renderBox()
	b.WriteString(boxContent)
	b.WriteString("\n\n")

	// Action buttons
	continueBtn := m.styles.Button.Render(" Continue ")
	retryBtn := m.styles.Muted.Render("[ Retry ]")
	b.WriteString(continueBtn + "    " + retryBtn)

	// Center everything
	content := b.String()
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m SuccessModel) renderBox() string {
	var b strings.Builder

	// Scenario name
	b.WriteString(m.styles.Subtitle.Render(m.scenarioName))
	b.WriteString("\n\n")

	// Check marks
	if m.message != "" {
		lines := strings.Split(m.message, "\n")
		for _, line := range lines {
			if line != "" {
				b.WriteString(m.styles.Check.Render("✓ " + line))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	// Stats
	b.WriteString(m.styles.Stats.Render(fmt.Sprintf("⏱ Time: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n")
	b.WriteString(m.styles.Stats.Render(fmt.Sprintf("★ Points: +%d", m.points)))

	return m.styles.Box.Render(b.String())
}
