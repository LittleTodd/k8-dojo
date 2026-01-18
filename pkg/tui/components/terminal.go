// Package components provides reusable TUI components.
package components

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// TerminalOutputMsg is sent when new terminal output is available.
type TerminalOutputMsg struct{}

// TerminalStyles contains styles for the terminal component.
type TerminalStyles struct {
	Container     lipgloss.Style
	FocusedBorder lipgloss.Style
	Title         lipgloss.Style
}

// NewTerminalStyles creates adaptive terminal styles.
func NewTerminalStyles() TerminalStyles {
	border := lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}
	activeBorder := lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	prompt := lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}

	return TerminalStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border),

		FocusedBorder: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(activeBorder),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(prompt),
	}
}

// TerminalModel represents an embedded terminal using vt10x for emulation.
type TerminalModel struct {
	// PTY and process
	pty *os.File
	cmd *exec.Cmd

	// Virtual Terminal Emulator
	term vt10x.Terminal

	// Dimensions
	width  int
	height int

	// State
	focused bool
	running bool
	mu      sync.RWMutex

	// WaitGroup for goroutine cleanup
	wg sync.WaitGroup

	// Program reference for sending messages
	program *tea.Program

	// Styles
	styles TerminalStyles

	// Shell path
	shell string

	// Environment for kubectl
	kubeconfig     string
	kubeconfigPath string
}

// NewTerminalModel creates a new terminal model.
func NewTerminalModel() *TerminalModel {
	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Initialize with a default size, will be resized later
	return &TerminalModel{
		term:   vt10x.New(vt10x.WithSize(80, 24)),
		styles: NewTerminalStyles(),
		shell:  shell,
	}
}

// SetProgram sets the tea.Program reference for sending refresh messages.
func (m *TerminalModel) SetProgram(p *tea.Program) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.program = p
}

// SetKubeconfig sets the kubeconfig path for kubectl commands.
func (m *TerminalModel) SetKubeconfig(kubeconfig string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.kubeconfig = kubeconfig
}

// Start spawns a new shell with PTY.
func (m *TerminalModel) Start() tea.Cmd {
	return func() tea.Msg {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.running {
			return nil
		}

		// Create command
		m.cmd = exec.Command(m.shell)
		m.cmd.Env = append(os.Environ(),
			"TERM=xterm-256color",
			"PS1=$ ",
			"KUBE_EDITOR=vim -c 'syntax on'", // Force vim with syntax highlighting for kubectl
			"EDITOR=vim",                     // Default editor
			"VIMINIT=syntax on",              // Ensure syntax is on for direct vim usage
		)

		// Add kubeconfig if set
		if m.kubeconfig != "" {
			// Create temp file for kubeconfig
			tmpFile, err := os.CreateTemp("", "k8s-dojo-*.kubeconfig")
			if err != nil {
				// We can't write to term directly easily without PTY, just ignore or log
				return TerminalOutputMsg{}
			}

			if _, err := tmpFile.Write([]byte(m.kubeconfig)); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				return TerminalOutputMsg{}
			}
			tmpFile.Close()
			m.kubeconfigPath = tmpFile.Name()
			m.cmd.Env = append(m.cmd.Env, "KUBECONFIG="+m.kubeconfigPath)
		}

		// Start with PTY
		var err error
		m.pty, err = pty.Start(m.cmd)
		if err != nil {
			if m.kubeconfigPath != "" {
				os.Remove(m.kubeconfigPath)
				m.kubeconfigPath = ""
			}
			return TerminalOutputMsg{}
		}

		// Set initial size
		if m.width > 0 && m.height > 0 {
			_ = pty.Setsize(m.pty, &pty.Winsize{
				Rows: uint16(m.height - 2),
				Cols: uint16(m.width - 4),
			})
			m.term.Resize(m.width-4, m.height-2)
		}

		m.running = true

		// Write specific Welcome message to specific VTE
		// Note: We can write to VTE directly, bypassing PTY echo if we want
		fmt.Fprintln(m.term, "Terminal ready. Use kubectl commands below:")

		// Start reading output in background
		m.wg.Add(1)
		go m.readOutput()

		return TerminalOutputMsg{}
	}
}

