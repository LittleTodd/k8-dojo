// Package tui provides the terminal user interface using Bubbletea.
package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"k8s-dojo/pkg/cluster"
	"k8s-dojo/pkg/engine"
	"k8s-dojo/pkg/k8s"
	"k8s-dojo/pkg/scenario"
	"k8s-dojo/pkg/state"
	"k8s-dojo/pkg/tui/components"
)

// FocusArea represents which panel is focused.
type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
	FocusTerminal
)

// View represents the current TUI view.
type View int

const (
	ViewVersionSelect View = iota
	ViewBootstrap
	ViewDashboard
	ViewScenarioRunning
	ViewSuccess
	ViewConfirmRestart
	ViewConfirmQuit
)

// AppModel is the main Bubbletea model with the new component architecture.
type AppModel struct {
	// Theme and styles
	theme  Theme
	styles Styles
	keymap KeyMap
	layout Layout

	// Current view and focus
	view         View
	previousView View
	focus        FocusArea

	// Version selection
	versions        []cluster.SupportedVersion
	selectedVersion int

	// Bootstrap
	bootstrap         components.ProgressModel
	bootstrapErr      error
	bootstrapRealDone bool
	bootstrapStep     int

	// Components
	header    components.HeaderModel
	sidebar   components.SidebarModel
	content   components.ContentModel
	terminal  *components.TerminalModel
	statusbar components.StatusBarModel
	success   components.SuccessModel

	// Kubeconfig path for terminal
	kubeconfig string

	// Scenario list (for dashboard)
	scenarioList list.Model

	// Cluster & Engine
	clusterManager *cluster.Manager
	k8sClient      *k8s.Client
	engineInstance *engine.Engine
	registry       *scenario.Registry
	stateManager   *state.Manager

	// State
	completedScenarios map[string]bool
	confirmSelection   int // 0: Yes, 1: No

	// Running scenario
	currentScenario scenario.Scenario
	lastCheckResult scenario.Result
	checkInterval   time.Duration

	// Window size
	width  int
	height int

	// Quit flag
	quitting bool
}

// scenarioItem implements list.Item for scenarios.
type scenarioItem struct {
	scenario scenario.Scenario
}

func (i scenarioItem) Title() string       { return i.scenario.GetMetadata().Name }
func (i scenarioItem) Description() string { return i.scenario.GetMetadata().Description }
func (i scenarioItem) FilterValue() string { return i.scenario.GetMetadata().Name }

// Messages
type bootstrapDoneMsg struct {
	kubeconfig string
	err        error
}

type checkResultMsg struct {
	result scenario.Result
}

type tickMsg time.Time

type progressTickMsg time.Time
type finalDelayMsg time.Time

// NewAppModel creates a new TUI model with the enhanced architecture.
func NewAppModel() AppModel {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	keymap := DefaultKeyMap()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return AppModel{
		theme:              theme,
		styles:             styles,
		keymap:             keymap,
		layout:             NewLayout(80, 24),
		view:               ViewVersionSelect,
		focus:              FocusSidebar,
		versions:           cluster.SupportedVersions(),
		checkInterval:      2 * time.Second,
		header:             components.NewHeaderModel(),
		sidebar:            components.NewSidebarModel(),
		content:            components.NewContentModel(),
		terminal:           components.NewTerminalModel(),
		statusbar:          components.NewStatusBarModel(),
		success:            components.NewSuccessModel(),
		bootstrap:          components.NewProgressModel(),
		completedScenarios: make(map[string]bool),
	}
}

// SetTerminalProgram sets the tea.Program reference on the terminal for async output refresh.
func (m *AppModel) SetTerminalProgram(p *tea.Program) {
	m.terminal.SetProgram(p)
}

// Init initializes the model.
func (m AppModel) Init() tea.Cmd {
	// Note: Don't call tea.EnterAltScreen here since main.go uses tea.WithAltScreen()
	return m.bootstrap.Init()
}

