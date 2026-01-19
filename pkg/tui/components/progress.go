// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressModel represents a progress indicator for loading states.
type ProgressModel struct {
	title    string
	subtitle string
	spinner  spinner.Model
	progress progress.Model
	percent  float64
	steps    []ProgressStep
	width    int
	styles   ProgressStyles
}

// ProgressStep represents a step in the progress.
type ProgressStep struct {
	Label    string
	Complete bool
	Active   bool
}

// ProgressStyles contains styles for the progress component.
type ProgressStyles struct {
	Container   lipgloss.Style
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	StepDone    lipgloss.Style
	StepActive  lipgloss.Style
	StepPending lipgloss.Style
	Muted       lipgloss.Style
	Border      lipgloss.Style
}

// NewProgressStyles creates adaptive progress styles.
func NewProgressStyles() ProgressStyles {
	primary := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	secondary := lipgloss.AdaptiveColor{Light: "#209fb5", Dark: "#74c7ec"}
	accent := lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}
	success := lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	textMuted := lipgloss.AdaptiveColor{Light: "#8c8fa1", Dark: "#6c7086"}
	border := lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}

	return ProgressStyles{
		Container: lipgloss.NewStyle().
			Padding(2, 4),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(secondary).
			MarginBottom(2),

		StepDone: lipgloss.NewStyle().
			Foreground(success),

		StepActive: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		StepPending: lipgloss.NewStyle().
			Foreground(textMuted),

		Muted: lipgloss.NewStyle().
			Foreground(textMuted),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),
	}
}

// NewProgressModel creates a new progress model.
func NewProgressModel() ProgressModel {
	accent := lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accent)

	// Use solid fill to improve rendering stability and avoid gradient artifacts
	p := progress.New(
		progress.WithSolidFill("#8839ef"),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return ProgressModel{
		spinner:  s,
		progress: p,
		styles:   NewProgressStyles(),
	}
}

// Init initializes the progress model.
func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// SetTitle sets the progress title.
func (m *ProgressModel) SetTitle(title string) {
	m.title = title
}

// SetSubtitle sets the progress subtitle.
func (m *ProgressModel) SetSubtitle(subtitle string) {
	m.subtitle = subtitle
}

// SetPercent sets the progress percentage (0-1).
func (m *ProgressModel) SetPercent(percent float64) {
	m.percent = percent
}

// SetSteps sets the progress steps.
func (m *ProgressModel) SetSteps(steps []ProgressStep) {
	m.steps = steps
}

// GetSteps returns a copy of the progress steps.
func (m *ProgressModel) GetSteps() []ProgressStep {
	copy := make([]ProgressStep, len(m.steps))
	for i, s := range m.steps {
		copy[i] = s
	}
	return copy
}

// SetWidth sets the width.
func (m *ProgressModel) SetWidth(width int) {
	m.width = width
	m.progress.Width = width - 20
	if m.progress.Width > 50 {
		m.progress.Width = 50
	}
}

// Update handles spinner ticks.
// Note: We intentionally DO NOT handle progress.FrameMsg here because we are using
// ViewAs() for static rendering based on manual percentage updates.
func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the progress component.
func (m ProgressModel) View() string {
	var b strings.Builder

	// Title
	if m.title != "" {
		b.WriteString(m.styles.Title.Render(m.title))
		b.WriteString("\n\n")
	}

	// Progress bar - sanitize to ensure no newlines
	barRaw := m.progress.ViewAs(m.percent)
	barRaw = strings.ReplaceAll(barRaw, "\n", "")
	percentage := fmt.Sprintf(" %.0f%%", m.percent*100)

	// Force horizontal layout to prevent splitting
	barLine := lipgloss.JoinHorizontal(lipgloss.Center, barRaw, percentage)
	b.WriteString(barLine)
	b.WriteString("\n\n")

	// Spinner + Subtitle
	if m.subtitle != "" {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Subtitle.Render(m.subtitle))
		b.WriteString("\n\n")
	}

	// Steps
	if len(m.steps) > 0 {
		stepsBox := m.renderSteps()
		b.WriteString(stepsBox)
	}

	return m.styles.Container.Render(b.String())
}

func (m ProgressModel) renderSteps() string {
	var b strings.Builder

	for _, step := range m.steps {
		var icon string
		var style lipgloss.Style

		if step.Complete {
			icon = "✓"
			style = m.styles.StepDone
		} else if step.Active {
			icon = "⋯"
			style = m.styles.StepActive
		} else {
			icon = "○"
			style = m.styles.StepPending
		}

		line := fmt.Sprintf("%s %s", icon, step.Label)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return m.styles.Border.Render(b.String())
}