// readOutput continuously reads from PTY and sends messages.
func (m *TerminalModel) readOutput() {
	defer m.wg.Done()

	buf := make([]byte, 4096)
	for {
		m.mu.RLock()
		running := m.running
		ptyFile := m.pty
		m.mu.RUnlock()

		if !running || ptyFile == nil {
			return
		}

		n, err := ptyFile.Read(buf)
		if err != nil {
			if err != io.EOF {
				m.mu.Lock()
				fmt.Fprintln(m.term, "\nTerminal closed")
				m.running = false
				m.mu.Unlock()

				m.mu.RLock()
				p := m.program
				m.mu.RUnlock()
				if p != nil {
					p.Send(TerminalOutputMsg{})
				}
			}
			return
		}

		if n > 0 {
			m.mu.Lock()
			// Direct Write to VT10x emulator
			_, _ = m.term.Write(buf[:n])
			m.mu.Unlock()

			m.mu.RLock()
			p := m.program
			m.mu.RUnlock()
			if p != nil {
				p.Send(TerminalOutputMsg{})
			}
		}
	}
}

// Stop closes the PTY and terminates the shell.
func (m *TerminalModel) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false

	if m.pty != nil {
		m.pty.Close() // This will cause readOutput to exit err from Read
		m.pty = nil
	}
	m.mu.Unlock()

	// Wait for readOutput
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Signal(syscall.SIGTERM)
		// Cleanup process logic ... simplified for brevity, assume system handles orphans or eventual kill
	}

	if m.kubeconfigPath != "" {
		_ = os.Remove(m.kubeconfigPath)
		m.kubeconfigPath = ""
	}

	// Reset terminal state
	m.term = vt10x.New(vt10x.WithSize(80, 24))
}

// SetSize sets the terminal dimensions.
func (m *TerminalModel) SetSize(width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.width = width
	m.height = height

	// Inner terminal size (minus border)
	termW := width - 4
	termH := height - 2
	if termW < 1 {
		termW = 1
	}
	if termH < 1 {
		termH = 1
	}

	if m.pty != nil {
		_ = pty.Setsize(m.pty, &pty.Winsize{
			Rows: uint16(termH),
			Cols: uint16(termW),
		})
	}
	// Resize emulator
	m.term.Resize(termW, termH)
}

// SetFocus sets the focus state.
func (m *TerminalModel) SetFocus(focused bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.focused = focused
}

// IsFocused returns the focus state.
func (m *TerminalModel) IsFocused() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.focused
}

// IsRunning returns whether the terminal is running.
func (m *TerminalModel) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SendInput sends a string to the terminal.
func (m *TerminalModel) SendInput(input string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pty != nil && m.running {
		_, _ = m.pty.WriteString(input)
	}
}

// ScrollUp/Down - Not supported in basic vt10x without history wrapper, stubs for now
func (m *TerminalModel) ScrollUp(lines int)   {}
func (m *TerminalModel) ScrollDown(lines int) {}

// Update handles input and messages.
func (m *TerminalModel) Update(msg tea.Msg) tea.Cmd {
	m.mu.RLock()
	focused := m.focused
	m.mu.RUnlock()

	if !focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Paste {
			// Still wrapping paste to be safe
			m.SendInput("\x1b[200~" + string(msg.Runes) + "\x1b[201~")
			return nil
		}

		// Handle keys mapping to VT100 sequences
		// Same as before
		switch msg.Type {
		case tea.KeyEnter:
			m.SendInput("\r")
		case tea.KeyBackspace:
			m.SendInput("\x7f") // or \x08 depending on terminal config
		case tea.KeyTab:
			return nil
		case tea.KeyCtrlC:
			m.SendInput("\x03")
		case tea.KeyCtrlD:
			m.SendInput("\x04")
		case tea.KeyCtrlZ:
			m.SendInput("\x1a")
		case tea.KeyCtrlL:
			m.SendInput("\x0c")
		case tea.KeyUp:
			m.SendInput("\x1b[A")
		case tea.KeyDown:
			m.SendInput("\x1b[B")
		case tea.KeyLeft:
			m.SendInput("\x1b[D")
		case tea.KeyRight:
			m.SendInput("\x1b[C")
		case tea.KeyHome:
			m.SendInput("\x1b[H")
		case tea.KeyEnd:
			m.SendInput("\x1b[F")
		case tea.KeyDelete:
			m.SendInput("\x1b[3~")
		case tea.KeyPgUp:
			m.SendInput("\x1b[5~")
		case tea.KeyPgDown:
			m.SendInput("\x1b[6~")
		case tea.KeyRunes:
			m.SendInput(string(msg.Runes))
		case tea.KeySpace:
			m.SendInput(" ")
		case tea.KeyEsc:
			m.SendInput("\x1b")
		default:
			if s := msg.String(); len(s) == 1 {
				m.SendInput(s)
			}
		}
		return nil
	}
	return nil
}

