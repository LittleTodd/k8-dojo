// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SidebarItem represents an item in the sidebar.
type SidebarItem struct {
	ID          string
	Title       string
	Description string
	IsCategory  bool
	Category    string
	Completed   bool
	Children    []SidebarItem
}

// SidebarModel represents the left sidebar with collapsible categories.
type SidebarModel struct {
	items          []SidebarItem
	cursor         int
	expanded       map[string]bool
	width          int
	height         int
	focused        bool
	styles         SidebarStyles
	completedCount int
	totalCount     int
}

// SidebarStyles contains styles for the sidebar.
type SidebarStyles struct {
	Container      lipgloss.Style
	FocusedBorder  lipgloss.Style
	Category       lipgloss.Style
	CategoryActive lipgloss.Style
	Item           lipgloss.Style
	ItemActive     lipgloss.Style
	ItemCompleted  lipgloss.Style
	Progress       lipgloss.Style
	Muted          lipgloss.Style
}

// NewSidebarStyles creates adaptive sidebar styles.
func NewSidebarStyles() SidebarStyles {
	// Use AdaptiveColor for automatic light/dark mode detection
	border := lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}
	activeBorder := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	text := lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}
	textMuted := lipgloss.AdaptiveColor{Light: "#8c8fa1", Dark: "#6c7086"}
	primary := lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	success := lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	subtext := lipgloss.AdaptiveColor{Light: "#6c6f85", Dark: "#a6adc8"}

	return SidebarStyles{
		Container: lipgloss.NewStyle().
			Padding(1, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border),

		FocusedBorder: lipgloss.NewStyle().
			Padding(1, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeBorder),

		Category: lipgloss.NewStyle().
			Bold(true).
			Foreground(text),

		CategoryActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		Item: lipgloss.NewStyle().
			Foreground(subtext),

		ItemActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		ItemCompleted: lipgloss.NewStyle().
			Foreground(success),

		Progress: lipgloss.NewStyle().
			Foreground(textMuted),

		Muted: lipgloss.NewStyle().
			Foreground(textMuted),
	}
}

// NewSidebarModel creates a new sidebar model.
func NewSidebarModel() SidebarModel {
	return SidebarModel{
		expanded: make(map[string]bool),
		styles:   NewSidebarStyles(),
	}
}

// SetItems sets the sidebar items.
func (m *SidebarModel) SetItems(items []SidebarItem) {
	m.items = items
	m.cursor = 0

	// Auto-expand all categories initially
	for _, item := range items {
		if item.IsCategory {
			m.expanded[item.ID] = true
		}
	}

	// Count total and completed
	m.totalCount = 0
	m.completedCount = 0
	for _, item := range items {
		for _, child := range item.Children {
			m.totalCount++
			if child.Completed {
				m.completedCount++
			}
		}
	}
}

// SetSize sets the sidebar dimensions.
func (m *SidebarModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocus sets the focus state.
func (m *SidebarModel) SetFocus(focused bool) {
	m.focused = focused
}

// IsFocused returns the focus state.
func (m SidebarModel) IsFocused() bool {
	return m.focused
}

// SelectedItem returns the currently selected item.
func (m SidebarModel) SelectedItem() *SidebarItem {
	flat := m.flattenItems()
	if m.cursor >= 0 && m.cursor < len(flat) {
		return flat[m.cursor]
	}
	return nil
}

// flattenItems returns a flat list of visible items.
func (m SidebarModel) flattenItems() []*SidebarItem {
	var result []*SidebarItem
	for i := range m.items {
		result = append(result, &m.items[i])
		if m.items[i].IsCategory && m.expanded[m.items[i].ID] {
			for j := range m.items[i].Children {
				result = append(result, &m.items[i].Children[j])
			}
		}
	}
	return result
}

// Update handles input.
func (m SidebarModel) Update(msg tea.Msg) (SidebarModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	flat := m.flattenItems()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(flat)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			// Collapse category
			if item := m.SelectedItem(); item != nil && item.IsCategory {
				m.expanded[item.ID] = false
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			// Expand category
			if item := m.SelectedItem(); item != nil && item.IsCategory {
				m.expanded[item.ID] = true
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.cursor = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.cursor = len(flat) - 1
		}
	}

	return m, nil
}

// View renders the sidebar.
func (m SidebarModel) View() string {
	var b strings.Builder

	// Title
	title := m.styles.Category.Render("▼ Modules")
	b.WriteString(title + "\n")

	flat := m.flattenItems()

	// Render items
	for i, item := range flat {
		isActive := i == m.cursor

		var line string
		if item.IsCategory {
			// Category header
			icon := categoryIcon(item.Title)
			arrow := "├─"
			if m.expanded[item.ID] {
				arrow = "▼"
			} else {
				arrow = "▶"
			}

			label := fmt.Sprintf("%s %s %s", arrow, icon, item.Title)
			if isActive {
				line = m.styles.CategoryActive.Render(label)
			} else {
				line = m.styles.Category.Render(label)
			}
		} else {
			// Scenario item
			var status string
			if item.Completed {
				status = "●"
			} else {
				status = "○"
			}

			// Truncate title if needed
			titleWidth := m.width - 10
			title := item.Title
			if len(title) > titleWidth && titleWidth > 3 {
				title = title[:titleWidth-2] + ".."
			}

			label := fmt.Sprintf("  │ %s %s", status, title)

			if isActive {
				line = m.styles.ItemActive.Render(label)
			} else if item.Completed {
				line = m.styles.ItemCompleted.Render(label)
			} else {
				line = m.styles.Item.Render(label)
			}
		}

		b.WriteString(line + "\n")
	}

	// Progress summary
	b.WriteString("\n")
	b.WriteString(m.styles.Muted.Render("──────────────────") + "\n")
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("Scenarios: %d", m.totalCount)) + "\n")

	percentage := 0
	if m.totalCount > 0 {
		percentage = m.completedCount * 100 / m.totalCount
	}
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("Completed: %d (%d%%)", m.completedCount, percentage)) + "\n")

	// Progress bar
	barWidth := m.width - 6
	if barWidth > 20 {
		barWidth = 20
	}
	filled := barWidth * percentage / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	b.WriteString(m.styles.Progress.Render(bar) + "\n")

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

func categoryIcon(category string) string {
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
