// Package tui provides responsive layout calculations.
package tui

// Layout contains calculated dimensions for the UI.
type Layout struct {
	// Terminal dimensions
	Width  int
	Height int

	// Sidebar dimensions
	SidebarWidth int

	// Content dimensions
	ContentWidth  int
	ContentHeight int

	// Header/Footer heights
	HeaderHeight    int
	StatusBarHeight int
}

// MinWidth is the minimum terminal width
const MinWidth = 80

// MinHeight is the minimum terminal height
const MinHeight = 24

// SidebarMinWidth is the minimum sidebar width
const SidebarMinWidth = 24

// SidebarMaxWidth is the maximum sidebar width
const SidebarMaxWidth = 40

// HeaderHeight is the fixed header height
const HeaderHeight = 3

// StatusBarHeight is the fixed status bar height
const StatusBarHeight = 2

// NewLayout creates a new layout based on terminal dimensions.
func NewLayout(width, height int) Layout {
	// Enforce minimum dimensions
	if width < MinWidth {
		width = MinWidth
	}
	if height < MinHeight {
		height = MinHeight
	}

	// Calculate sidebar width (25% of width, with min/max bounds)
	sidebarWidth := width / 4
	if sidebarWidth < SidebarMinWidth {
		sidebarWidth = SidebarMinWidth
	}
	if sidebarWidth > SidebarMaxWidth {
		sidebarWidth = SidebarMaxWidth
	}

	// Calculate content width (remaining space minus border chars)
	// Account for sidebar border (2) and content border (2)
	contentWidth := width - sidebarWidth - 4

	// Calculate content height (remove header, status bar, and borders)
	contentHeight := height - HeaderHeight - StatusBarHeight - 4

	return Layout{
		Width:           width,
		Height:          height,
		SidebarWidth:    sidebarWidth,
		ContentWidth:    contentWidth,
		ContentHeight:   contentHeight,
		HeaderHeight:    HeaderHeight,
		StatusBarHeight: StatusBarHeight,
	}
}

// MainAreaHeight returns the height available for the main content area.
func (l Layout) MainAreaHeight() int {
	return l.Height - l.HeaderHeight - l.StatusBarHeight
}

// IsTooSmall returns true if the terminal is too small.
func (l Layout) IsTooSmall() bool {
	return l.Width < MinWidth || l.Height < MinHeight
}

// CenteredWidth returns a width for centered content.
func (l Layout) CenteredWidth() int {
	width := l.Width * 2 / 3
	if width > 60 {
		width = 60
	}
	return width
}

// CenteredBoxWidth returns width for a centered dialog box.
func (l Layout) CenteredBoxWidth() int {
	width := l.Width / 2
	if width < 30 {
		width = 30
	}
	if width > 50 {
		width = 50
	}
	return width
}
