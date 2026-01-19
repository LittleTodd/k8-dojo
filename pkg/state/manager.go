package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// State represents the persistent application state.
type State struct {
	CompletedScenarios map[string]bool `json:"completed_scenarios"`
	LastActiveScenario string          `json:"last_active_scenario,omitempty"`
}

// Manager handles saving and loading of application state.
type Manager struct {
	path string
	mu   sync.RWMutex
}

// NewManager creates a new state manager.
// If path is empty, it defaults to ~/.k8s-dojo/state.json
func NewManager(path string) (*Manager, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(home, ".k8s-dojo", "state.json")
	}

	return &Manager{
		path: path,
	}, nil
}

// Load loads the state from disk.
// If the file doesn't exist, it returns an empty state.
func (m *Manager) Load() (*State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Default empty state
	state := &State{
		CompletedScenarios: make(map[string]bool),
	}

	data, err := os.ReadFile(m.path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if state.CompletedScenarios == nil {
		state.CompletedScenarios = make(map[string]bool)
	}

	return state, nil
}

// Save persists the state to disk.
func (m *Manager) Save(state *State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	if err := os.WriteFile(m.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// MarkScenarioCompleted updates the state to mark a scenario as completed.
func (m *Manager) MarkScenarioCompleted(scenarioID string) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	state.CompletedScenarios[scenarioID] = true

	return m.Save(state)
}