// Update handles messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit handling
		// Skip global quit if in terminal to allow shell interrupts
		allowQuit := true
		if m.view == ViewScenarioRunning && m.focus == FocusTerminal {
			allowQuit = false
		}

		if allowQuit && key.Matches(msg, m.keymap.Quit) {
			// Bootstrap: Immediate quit
			if m.view == ViewBootstrap {
				m.quitting = true
				return m, m.cleanup()
			}

			// If already in a confirmation/dialog view, let that view handle the key (usually cancel)
			if m.view == ViewConfirmQuit || m.view == ViewConfirmRestart {
				// Fall through to view-specific update
			} else {
				// For all other views, show confirmation
				m.previousView = m.view // Remember where we came from
				m.view = ViewConfirmQuit
				m.confirmSelection = 1 // Default to No
				return m, nil
			}
		}

		// Tab for focus switching (Sidebar → Content → Terminal → Sidebar)
		if key.Matches(msg, m.keymap.Tab) && m.view == ViewScenarioRunning {
			switch m.focus {
			case FocusSidebar:
				m.focus = FocusContent
			case FocusContent:
				m.focus = FocusTerminal
			case FocusTerminal:
				m.focus = FocusSidebar
			}
			m.updateFocusStyles()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = NewLayout(msg.Width, msg.Height)
		m.updateComponentSizes()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.bootstrap, cmd = m.bootstrap.Update(msg)
		cmds = append(cmds, cmd)

	case bootstrapDoneMsg:
		return m.handleBootstrapDone(msg)

	case checkResultMsg:
		return m.handleCheckResult(msg)

	case tickMsg:
		if m.view == ViewScenarioRunning {
			return m, m.checkScenario()
		}

	case progressTickMsg:
		if m.view == ViewBootstrap {
			steps := m.bootstrap.GetSteps()

			// If we are past the last step, check if we can finish
			if m.bootstrapStep >= len(steps) {
				if m.bootstrapRealDone {
					return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
						return finalDelayMsg(t)
					})
				}
				// Waiting for real job to finish...
				m.bootstrap.SetSubtitle("Finalizing cluster setup...")
				return m, nil
			}

			// Mark previous step as complete (the one that was active)
			if m.bootstrapStep > 0 && m.bootstrapStep-1 < len(steps) {
				steps[m.bootstrapStep-1].Complete = true
				steps[m.bootstrapStep-1].Active = false
			}

			// Mark current step as active
			steps[m.bootstrapStep].Active = true

			m.bootstrap.SetSteps(steps)

			// Calculate progress as percentage of completed steps
			// bootstrapStep is the current step (0-indexed), so (bootstrapStep+1)/total
			pct := float64(m.bootstrapStep+1) / float64(len(steps))
			m.bootstrap.SetPercent(pct)

			m.bootstrapStep++
			return m, m.tickProgress()
		}

	case finalDelayMsg:
		return m.finalizeBootstrap()

	case scenarioStartedMsg:
		if msg.err != nil {
			m.content.SetStatus(fmt.Sprintf("Failed to start scenario: %v", msg.err), false)
			return m, nil
		}
		m.content.SetStatus("Scenario started. Use kubectl in the terminal below to investigate!", false)
		return m, tea.Tick(m.checkInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case components.TerminalOutputMsg:
		// Terminal has new output, just return to trigger re-render
		return m, nil
	}

	// Handle view-specific updates
	switch m.view {
	case ViewVersionSelect:
		return m.updateVersionSelect(msg)
	case ViewBootstrap:
		return m.updateBootstrap(msg)
	case ViewDashboard:
		return m.updateDashboard(msg)
	case ViewScenarioRunning:
		return m.updateScenarioRunning(msg)
	case ViewSuccess:
		return m.updateSuccess(msg)
	case ViewConfirmRestart:
		return m.updateConfirmRestart(msg)
	case ViewConfirmQuit:
		return m.updateConfirmQuit(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) updateComponentSizes() {
	m.header.SetWidth(m.width)
	m.sidebar.SetSize(m.layout.SidebarWidth, m.layout.MainAreaHeight())
	// Calculate split content areas manually to ensure correctness
	mainH := m.layout.MainAreaHeight()
	infoH := mainH * 40 / 100
	if infoH < 8 {
		infoH = 8
	}
	termH := mainH - infoH

	// Content gets the info panel height (upper area)
	m.content.SetSize(m.layout.ContentWidth, infoH)
	// Terminal gets the terminal height (lower area)
	m.terminal.SetSize(m.layout.ContentWidth, termH)

	m.statusbar.SetWidth(m.width)
	m.success.SetSize(m.width, m.height)
	m.bootstrap.SetWidth(m.width)
}

func (m *AppModel) updateFocusStyles() {
	m.sidebar.SetFocus(m.focus == FocusSidebar)
	m.content.SetFocus(m.focus == FocusContent)
	m.terminal.SetFocus(m.focus == FocusTerminal)
}

func (m AppModel) handleBootstrapDone(msg bootstrapDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.bootstrapErr = msg.err
		return m, nil
	}

	// Create K8s client
	client, err := k8s.NewClientFromKubeconfig(msg.kubeconfig)
	if err != nil {
		m.bootstrapErr = err
		return m, nil
	}
	m.k8sClient = client
	m.kubeconfig = msg.kubeconfig
	m.terminal.SetKubeconfig(msg.kubeconfig)
	m.registry = scenario.NewRegistry(client.Clientset)
	m.engineInstance = engine.NewEngine(m.registry)

	// Initialize state manager and load state
	m.stateManager, err = state.NewManager("")
	if err == nil {
		if st, err := m.stateManager.Load(); err == nil {
			m.completedScenarios = st.CompletedScenarios
		}
	}

	// Build sidebar items from categories
	m.buildSidebarItems()

	// Set header version
	m.header.SetVersion(m.versions[m.selectedVersion].Version)

	// Mark bootstrap as finished
	m.bootstrapRealDone = true

	// Check if animation is already checking for us
	// We do verify explicitly here in case animation ended long ago
	steps := m.bootstrap.GetSteps()
	if m.bootstrapStep >= len(steps) {
		// Animation finished waiting, trigger completion
		m.bootstrap.SetSubtitle("Cluster ready!")

		// Ensure visual 100% just in case
		for i := range steps {
			steps[i].Complete = true
			steps[i].Active = false
		}
		m.bootstrap.SetSteps(steps)
		m.bootstrap.SetPercent(1.0)

		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return finalDelayMsg(t)
		})
	}

	// If animation is still running, do nothing. It will catch m.bootstrapRealDone flag.
	return m, nil
}

