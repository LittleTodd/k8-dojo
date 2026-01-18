// Package scenario defines the interface and types for troubleshooting scenarios.
package scenario

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

// Difficulty represents the difficulty level of a scenario.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "Easy"
	DifficultyMedium Difficulty = "Medium"
	DifficultyHard   Difficulty = "Hard"
)

// Status represents the current status of a scenario.
type Status string

const (
	StatusPending  Status = "pending"  // Not yet started
	StatusRunning  Status = "running"  // Scenario is active, waiting for solution
	StatusSolved   Status = "solved"   // User solved the problem
	StatusFailed   Status = "failed"   // User gave up or timeout
	StatusCleaning Status = "cleaning" // Cleaning up resources
)

// Metadata contains descriptive information about a scenario.
type Metadata struct {
	ID          string
	Name        string
	Description string
	Difficulty  Difficulty
	Category    string
	Hints       []string
	TimeLimit   time.Duration // 0 means no limit
}

// Result contains the outcome of a validation check.
type Result struct {
	Solved  bool
	Message string
}

// Scenario defines the interface that all troubleshooting scenarios must implement.
type Scenario interface {
	// GetMetadata returns the scenario's metadata.
	GetMetadata() Metadata

	// Setup injects the faulty resources into the cluster.
	// Returns an error if setup fails.
	Setup(ctx context.Context) error

	// Validate checks if the user has solved the problem.
	// Returns a Result indicating whether the scenario is solved.
	Validate(ctx context.Context) Result

	// Cleanup removes all resources created by this scenario.
	Cleanup(ctx context.Context) error

	// GetNamespace returns the namespace used by this scenario.
	GetNamespace() string
}

// BaseScenario provides common functionality for scenarios.
type BaseScenario struct {
	Namespace string
}

// GetNamespace returns the namespace used by this scenario.
func (b *BaseScenario) GetNamespace() string {
	return b.Namespace
}

func mustParse(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}