// View renders the terminal using vt10x state.
func (m *TerminalModel) View() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var builder strings.Builder

	cols, rows := m.term.Size()
	cursor := m.term.Cursor()
	cursorX, cursorY := cursor.X, cursor.Y

	// Iterate through visible rows
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Get cell info
			cell := m.term.Cell(x, y)
			c := cell.Char
			fg := cell.FG
			bg := cell.BG

			style := lipgloss.NewStyle()

			// Determine colors with contrast correction
			var fgColor lipgloss.TerminalColor = lipgloss.Color("#cdd6f4") // Default text
			var bgColor lipgloss.TerminalColor = lipgloss.NoColor{}

			hasCustomBG := false
			const DefaultFG_Int = 16777216
			const DefaultBG_Int = 16777217

			bgInt := int(bg)
			fgInt := int(fg)

			// Map Background
			if bgInt == DefaultBG_Int {
				bgColor = lipgloss.NoColor{} // Restore transparency
				// hasCustomBG remains false
			} else if bgInt == DefaultFG_Int {
				// Inversed Default FG (was Light, now we map Default FG to Dark Text)
				// If text is Dark (#4c4f69), then Inverse BG should be Dark (#4c4f69).
				bgColor = lipgloss.Color("#4c4f69")
				hasCustomBG = true
			} else {
				bgColor = lipgloss.Color(fmt.Sprintf("%d", bg))
				hasCustomBG = true
			}

			// Map Foreground
			if fgInt == DefaultFG_Int {
				fgColor = lipgloss.Color("#4c4f69") // Switch to Dark Text (Latte Text)
			} else if fgInt == DefaultBG_Int {
				fgColor = lipgloss.Color("#eff1f5") // DefaultBG as FG -> Light (Latte Base)
			} else {
				fgColor = lipgloss.Color(fmt.Sprintf("%d", fg))
			}

			// Contrast Correction: Force black text on light backgrounds
			if hasCustomBG {
				isLight := false
				if bgInt == DefaultFG_Int {
					// BG is DefaultFG (#4c4f69 Dark). So isLight = False.
					isLight = false
				} else {
					isLight = isLightColor(bgInt)
				}

				if isLight {
					fgColor = lipgloss.Color("#000000") // Force Hex Black
				}

				// If BG is DefaultFG (#4c4f69 Dark), ensure Text is Light.
				if bgInt == DefaultFG_Int {
					fgColor = lipgloss.Color("#eff1f5")
				}
			}

			style = style.Foreground(fgColor)
			if hasCustomBG {
				style = style.Background(bgColor)
			}

			// Cursor rendering
			if m.focused && x == cursorX && y == cursorY {
				style = style.Reverse(true)
			}

			// Handle empty cells
			if c == 0 {
				c = ' '
			}

			builder.WriteString(style.Render(string(c)))
		}
		builder.WriteString("\n")
	}

	// Styles
	container := m.styles.Container
	if m.focused {
		container = m.styles.FocusedBorder
	}

	title := " Terminal (vt10x) "
	return container.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(m.styles.Title.Render(title) + "\n" + builder.String())
}

func isLightColor(c int) bool {
	// Standard Colors (0-15)
	if c == 7 || c == 15 {
		return true
	}
	if c >= 9 && c <= 14 {
		return true // Bright colors are generally light
	}
	// 8 is Dark Gray, 0-6 are Dark.

	// Grayscale (232-255)
	if c >= 244 {
		return true // 244 is light gray
	}

	// Color Cube (16-231)
	if c >= 16 && c <= 231 {
		val := c - 16
		b := val % 6
		val /= 6
		g := val % 6
		r := val / 6

		// Simple relative luminance approximation
		// Max sum is 15 (5+5+5)
		sum := r + g + b
		return sum >= 9
	}

	return false
}
