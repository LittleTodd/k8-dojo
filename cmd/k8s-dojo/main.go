// k8s-dojo: Zero-setup Kubernetes troubleshooting training CLI.
package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/klog/v2"

	"k8s-dojo/pkg/tui"
)

func init() {
	// Silence klog to avoid polluting the TUI
	klog.SetOutput(io.Discard)
	klog.LogToStderr(false)
}

func main() {
	// Run the TUI with the new enhanced architecture
	model := tui.NewAppModel()
	p := tea.NewProgram(&model, tea.WithAltScreen())

	// Set the program reference on the terminal for async output refresh
	model.SetTerminalProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running k8s-dojo: %v\n", err)
		os.Exit(1)
	}
}
