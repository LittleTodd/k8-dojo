// Package tui provides the terminal user interface using Bubbletea.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"k8s-dojo/pkg/cluster"
	"k8s-dojo/pkg/engine"
	"k8s-dojo/pkg/k8s"
	"k8s-dojo/pkg/scenario"
)

// View represents the current TUI view.
type View int

const (
	ViewVersionSelect View = iota
	ViewBootstrap
	ViewCategorySelect
	ViewScenarioSelect
	ViewScenarioRunning
	ViewSuccess
)

// Model is the main Bubbletea model.
type Model struct {
	// Theme and styles
	theme  Theme
	styles Styles

	// Current view
	view View

	// Version selection
	versions        []cluster.SupportedVersion
	selectedVersion int

	// Bootstrap
	spinner      spinner.Model
	bootstrapMsg string
	bootstrapErr error

	// Cluster & Engine
	clusterManager *cluster.Manager
	k8sClient      *k8s.Client
	engine         *engine.Engine
	registry       *scenario.Registry

	// Category selection
	categories       []string
	selectedCategory int

	// Scenario selection
	scenarioList list.Model

	// Running scenario
	currentScenario scenario.Scenario
	lastCheckResult scenario.Result
	checkInterval   time.Duration
	elapsedTime     time.Duration

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

// NewModel creates a new TUI model.
func NewModel() Model {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return Model{
		theme:         theme,
		styles:        styles,
		view:          ViewVersionSelect,
		versions:      cluster.SupportedVersions(),
		spinner:       s,
		checkInterval: 2 * time.Second,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.spinner.Tick,
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, m.cleanup()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case bootstrapDoneMsg:
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
		m.engine = engine.NewEngine(m.registry)

		// Extract unique categories
		catMap := make(map[string]bool)
		for _, s := range m.registry.List() {
			cat := s.GetMetadata().Category
			if cat == "" {
				cat = "Uncategorized"
			}
			catMap[cat] = true
		}
		m.categories = make([]string, 0, len(catMap))
		for c := range catMap {
			m.categories = append(m.categories, c)
		}
		// Sort categories? (Optional, map iteration is random)
		// Let's rely on simple slice for now, maybe sort later or hardcode order?
		// Actually, map iteration is random, which is bad for UI.
		// Let's hardcode a preferred order or sort.
		// For now, let's just use what we found and sort strictly?
		// Better: define order.
		preferredBase := []string{"Networking", "Lifecycle", "Scheduling", "Security", "Storage", "Ops", "Resources", "Kernel"}
		sortedCats := []string{}

		// Add preferred first
		for _, p := range preferredBase {
			if catMap[p] {
				sortedCats = append(sortedCats, p)
				delete(catMap, p)
			}
		}
		// Add remaining
		for c := range catMap {
			sortedCats = append(sortedCats, c)
		}
		m.categories = sortedCats

		m.view = ViewCategorySelect
		return m, nil

	case checkResultMsg:
		m.lastCheckResult = msg.result
		m.elapsedTime = m.engine.GetElapsedTime()
		if msg.result.Solved {
			m.view = ViewSuccess
			return m, nil
		}
		return m, tea.Tick(m.checkInterval, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tickMsg:
		if m.view == ViewScenarioRunning {
			return m, m.checkScenario()
		}
	}

	// Handle view-specific updates
	switch m.view {
	case ViewVersionSelect:
		return m.updateVersionSelect(msg)
	case ViewScenarioSelect:
		return m.updateScenarioSelect(msg)
	case ViewCategorySelect:
		return m.updateCategorySelect(msg)
	case ViewScenarioRunning:
		return m.updateScenarioRunning(msg)
	case ViewSuccess:
		return m.updateSuccess(msg)
	}

	return m, nil
}

func (m Model) updateVersionSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedVersion > 0 {
				m.selectedVersion--
			}
		case "down", "j":
			if m.selectedVersion < len(m.versions)-1 {
				m.selectedVersion++
			}
		case "enter":
			m.view = ViewBootstrap
			m.bootstrapMsg = fmt.Sprintf("Creating cluster with Kubernetes %s...", m.versions[m.selectedVersion].Version)
			return m, m.bootstrap()
		}
	}
	return m, nil
}

