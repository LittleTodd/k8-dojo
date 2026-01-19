// Package engine provides the game engine for running scenarios.
package engine

import (
	"context"
	"fmt"
	"time"

	"k8s-dojo/pkg/scenario"
)

// State represents the current state of the game.
type State string

const (
	StateIdle      State = "idle"
	StateRunning   State = "running"
	StateValidated State = "validated"
	StateCleaning  State = "cleaning"
)

// Engine manages the lifecycle of scenarios.
type Engine struct {
	registry        *scenario.Registry
	currentScenario scenario.Scenario
	state           State
	startTime       time.Time
}

// NewEngine creates a new game engine.
func NewEngine(registry *scenario.Registry) *Engine {
	return &Engine{
		registry: registry,
		state:    StateIdle,
	}
}

// ListScenarios returns all available scenarios.
func (e *Engine) ListScenarios() []scenario.Scenario {
	return e.registry.List()
}

// StartScenario starts a scenario by its ID.
func (e *Engine) StartScenario(ctx context.Context, id string) error {
	s := e.registry.Get(id)
	if s == nil {
		return fmt.Errorf("scenario not found: %s", id)
	}

	// Ensure clean slate by cleaning up any previous state
	fmt.Printf("Ensuring clean state for scenario: %s\n", s.GetMetadata().Name)
	// We ignore the error here because it's likely "not found" if the scenario wasn't running
	_ = s.Cleanup(ctx)

	// Setup the scenario
	fmt.Printf("Setting up scenario: %s\n", s.GetMetadata().Name)
	if err := s.Setup(ctx); err != nil {
		return fmt.Errorf("failed to setup scenario: %w", err)
	}

	e.currentScenario = s
	e.state = StateRunning
	e.startTime = time.Now()

	return nil
}

// Check validates if the current scenario is solved.
func (e *Engine) Check(ctx context.Context) (scenario.Result, error) {
	if e.currentScenario == nil {
		return scenario.Result{}, fmt.Errorf("no scenario is running")
	}

	result := e.currentScenario.Validate(ctx)
	if result.Solved {
		e.state = StateValidated
	}

	return result, nil
}

// Cleanup cleans up the current scenario.
func (e *Engine) Cleanup(ctx context.Context) error {
	if e.currentScenario == nil {
		return nil
	}

	e.state = StateCleaning
	fmt.Printf("Cleaning up scenario: %s\n", e.currentScenario.GetMetadata().Name)

	if err := e.currentScenario.Cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup scenario: %w", err)
	}

	e.currentScenario = nil
	e.state = StateIdle

	return nil
}

// GetState returns the current state.
func (e *Engine) GetState() State {
	return e.state
}

// GetCurrentScenario returns the currently running scenario.
func (e *Engine) GetCurrentScenario() scenario.Scenario {
	return e.currentScenario
}

// GetElapsedTime returns how long the current scenario has been running.
func (e *Engine) GetElapsedTime() time.Duration {
	if e.state == StateIdle || e.startTime.IsZero() {
		return 0
	}
	return time.Since(e.startTime)
}