func (m AppModel) finalizeBootstrap() (tea.Model, tea.Cmd) {
	// Switch to dashboard view
	m.view = ViewDashboard
	m.focus = FocusSidebar
	m.updateFocusStyles()
	return m, nil
}

func (m *AppModel) buildSidebarItems() {
	// Group scenarios by category
	catMap := make(map[string][]scenario.Scenario)
	preferredOrder := []string{"Networking", "Lifecycle", "Scheduling", "Security", "Storage", "Ops", "Resources", "Kernel"}

	for _, s := range m.registry.List() {
		cat := s.GetMetadata().Category
		if cat == "" {
			cat = "Uncategorized"
		}
		catMap[cat] = append(catMap[cat], s)
	}

	var items []components.SidebarItem
	for _, cat := range preferredOrder {
		if scenarios, ok := catMap[cat]; ok {
			catItem := components.SidebarItem{
				ID:         cat,
				Title:      cat,
				IsCategory: true,
			}
			for _, s := range scenarios {
				catItem.Children = append(catItem.Children, components.SidebarItem{
					ID:          s.GetMetadata().ID,
					Title:       s.GetMetadata().Name,
					Description: s.GetMetadata().Description,
					Category:    cat,
					Completed:   m.completedScenarios[s.GetMetadata().ID],
				})
			}
			items = append(items, catItem)
			delete(catMap, cat)
		}
	}

	// Add remaining categories
	for cat, scenarios := range catMap {
		catItem := components.SidebarItem{
			ID:         cat,
			Title:      cat,
			IsCategory: true,
		}
		for _, s := range scenarios {
			catItem.Children = append(catItem.Children, components.SidebarItem{
				ID:          s.GetMetadata().ID,
				Title:       s.GetMetadata().Name,
				Description: s.GetMetadata().Description,
				Category:    cat,
				Completed:   m.completedScenarios[s.GetMetadata().ID],
			})
		}
		items = append(items, catItem)
	}

	m.sidebar.SetItems(items)
}

