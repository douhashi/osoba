package helpers

import (
	"testing"
)

// CommandMocker provides utilities for mocking command execution functions.
// This is useful for replacing global function variables in tests.
type CommandMocker struct {
	t        *testing.T
	original interface{}
	target   interface{}
}

// NewCommandMocker creates a new CommandMocker.
// The target parameter should be a pointer to the function variable to be mocked.
func NewCommandMocker(t *testing.T, target interface{}) *CommandMocker {
	t.Helper()
	return &CommandMocker{
		t:      t,
		target: target,
	}
}

// Mock replaces the target function with the mock implementation.
// It returns a cleanup function that should be called to restore the original.
//
// Example:
//
//	mockFunc := func() error { return nil }
//	mocker := NewCommandMocker(t, &originalFunc)
//	cleanup := mocker.Mock(mockFunc)
//	defer cleanup()
func (m *CommandMocker) Mock(mockImpl interface{}) func() {
	m.t.Helper()

	// This is a simplified version that would need reflection
	// to work properly with any function type.
	// For now, it serves as a placeholder for the concept.

	return func() {
		// Restore original
	}
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Output string
	Error  error
}

// CommandRecorder records command executions for verification in tests.
type CommandRecorder struct {
	t        *testing.T
	commands []CommandCall
}

// CommandCall represents a single command execution.
type CommandCall struct {
	Command string
	Args    []string
	Result  CommandResult
}

// NewCommandRecorder creates a new CommandRecorder.
func NewCommandRecorder(t *testing.T) *CommandRecorder {
	t.Helper()
	return &CommandRecorder{
		t:        t,
		commands: make([]CommandCall, 0),
	}
}

// Record records a command execution.
func (r *CommandRecorder) Record(command string, args []string, result CommandResult) {
	r.commands = append(r.commands, CommandCall{
		Command: command,
		Args:    args,
		Result:  result,
	})
}

// GetCalls returns all recorded command calls.
func (r *CommandRecorder) GetCalls() []CommandCall {
	return r.commands
}

// GetCallsForCommand returns all recorded calls for a specific command.
func (r *CommandRecorder) GetCallsForCommand(command string) []CommandCall {
	var calls []CommandCall
	for _, call := range r.commands {
		if call.Command == command {
			calls = append(calls, call)
		}
	}
	return calls
}

// AssertCalled asserts that a command was called at least once.
func (r *CommandRecorder) AssertCalled(command string) {
	r.t.Helper()
	for _, call := range r.commands {
		if call.Command == command {
			return
		}
	}
	r.t.Errorf("expected command %q to be called, but it was not", command)
}

// AssertNotCalled asserts that a command was not called.
func (r *CommandRecorder) AssertNotCalled(command string) {
	r.t.Helper()
	for _, call := range r.commands {
		if call.Command == command {
			r.t.Errorf("expected command %q not to be called, but it was", command)
			return
		}
	}
}

// AssertCallCount asserts that a command was called a specific number of times.
func (r *CommandRecorder) AssertCallCount(command string, expectedCount int) {
	r.t.Helper()
	count := 0
	for _, call := range r.commands {
		if call.Command == command {
			count++
		}
	}
	if count != expectedCount {
		r.t.Errorf("expected command %q to be called %d times, but it was called %d times", command, expectedCount, count)
	}
}

// Reset clears all recorded commands.
func (r *CommandRecorder) Reset() {
	r.commands = r.commands[:0]
}