func (m Model) updateCategorySelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedCategory > 0 {
				m.selectedCategory--
			}
		case "down", "j":
			if m.selectedCategory < len(m.categories)-1 {
				m.selectedCategory++
			}
		case "enter":
			selectedCat := m.categories[m.selectedCategory]

			// Setup list for this category
			delegate := list.NewDefaultDelegate()
			delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
				Foreground(m.theme.Primary).
				BorderForeground(m.theme.Primary)
			delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
				Foreground(m.theme.Secondary).
				BorderForeground(m.theme.Primary)

			items := make([]list.Item, 0)
			for _, s := range m.registry.List() {
				if s.GetMetadata().Category == selectedCat {
					items = append(items, scenarioItem{scenario: s})
				}
			}
			m.scenarioList = list.New(items, delegate, m.width, m.height-4)
			m.scenarioList.Title = fmt.Sprintf("Module: %s", selectedCat)
			m.scenarioList.SetShowStatusBar(false)
			m.scenarioList.Styles.Title = m.styles.Title

			m.view = ViewScenarioSelect
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateScenarioSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "esc" {
			m.view = ViewCategorySelect
			return m, nil
		}
		if keyMsg.String() == "enter" {
			if item, ok := m.scenarioList.SelectedItem().(scenarioItem); ok {
				m.currentScenario = item.scenario
				m.view = ViewScenarioRunning
				return m, tea.Batch(
					m.startScenario(),
					tea.Tick(m.checkInterval, func(t time.Time) tea.Msg {
						return tickMsg(t)
					}),
				)
			}
		}
	}

	var cmd tea.Cmd
	m.scenarioList, cmd = m.scenarioList.Update(msg)
	return m, cmd
}

func (m Model) updateScenarioRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "h":
			// Show hints (TODO: implement hint cycling)
		case "c":
			// Manual check
			return m, m.checkScenario()
		}
	}
	return m, nil
}

func (m Model) updateSuccess(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter", " ":
			// Go back to scenario selection
			ctx := context.Background()
			_ = m.engine.Cleanup(ctx)
			m.view = ViewScenarioSelect
			m.currentScenario = nil
			m.lastCheckResult = scenario.Result{}
		}
	}
	return m, nil
}

// Commands

func (m Model) bootstrap() tea.Cmd {
	return func() tea.Msg {
		m.clusterManager = cluster.NewManager()
		kubeconfig, err := m.clusterManager.EnsureCluster(m.versions[m.selectedVersion])
		return bootstrapDoneMsg{kubeconfig: kubeconfig, err: err}
	}
}

func (m Model) startScenario() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.engine.StartScenario(ctx, m.currentScenario.GetMetadata().ID)
		if err != nil {
			return checkResultMsg{result: scenario.Result{Solved: false, Message: err.Error()}}
		}
		return checkResultMsg{result: scenario.Result{Solved: false, Message: "Scenario started. Open another terminal and use kubectl to investigate!"}}
	}
}

func (m Model) checkScenario() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		result, err := m.engine.Check(ctx)
		if err != nil {
			return checkResultMsg{result: scenario.Result{Solved: false, Message: err.Error()}}
		}
		return checkResultMsg{result: result}
	}
}

func (m Model) cleanup() tea.Cmd {
	return func() tea.Msg {
		if m.engine != nil {
			ctx := context.Background()
			_ = m.engine.Cleanup(ctx)
		}
		return tea.Quit()
	}
}