func (m AppModel) handleCheckResult(msg checkResultMsg) (tea.Model, tea.Cmd) {
	m.lastCheckResult = msg.result

	if m.engineInstance != nil {
		elapsed := m.engineInstance.GetElapsedTime()
		m.content.SetStatus(msg.result.Message, msg.result.Solved)

		if msg.result.Solved {
			// Persist completion state
			if m.stateManager != nil {
				_ = m.stateManager.MarkScenarioCompleted(m.currentScenario.GetMetadata().ID)
			}
			m.completedScenarios[m.currentScenario.GetMetadata().ID] = true

			m.success.SetScenario(m.currentScenario.GetMetadata().Name)
			m.success.SetMessage(msg.result.Message)
			m.success.SetElapsedTime(elapsed)
			m.view = ViewSuccess
			return m, nil
		}
	}

	return m, tea.Tick(m.checkInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m AppModel) updateVersionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keymap.Up):
			if m.selectedVersion > 0 {
				m.selectedVersion--
			}
		case key.Matches(keyMsg, m.keymap.Down):
			if m.selectedVersion < len(m.versions)-1 {
				m.selectedVersion++
			}
		case key.Matches(keyMsg, m.keymap.Enter):
			m.view = ViewBootstrap
			m.bootstrap.SetTitle("Preparing Training Environment")
			m.bootstrap.SetSubtitle(fmt.Sprintf("Creating Kind cluster (%s)...", m.versions[m.selectedVersion].Version))
			// Define steps - first two are already complete
			steps := []components.ProgressStep{
				{Label: "Docker detected", Complete: true},
				{Label: "Kind installed", Complete: true},
				{Label: "Pulling node image", Active: true},
				{Label: "Starting control plane"},
				{Label: "Configuring kubeconfig"},
			}
			m.bootstrap.SetSteps(steps)
			// Start from step 2 (0-indexed) since first two steps are complete
			// This means bootstrapStep represents the NEXT step to process
			m.bootstrapStep = 2
			// Initial percent: step 2 out of 5 steps = ~33%
			m.bootstrap.SetPercent(float64(m.bootstrapStep) / float64(len(steps)))
			return m, tea.Batch(
				m.doBootstrap(),
				m.tickProgress(),
			)
		}
	}
	return m, nil
}

func (m AppModel) updateBootstrap(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.bootstrap, cmd = m.bootstrap.Update(msg)
	return m, cmd
}

func (m AppModel) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keymap.Enter) {
			// Start selected scenario
			if item := m.sidebar.SelectedItem(); item != nil && !item.IsCategory {
				for _, s := range m.registry.List() {
					if s.GetMetadata().ID == item.ID {
						m.currentScenario = s

						// Check if already completed
						if m.completedScenarios[s.GetMetadata().ID] {
							m.view = ViewConfirmRestart
							m.confirmSelection = 1 // Default to No (Safe)
							return m, nil
						}

						return m.startSelectedScenario(s)
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.sidebar, cmd = m.sidebar.Update(msg)
	return m, cmd
}

func (m AppModel) updateScenarioRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Only handle shortcuts if NOT focused on terminal
		if m.focus != FocusTerminal {
			switch {
			case key.Matches(keyMsg, m.keymap.Check):
				return m, m.checkScenario()
			case key.Matches(keyMsg, m.keymap.ToggleHints):
				m.content.ToggleHints()
			case key.Matches(keyMsg, m.keymap.NextHint):
				m.content.NextHint()
			case key.Matches(keyMsg, m.keymap.PrevHint):
				m.content.PrevHint()
			case key.Matches(keyMsg, m.keymap.Escape):
				// Return to dashboard
				ctx := context.Background()
				if m.engineInstance != nil {
					_ = m.engineInstance.Cleanup(ctx)
				}
				m.terminal.Stop()
				m.header.SetTitle("🥋 K8s-Dojo")
				m.header.ResetTimer()
				m.view = ViewDashboard
				m.currentScenario = nil
				return m, nil
			}
		}
	}

	// Update focused component
	switch m.focus {
	case FocusSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd
	case FocusTerminal:
		var cmd tea.Cmd
		cmd = m.terminal.Update(msg) // Terminal update returns cmd only, mutates state pointer
		return m, cmd
	default: // FocusContent
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		return m, cmd
	}
}

func (m AppModel) updateSuccess(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		// Navigation
		case key.Matches(keyMsg, m.keymap.Left), key.Matches(keyMsg, m.keymap.ShiftTab), key.Matches(keyMsg, m.keymap.Up):
			m.success.PrevButton()
			return m, nil
		case key.Matches(keyMsg, m.keymap.Right), key.Matches(keyMsg, m.keymap.Tab), key.Matches(keyMsg, m.keymap.Down):
			m.success.NextButton()
			return m, nil

		case key.Matches(keyMsg, m.keymap.Enter):
			if m.success.SelectedButton() == 1 {
				// Retry
				return m.handleRetry()
			}
			// Continue
			return m.handleReturnToDashboard()

		case key.Matches(keyMsg, m.keymap.ReturnMenu):
			return m.handleReturnToDashboard()

		case key.Matches(keyMsg, m.keymap.Retry):
			return m.handleRetry()
		}
	}
	return m, nil
}

