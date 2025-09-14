package executor

import (
	"context"
	"testing"

	"github.com/seqr-cli/seqr/internal/config"
)

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor(true)
	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	status := executor.GetStatus()
	if status.State != StateReady {
		t.Errorf("Expected initial state to be Ready, got %v", status.State)
	}
}

func TestExecutor_Execute_EmptyCommands(t *testing.T) {
	executor := NewExecutor(false)
	ctx := context.Background()

	cfg := &config.Config{
		Version:  "1.0",
		Commands: []config.Command{},
	}

	err := executor.Execute(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error for empty commands, got nil")
	}
}

func TestExecutor_Execute_Success(t *testing.T) {
	executor := NewExecutor(false)
	ctx := context.Background()

	cfg := &config.Config{
		Version: "1.0",
		Commands: []config.Command{
			{
				Name:    "test",
				Command: "echo",
				Args:    []string{"hello"},
				Mode:    config.ModeOnce,
			},
		},
	}

	err := executor.Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Expected successful execution, got error: %v", err)
	}

	status := executor.GetStatus()
	if status.State != StateSuccess {
		t.Errorf("Expected final state to be Success, got %v", status.State)
	}
}