// View renders the UI.
func (m Model) View() string {
	if m.quitting {
		return m.styles.TextMuted.Render("Cleaning up... Goodbye!") + "\n"
	}

	switch m.view {
	case ViewVersionSelect:
		return m.viewVersionSelect()
	case ViewBootstrap:
		return m.viewBootstrap()
	case ViewCategorySelect:
		return m.viewCategorySelect()
	case ViewScenarioSelect:
		return m.viewScenarioSelect()
	case ViewScenarioRunning:
		return m.viewScenarioRunning()
	case ViewSuccess:
		return m.viewSuccess()
	}

	return ""
}

func (m Model) viewCategorySelect() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("🥋 K8s-Dojo Modules"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Text.Render("Select a Training Module:"))
	b.WriteString("\n\n")

	for i, c := range m.categories {
		cursor := "  "
		if i == m.selectedCategory {
			cursor = "> "
		}
		label := c
		if i == m.selectedCategory {
			label = m.styles.ActiveItem.Render(label)
		} else {
			label = m.styles.Text.Render(label)
		}
		b.WriteString(cursor + label + "\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("↑/↓: navigate • enter: select • q: quit"))

	return b.String()
}

func (m Model) viewVersionSelect() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("🥋 K8s-Dojo"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Text.Render("Select Kubernetes Version:"))
	b.WriteString("\n\n")

	for i, v := range m.versions {
		cursor := "  "
		if i == m.selectedVersion {
			cursor = "> "
		}
		label := v.Version
		if v.IsLatest {
			label += " (Latest)"
		}
		if i == m.selectedVersion {
			label = m.styles.ActiveItem.Render(label)
		} else {
			label = m.styles.Text.Render(label)
		}
		b.WriteString(cursor + label + "\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Help.Render("↑/↓: navigate • enter: select • q: quit"))

	return b.String()
}

func (m Model) viewBootstrap() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("🥋 K8s-Dojo"))
	b.WriteString("\n\n")

	if m.bootstrapErr != nil {
		b.WriteString(m.styles.Error.Render("❌ Error: " + m.bootstrapErr.Error()))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Help.Render("Press q to quit."))
	} else {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Text.Render(m.bootstrapMsg))
		b.WriteString("\n\n")
		b.WriteString(m.styles.TextMuted.Render("This may take a few minutes on first run..."))
	}

	return b.String()
}

func (m Model) viewScenarioSelect() string {
	return m.scenarioList.View()
}

func (m Model) viewScenarioRunning() string {
	var b strings.Builder

	// Header
	b.WriteString(m.styles.Title.Render("🥋 " + m.currentScenario.GetMetadata().Name))
	b.WriteString("\n\n")

	// Description
	b.WriteString(m.styles.Text.Render(m.currentScenario.GetMetadata().Description))
	b.WriteString("\n\n")

	// Namespace info
	b.WriteString(m.styles.Subtitle.Render("Namespace: " + m.currentScenario.GetNamespace()))
	b.WriteString("\n\n")

	// Status
	b.WriteString(m.styles.Highlight.Render("Status: "))
	b.WriteString(m.styles.Text.Render(m.lastCheckResult.Message))
	b.WriteString("\n\n")

	// Elapsed time
	b.WriteString(m.styles.TextMuted.Render(fmt.Sprintf("Time: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n\n")

	// Instructions
	instructions := "Open another terminal and run:\n\n"
	instructions += fmt.Sprintf("  kubectl config use-context kind-%s\n", cluster.ClusterName)
	instructions += fmt.Sprintf("  kubectl get pods -n %s\n", m.currentScenario.GetNamespace())

	b.WriteString(m.styles.Box.Render(instructions))
	b.WriteString("\n\n")

	// Help
	b.WriteString(m.styles.Help.Render("c: check now • h: hints • q: quit"))

	return b.String()
}

func (m Model) viewSuccess() string {
	var b strings.Builder

	b.WriteString(m.styles.Success.Render("🎉 SUCCESS!"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Text.Render(m.lastCheckResult.Message))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Subtitle.Render(fmt.Sprintf("Completed in: %s", m.elapsedTime.Round(time.Second))))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Help.Render("Press Enter to continue • q: quit"))

	return b.String()
}