func (m AppModel) handleReturnToDashboard() (tea.Model, tea.Cmd) {
	// Mark current scenario as completed
	if m.currentScenario != nil {
		m.completedScenarios[m.currentScenario.GetMetadata().ID] = true
	}

	ctx := context.Background()
	if m.engineInstance != nil {
		_ = m.engineInstance.Cleanup(ctx)
	}

	m.header.SetTitle("🥋 K8s-Dojo")
	m.header.ResetTimer()

	// Refresh sidebar to show updated status
	m.buildSidebarItems()

	m.view = ViewDashboard
	m.focus = FocusSidebar // Explicitly set focus to Sidebar
	m.updateFocusStyles()  // Apply focus styles

	m.currentScenario = nil
	m.lastCheckResult = scenario.Result{}

	return m, nil
}

func (m AppModel) handleRetry() (tea.Model, tea.Cmd) {
	// Restart same scenario
	m.header.StartTimer()
	m.view = ViewScenarioRunning
	return m, tea.Batch(
		m.startScenario(),
		tea.Tick(m.checkInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
	)
}

// Commands

func (m AppModel) doBootstrap() tea.Cmd {
	return func() tea.Msg {
		manager := cluster.NewManager()
		kubeconfig, err := manager.EnsureCluster(m.versions[m.selectedVersion])
		return bootstrapDoneMsg{kubeconfig: kubeconfig, err: err}
	}
}

type scenarioStartedMsg struct {
	err error
}

func (m AppModel) startScenario() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engineInstance.StartScenario(ctx, m.currentScenario.GetMetadata().ID)
		return scenarioStartedMsg{err: err}
	}
}

func (m AppModel) startSelectedScenario(s scenario.Scenario) (tea.Model, tea.Cmd) {
	m.view = ViewScenarioRunning
	m.header.SetTitle("🥋 " + s.GetMetadata().Name)
	m.header.StartTimer()

	// Setup content panel
	m.content.SetScenario(
		s.GetMetadata().Name,
		s.GetMetadata().Description,
		s.GetNamespace(),
	)
	m.content.SetCommands([]string{
		fmt.Sprintf("kubectl config use-context kind-%s", cluster.ClusterName),
		fmt.Sprintf("kubectl get pods -n %s", s.GetNamespace()),
	})
	m.content.SetHints(s.GetMetadata().Hints)
	m.content.SetStatus("Setting up scenario environment...", false)

	// Auto-focus terminal for immediate input
	m.focus = FocusTerminal
	m.updateFocusStyles()

	return m, tea.Batch(
		m.startScenario(),
		m.terminal.Start(),
		// Note: We DO NOT start the check ticker here.
		// The check ticker will be started by handleCheckResult when startScenario completes.
		// This prevents "no scenario is running" errors if checking happens before start finishes.
	)
}

func (m AppModel) checkScenario() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := m.engineInstance.Check(ctx)
		if err != nil {
			return checkResultMsg{result: scenario.Result{Solved: false, Message: err.Error()}}
		}
		return checkResultMsg{result: result}
	}
}

func (m AppModel) tickProgress() tea.Cmd {
	return tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg {
		return progressTickMsg(t)
	})
}

func (m AppModel) cleanup() tea.Cmd {
	return func() tea.Msg {
		if m.engineInstance != nil {
			ctx := context.Background()
			_ = m.engineInstance.Cleanup(ctx)
		}
		return tea.Quit()
	}
}

// View renders the UI.
func (m AppModel) View() string {
	if m.quitting {
		return m.styles.TextMuted.Render("Cleaning up... Goodbye!") + "\n"
	}

	if m.layout.IsTooSmall() {
		return m.styles.Error.Render(fmt.Sprintf(
			"Terminal too small. Minimum: %dx%d, Current: %dx%d",
			MinWidth, MinHeight, m.width, m.height,
		))
	}

	switch m.view {
	case ViewVersionSelect:
		return m.viewVersionSelect()
	case ViewBootstrap:
		return m.viewBootstrap()
	case ViewDashboard:
		return m.viewDashboard()
	case ViewScenarioRunning:
		return m.viewScenarioRunning()
	case ViewSuccess:
		return m.viewSuccess()
	case ViewConfirmRestart:
		return m.viewConfirmRestart()
	case ViewConfirmQuit:
		return m.viewConfirmQuit()
	}

	return ""
}

