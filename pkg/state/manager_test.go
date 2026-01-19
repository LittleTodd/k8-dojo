package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "k8s-dojo-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	statePath := filepath.Join(tmpDir, "state.json")

	// Test NewManager
	mgr, err := NewManager(statePath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test Load on non-existent file
	state, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load failed on new file: %v", err)
	}
	if len(state.CompletedScenarios) != 0 {
		t.Errorf("Expected empty completed scenarios, got %d", len(state.CompletedScenarios))
	}

	// Test MarkScenarioCompleted
	if err := mgr.MarkScenarioCompleted("test-scenario"); err != nil {
		t.Fatalf("MarkScenarioCompleted failed: %v", err)
	}

	// Test Load after save
	state, err = mgr.Load()
	if err != nil {
		t.Fatalf("Load failed after save: %v", err)
	}
	if !state.CompletedScenarios["test-scenario"] {
		t.Error("Expected test-scenario to be completed")
	}

	// Test persistence with new manager instance
	mgr2, err := NewManager(statePath)
	if err != nil {
		t.Fatalf("NewManager 2 failed: %v", err)
	}
	state2, err := mgr2.Load()
	if err != nil {
		t.Fatalf("Load 2 failed: %v", err)
	}
	if !state2.CompletedScenarios["test-scenario"] {
		t.Error("Expected test-scenario to be completed in new instance")
	}
}
