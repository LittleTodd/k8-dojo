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
	"k8s-dojo/pkg/tui/components"
)

// FocusArea represents which panel is focused.
type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
)

// View represents the current TUI view.
type View int

const (
	ViewVersionSelect View = iota
	ViewBootstrap
	ViewDashboard
	ViewScenarioRunning
	ViewSuccess
)

// AppModel is the main Bubbletea model with the new component architecture.
type AppModel struct {
	// Theme and styles
	theme  Theme
	styles Styles
	keymap KeyMap
	layout Layout

	// Current view and focus
	view  View
	focus FocusArea

	// Version selection
	versions        []cluster.SupportedVersion
	selectedVersion int

	// Bootstrap
	bootstrap    components.ProgressModel
	bootstrapErr error

	// Components
	header    components.HeaderModel
	sidebar   components.SidebarModel
	content   components.ContentModel
	statusbar components.StatusBarModel
	success   components.SuccessModel

	// Scenario list (for dashboard)
	scenarioList list.Model

	// Cluster & Engine
	clusterManager *cluster.Manager
	k8sClient      *k8s.Client
	engineInstance *engine.Engine
	registry       *scenario.Registry

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

// NewAppModel creates a new TUI model with the enhanced architecture.
func NewAppModel() AppModel {
	theme := DefaultTheme()
	styles := NewStyles(theme)
	keymap := DefaultKeyMap()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return AppModel{
		theme:         theme,
		styles:        styles,
		keymap:        keymap,
		layout:        NewLayout(80, 24),
		view:          ViewVersionSelect,
		focus:         FocusSidebar,
		versions:      cluster.SupportedVersions(),
		checkInterval: 2 * time.Second,
		header:        components.NewHeaderModel(),
		sidebar:       components.NewSidebarModel(),
		content:       components.NewContentModel(),
		statusbar:     components.NewStatusBarModel(),
		success:       components.NewSuccessModel(),
		bootstrap:     components.NewProgressModel(),
	}
}

// Init initializes the model.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.bootstrap.Init(),
	)
}

// Update handles messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit handling
		if key.Matches(msg, m.keymap.Quit) && m.view != ViewBootstrap {
			m.quitting = true
			return m, m.cleanup()
		}

		// Tab for focus switching
		if key.Matches(msg, m.keymap.Tab) && m.view == ViewScenarioRunning {
			if m.focus == FocusSidebar {
				m.focus = FocusContent
			} else {
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
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) updateComponentSizes() {
	m.header.SetWidth(m.width)
	m.sidebar.SetSize(m.layout.SidebarWidth, m.layout.MainAreaHeight())
	m.content.SetSize(m.layout.ContentWidth, m.layout.MainAreaHeight())
	m.statusbar.SetWidth(m.width)
	m.success.SetSize(m.width, m.height)
	m.bootstrap.SetWidth(m.width)
}

func (m *AppModel) updateFocusStyles() {
	m.sidebar.SetFocus(m.focus == FocusSidebar)
	m.content.SetFocus(m.focus == FocusContent)
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
	m.registry = scenario.NewRegistry(client.Clientset)
	m.engineInstance = engine.NewEngine(m.registry)

	// Build sidebar items from categories
	m.buildSidebarItems()

	// Set header version
	m.header.SetVersion(m.versions[m.selectedVersion].Version)

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
					Completed:   false,
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
				Completed:   false,
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
			m.bootstrap.SetSubtitle(fmt.Sprintf("Creating Kind cluster (v%s)...", m.versions[m.selectedVersion].Version))
			m.bootstrap.SetSteps([]components.ProgressStep{
				{Label: "Docker detected", Complete: true},
				{Label: "Kind installed", Complete: true},
				{Label: "Pulling node image", Active: true},
				{Label: "Starting control plane", Complete: false},
				{Label: "Configuring kubeconfig", Complete: false},
			})
			m.bootstrap.SetPercent(0.2)
			return m, m.doBootstrap()
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
				// Find scenario by ID
				for _, s := range m.registry.List() {
					if s.GetMetadata().ID == item.ID {
						m.currentScenario = s
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
						m.content.SetStatus("Investigating...", false)

						return m, tea.Batch(
							m.startScenario(),
							tea.Tick(m.checkInterval, func(t time.Time) tea.Msg {
								return tickMsg(t)
							}),
						)
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
			m.header.SetTitle("🥋 K8s-Dojo")
			m.header.ResetTimer()
			m.view = ViewDashboard
			m.currentScenario = nil
			return m, nil
		}
	}

	// Update focused component
	if m.focus == FocusSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		return m, cmd
	} else {
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		return m, cmd
	}
}

func (m AppModel) updateSuccess(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keymap.Enter), key.Matches(keyMsg, m.keymap.ReturnMenu):
			ctx := context.Background()
			if m.engineInstance != nil {
				_ = m.engineInstance.Cleanup(ctx)
			}
			m.header.SetTitle("🥋 K8s-Dojo")
			m.header.ResetTimer()
			m.view = ViewDashboard
			m.currentScenario = nil
			m.lastCheckResult = scenario.Result{}
			return m, nil
		case key.Matches(keyMsg, m.keymap.Retry):
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
	}
	return m, nil
}

// Commands

func (m AppModel) doBootstrap() tea.Cmd {
	return func() tea.Msg {
		manager := cluster.NewManager()
		kubeconfig, err := manager.EnsureCluster(m.versions[m.selectedVersion])
		return bootstrapDoneMsg{kubeconfig: kubeconfig, err: err}
	}
}

func (m AppModel) startScenario() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engineInstance.StartScenario(ctx, m.currentScenario.GetMetadata().ID)
		if err != nil {
			return checkResultMsg{result: scenario.Result{Solved: false, Message: err.Error()}}
		}
		return checkResultMsg{result: scenario.Result{Solved: false, Message: "Scenario started. Open another terminal and use kubectl to investigate!"}}
	}
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

	boxStyle := m.styles.Box.Width(30).Align(lipgloss.Center)
	boxContent := m.styles.Subtitle.Render("Select Kubernetes Version") + "\n\n" + options

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

	// For dashboard, content shows selected item preview or instructions
	contentStyle := m.styles.Content.
		Width(m.layout.ContentWidth - 2).
		Height(m.layout.MainAreaHeight() - 2)

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

	// Join sidebar and content horizontally
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

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

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	// Status bar
	m.statusbar.SetKeys(components.ContextualStatusBar("scenario-running"))
	statusBar := m.statusbar.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, mainArea, statusBar)
}

func (m AppModel) viewSuccess() string {
	return m.success.View()
}