func (m AppModel) viewVersionSelect() string {
	// Centered version selection
	var content string

	title := m.styles.Title.Render("🥋  K 8 s - D o j o")
	subtitle := m.styles.Text.Render("Master Kubernetes Troubleshooting")

	var options string
	for i, v := range m.versions {
		cursor := "   "
		label := v.Version
		if v.IsLatest {
			label += " (Latest)"
		}
		if i == m.selectedVersion {
			cursor = " › "
			label = m.styles.ActiveItem.Render(label)
		} else {
			label = m.styles.Text.Render(label)
		}
		options += cursor + label + "\n"
	}

	boxStyle := m.styles.Box.Width(30).Align(lipgloss.Left)

	// Manually center the title since the box is now left-aligned
	titleText := m.styles.Subtitle.Render("Select Kubernetes Version")
	centeredTitle := lipgloss.PlaceHorizontal(26, lipgloss.Center, titleText)

	boxContent := centeredTitle + "\n\n" + options

	content = lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		subtitle,
		"",
		boxStyle.Render(boxContent),
	)

	// Status bar
	m.statusbar.SetKeys(components.ContextualStatusBar("version-select"))
	statusBar := m.statusbar.View()

	// Center content and add status bar at bottom
	mainArea := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, content)
	return lipgloss.JoinVertical(lipgloss.Left, mainArea, statusBar)
}

func (m AppModel) viewBootstrap() string {
	if m.bootstrapErr != nil {
		errContent := m.styles.Error.Render("❌ Error: " + m.bootstrapErr.Error())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, errContent)
	}

	content := m.bootstrap.View()
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m AppModel) viewDashboard() string {
	// Header
	header := m.header.View()

	// Main area: Sidebar + Content (but in dashboard, we show list in content area)
	sidebar := m.sidebar.View()

	// For dashboard, content shows selected item preview or instructions (top 40%)
	contentStyle := m.styles.Content.
		Width(m.layout.ContentWidth - 2).
		Height(m.layout.InfoHeight - 2)

	var contentText string
	if item := m.sidebar.SelectedItem(); item != nil && !item.IsCategory {
		contentText = m.styles.Title.Render("🔧 "+item.Title) + "\n\n"
		contentText += m.styles.Text.Render(item.Description) + "\n\n"
		contentText += m.styles.Highlight.Render("Press Enter to start")
	} else if item != nil && item.IsCategory {
		contentText = m.styles.Title.Render(CategoryIcon(item.Title)+" "+item.Title) + "\n\n"
		contentText += m.styles.TextMuted.Render("Use h/l to expand/collapse, j/k to navigate")
	} else {
		contentText = m.styles.TextMuted.Render("Select a scenario to begin")
	}
	content := contentStyle.Render(contentText)

	// In dashboard, we also show the terminal panel to maintain layout consistency
	terminal := m.terminal.View()
	rightSide := lipgloss.JoinVertical(lipgloss.Left, content, terminal)

	// Join sidebar and right side
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightSide)

	// Status bar
	m.statusbar.SetKeys(components.ContextualStatusBar("scenario-select"))
	statusBar := m.statusbar.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, mainArea, statusBar)
}

func (m AppModel) viewScenarioRunning() string {
	// Header
	header := m.header.View()

	// Main area: Sidebar + Content
	sidebar := m.sidebar.View()
	content := m.content.View()
	terminal := m.terminal.View()

	// Right side is content (top) + terminal (bottom)
	rightSide := lipgloss.JoinVertical(lipgloss.Left, content, terminal)

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightSide)

	// Status bar
	m.statusbar.SetKeys(components.ContextualStatusBar("scenario-running"))
	statusBar := m.statusbar.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, mainArea, statusBar)
}

func (m AppModel) viewSuccess() string {
	return m.success.View()
}

