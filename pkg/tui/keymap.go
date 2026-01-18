// Package tui provides keybinding definitions for the terminal user interface.
package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	// Global
	Quit     key.Binding
	Help     key.Binding
	Escape   key.Binding
	Tab      key.Binding
	ShiftTab key.Binding

	// Navigation (Vim-style)
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	GoToTop  key.Binding
	GoToEnd  key.Binding
	PageDown key.Binding
	PageUp   key.Binding

	// Selection
	Enter  key.Binding
	Search key.Binding

	// Scenario Running
	Check       key.Binding
	ToggleHints key.Binding
	NextHint    key.Binding
	PrevHint    key.Binding
	CopyCommand key.Binding

	// Success View
	Retry      key.Binding
	ReturnMenu key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Global
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),

		// Navigation (Vim-style)
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "top"),
		),
		GoToEnd: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u", "page up"),
		),

		// Selection
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),

		// Scenario Running
		Check: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "check"),
		),
		ToggleHints: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "hints"),
		),
		NextHint: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next hint"),
		),
		PrevHint: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev hint"),
		),
		CopyCommand: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy"),
		),

		// Success View
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
		),
		ReturnMenu: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "menu"),
		),
	}
}

// ShortHelp returns keybindings to show in the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Quit}
}

// FullHelp returns keybindings to show in the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Escape, k.Tab},
		{k.Help, k.Quit},
	}
}

// VersionSelectKeys returns keybindings for version selection view.
func (k KeyMap) VersionSelectKeys() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Quit}
}

// CategorySelectKeys returns keybindings for category selection view.
func (k KeyMap) CategorySelectKeys() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Escape, k.Quit}
}

// ScenarioSelectKeys returns keybindings for scenario selection view.
func (k KeyMap) ScenarioSelectKeys() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Escape, k.Search, k.Quit}
}

// ScenarioRunningKeys returns keybindings for scenario running view.
func (k KeyMap) ScenarioRunningKeys() []key.Binding {
	return []key.Binding{k.Check, k.ToggleHints, k.Tab, k.Help, k.Quit}
}

// SuccessKeys returns keybindings for success view.
func (k KeyMap) SuccessKeys() []key.Binding {
	return []key.Binding{k.Enter, k.Retry, k.ReturnMenu, k.Quit}
}
