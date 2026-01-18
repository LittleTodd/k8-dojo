// Package tui provides the terminal user interface theme.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color scheme for the TUI.
// Uses lipgloss.AdaptiveColor for automatic dark/light mode detection.
type Theme struct {
	// Primary colors
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Accent    lipgloss.AdaptiveColor

	// Status colors
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor

	// Text colors
	Text      lipgloss.AdaptiveColor
	TextMuted lipgloss.AdaptiveColor
	TextBold  lipgloss.AdaptiveColor

	// Background colors
	Background    lipgloss.AdaptiveColor
	BackgroundAlt lipgloss.AdaptiveColor

	// Border colors
	Border       lipgloss.AdaptiveColor
	BorderActive lipgloss.AdaptiveColor
}

// DefaultTheme returns the default theme with adaptive colors.
// Colors are based on popular terminal color schemes:
// - Dark mode: Inspired by Catppuccin Mocha
// - Light mode: Inspired by Catppuccin Latte
func DefaultTheme() Theme {
	return Theme{
		// Primary - Main brand color (pink/mauve)
		Primary: lipgloss.AdaptiveColor{
			Light: "#8839ef", // Mauve (Catppuccin Latte)
			Dark:  "#cba6f7", // Mauve (Catppuccin Mocha)
		},
		// Secondary - Complementary color (teal/sapphire)
		Secondary: lipgloss.AdaptiveColor{
			Light: "#209fb5", // Sapphire (Catppuccin Latte)
			Dark:  "#74c7ec", // Sapphire (Catppuccin Mocha)
		},
		// Accent - Highlight color (peach)
		Accent: lipgloss.AdaptiveColor{
			Light: "#fe640b", // Peach (Catppuccin Latte)
			Dark:  "#fab387", // Peach (Catppuccin Mocha)
		},

		// Status colors
		Success: lipgloss.AdaptiveColor{
			Light: "#40a02b", // Green (Catppuccin Latte)
			Dark:  "#a6e3a1", // Green (Catppuccin Mocha)
		},
		Warning: lipgloss.AdaptiveColor{
			Light: "#df8e1d", // Yellow (Catppuccin Latte)
			Dark:  "#f9e2af", // Yellow (Catppuccin Mocha)
		},
		Error: lipgloss.AdaptiveColor{
			Light: "#d20f39", // Red (Catppuccin Latte)
			Dark:  "#f38ba8", // Red (Catppuccin Mocha)
		},

		// Text colors
		Text: lipgloss.AdaptiveColor{
			Light: "#4c4f69", // Text (Catppuccin Latte)
			Dark:  "#cdd6f4", // Text (Catppuccin Mocha)
		},
		TextMuted: lipgloss.AdaptiveColor{
			Light: "#8c8fa1", // Overlay 0 (Catppuccin Latte)
			Dark:  "#6c7086", // Overlay 0 (Catppuccin Mocha)
		},
		TextBold: lipgloss.AdaptiveColor{
			Light: "#1e1e2e", // Crust (inverted for contrast)
			Dark:  "#ffffff", // White
		},

		// Background colors (use empty for terminal default)
		Background: lipgloss.AdaptiveColor{
			Light: "",
			Dark:  "",
		},
		BackgroundAlt: lipgloss.AdaptiveColor{
			Light: "#e6e9ef", // Mantle (Catppuccin Latte)
			Dark:  "#313244", // Surface 0 (Catppuccin Mocha)
		},

		// Border colors
		Border: lipgloss.AdaptiveColor{
			Light: "#bcc0cc", // Surface 1 (Catppuccin Latte)
			Dark:  "#45475a", // Surface 1 (Catppuccin Mocha)
		},
		BorderActive: lipgloss.AdaptiveColor{
			Light: "#8839ef", // Mauve (Catppuccin Latte)
			Dark:  "#cba6f7", // Mauve (Catppuccin Mocha)
		},
	}
}