func (m AppModel) updateConfirmRestart(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		// Navigation
		case key.Matches(keyMsg, m.keymap.Left), key.Matches(keyMsg, m.keymap.ShiftTab), key.Matches(keyMsg, m.keymap.Up):
			m.confirmSelection = (m.confirmSelection - 1 + 2) % 2
			return m, nil
		case key.Matches(keyMsg, m.keymap.Right), key.Matches(keyMsg, m.keymap.Tab), key.Matches(keyMsg, m.keymap.Down):
			m.confirmSelection = (m.confirmSelection + 1) % 2
			return m, nil

		case key.Matches(keyMsg, m.keymap.Enter):
			if m.confirmSelection == 0 {
				return m.startSelectedScenario(m.currentScenario)
			}
			// Cancel
			m.view = ViewDashboard
			m.currentScenario = nil
			return m, nil

		case key.Matches(keyMsg, m.keymap.Escape), key.Matches(keyMsg, m.keymap.Quit):
			// Cancel
			m.view = ViewDashboard
			m.currentScenario = nil
			return m, nil
		// Allow 'y' and 'n' as distinct from general keymap
		case keyMsg.String() == "y":
			return m.startSelectedScenario(m.currentScenario)
		case keyMsg.String() == "n":
			m.view = ViewDashboard
			m.currentScenario = nil
			return m, nil
		}
	}
	return m, nil
}

func (m AppModel) viewConfirmRestart() string {
	// Re-use success styles for consistent look, or simpler box
	title := m.styles.Title.Render("⚠️  Restart Scenario?")

	msg := fmt.Sprintf("\nYou have already completed\n'%s'.\n\nRestarting will reset the environment.\nAre you sure?\n", m.currentScenario.GetMetadata().Name)

	yesBtn := "[ Yes (y) ]"
	noBtn := "[ No (n) ]"

	if m.confirmSelection == 0 {
		yesBtn = m.styles.ActiveItem.Render(yesBtn)
		noBtn = m.styles.TextMuted.Render(noBtn)
	} else {
		yesBtn = m.styles.TextMuted.Render(yesBtn)
		noBtn = m.styles.ActiveItem.Render(noBtn)
	}

	buttons := yesBtn + "    " + noBtn

	boxStyle := m.styles.Box.Width(50).Align(lipgloss.Center).BorderForeground(lipgloss.Color("#fab387")) // Peach/Orange for warning
	boxContent := title + "\n" + m.styles.Text.Render(msg) + "\n" + buttons

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(boxContent))
}

func (m AppModel) updateConfirmQuit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		// Navigation
		case key.Matches(keyMsg, m.keymap.Left), key.Matches(keyMsg, m.keymap.ShiftTab), key.Matches(keyMsg, m.keymap.Up):
			m.confirmSelection = (m.confirmSelection - 1 + 2) % 2
			return m, nil
		case key.Matches(keyMsg, m.keymap.Right), key.Matches(keyMsg, m.keymap.Tab), key.Matches(keyMsg, m.keymap.Down):
			m.confirmSelection = (m.confirmSelection + 1) % 2
			return m, nil

		case key.Matches(keyMsg, m.keymap.Enter):
			if m.confirmSelection == 0 {
				// Yes, quit
				m.quitting = true
				return m, m.cleanup()
			}
			// Cancel
			m.view = ViewDashboard
			return m, nil

		case key.Matches(keyMsg, m.keymap.Escape), key.Matches(keyMsg, m.keymap.Quit):
			// Cancel
			m.view = m.previousView
			return m, nil

		// Allow 'y' and 'n'
		case keyMsg.String() == "y":
			m.quitting = true
			return m, m.cleanup()
		case keyMsg.String() == "n":
			m.view = m.previousView
			return m, nil
		}
	}
	return m, nil
}

func (m AppModel) viewConfirmQuit() string {
	title := m.styles.Title.Render("👋  Quit K8s-Dojo?")

	msg := "\nAre you sure you want to exit?\n"

	yesBtn := "[ Yes (y) ]"
	noBtn := "[ No (n) ]"

	if m.confirmSelection == 0 {
		yesBtn = m.styles.ActiveItem.Render(yesBtn)
		noBtn = m.styles.TextMuted.Render(noBtn)
	} else {
		yesBtn = m.styles.TextMuted.Render(yesBtn)
		noBtn = m.styles.ActiveItem.Render(noBtn)
	}

	buttons := yesBtn + "    " + noBtn

	boxStyle := m.styles.Box.Width(40).Align(lipgloss.Center).BorderForeground(lipgloss.Color("#fab387"))
	boxContent := title + "\n" + m.styles.Text.Render(msg) + "\n" + buttons

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(boxContent))
}
