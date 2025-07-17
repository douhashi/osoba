package mocks

import (
	"fmt"

	"github.com/stretchr/testify/mock"
)

// MockCommandExecutor is a mock implementation of tmux.CommandExecutor interface
type MockCommandExecutor struct {
	mock.Mock
}

// NewMockCommandExecutor creates a new instance of MockCommandExecutor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockCommandExecutor) WithDefaultBehavior() *MockCommandExecutor {
	// tmux list-sessions のデフォルト動作
	m.On("Execute", "tmux", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "list-sessions"
	})).Maybe().Return("", nil)

	// tmux new-session のデフォルト動作
	m.On("Execute", "tmux", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "new-session"
	})).Maybe().Return("", nil)

	// tmux new-window のデフォルト動作
	m.On("Execute", "tmux", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "new-window"
	})).Maybe().Return("", nil)

	// tmux kill-session のデフォルト動作
	m.On("Execute", "tmux", mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "kill-session"
	})).Maybe().Return("", nil)

	return m
}

// Execute mocks the Execute method
func (m *MockCommandExecutor) Execute(cmd string, args ...string) (string, error) {
	// Convert variadic args to []string for consistent matching
	argsList := []string{}
	if len(args) > 0 {
		argsList = args
	}
	ret := m.Called(cmd, argsList)
	return ret.String(0), ret.Error(1)
}

// WithTmuxSessions sets up mock to return specific tmux sessions
func (m *MockCommandExecutor) WithTmuxSessions(sessions ...string) *MockCommandExecutor {
	sessionsOutput := ""
	for _, session := range sessions {
		if sessionsOutput != "" {
			sessionsOutput += "\n"
		}
		sessionsOutput += session + ": 1 windows (created Wed Jan 1 00:00:00 2025)"
	}

	m.On("Execute", "tmux", []string{"list-sessions", "-F", "#{session_name}"}).
		Return(sessionsOutput, nil)

	return m
}

// WithTmuxWindows sets up mock to return specific tmux windows for a session
func (m *MockCommandExecutor) WithTmuxWindows(sessionName string, windows ...string) *MockCommandExecutor {
	windowsOutput := ""
	for i, window := range windows {
		if windowsOutput != "" {
			windowsOutput += "\n"
		}
		windowsOutput += fmt.Sprintf("%d: %s", i, window)
	}

	m.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}: #{window_name}"}).
		Return(windowsOutput, nil)

	return m
}

// MockExitError is a mock implementation of exec.ExitError
type MockExitError struct {
	ExitCode int
}

// Error implements the error interface
func (e *MockExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.ExitCode)
}

// WithCommandError sets up mock to return an error for specific command
func (m *MockCommandExecutor) WithCommandError(cmd string, args []string, err error) *MockCommandExecutor {
	m.On("Execute", cmd, args).Return("", err)
	return m
}

// WithCommandOutput sets up mock to return specific output for a command
func (m *MockCommandExecutor) WithCommandOutput(cmd string, args []string, output string) *MockCommandExecutor {
	m.On("Execute", cmd, args).Return(output, nil)
	return m
}
