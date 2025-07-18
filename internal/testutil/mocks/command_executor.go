package mocks

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/stretchr/testify/mock"
)

// MockTmuxCommandExecutor is a mock implementation of tmux.CommandExecutor interface
type MockTmuxCommandExecutor struct {
	mock.Mock
}

// NewMockTmuxCommandExecutor creates a new instance of MockTmuxCommandExecutor
func NewMockTmuxCommandExecutor() *MockTmuxCommandExecutor {
	return &MockTmuxCommandExecutor{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockTmuxCommandExecutor) WithDefaultBehavior() *MockTmuxCommandExecutor {
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
func (m *MockTmuxCommandExecutor) Execute(cmd string, args ...string) (string, error) {
	// Convert variadic args to []string for consistent matching
	argsList := []string{}
	if len(args) > 0 {
		argsList = args
	}
	ret := m.Called(cmd, argsList)
	return ret.String(0), ret.Error(1)
}

// WithTmuxSessions sets up mock to return specific tmux sessions
func (m *MockTmuxCommandExecutor) WithTmuxSessions(sessions ...string) *MockTmuxCommandExecutor {
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
func (m *MockTmuxCommandExecutor) WithTmuxWindows(sessionName string, windows ...string) *MockTmuxCommandExecutor {
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
func (m *MockTmuxCommandExecutor) WithCommandError(cmd string, args []string, err error) *MockTmuxCommandExecutor {
	m.On("Execute", cmd, args).Return("", err)
	return m
}

// WithCommandOutput sets up mock to return specific output for a command
func (m *MockTmuxCommandExecutor) WithCommandOutput(cmd string, args []string, output string) *MockTmuxCommandExecutor {
	m.On("Execute", cmd, args).Return(output, nil)
	return m
}

// MockSystemCommandExecutor is a mock implementation of system command execution interface
type MockSystemCommandExecutor struct {
	mock.Mock
}

// NewMockSystemCommandExecutor creates a new instance of MockSystemCommandExecutor
func NewMockSystemCommandExecutor() *MockSystemCommandExecutor {
	return &MockSystemCommandExecutor{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockSystemCommandExecutor) WithDefaultBehavior() *MockSystemCommandExecutor {
	// LookPath returns a dummy path by default
	m.On("LookPath", mock.Anything).Maybe().Return("/usr/bin/cmd", nil)

	// CommandContext returns a dummy command
	m.On("CommandContext", mock.Anything, mock.Anything, mock.Anything).Maybe().
		Return(&exec.Cmd{Path: "/usr/bin/cmd", Args: []string{"cmd"}})

	return m
}

// CommandContext mocks the CommandContext method
func (m *MockSystemCommandExecutor) CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	args := m.Called(ctx, name, arg)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*exec.Cmd)
}

// LookPath mocks the LookPath method
func (m *MockSystemCommandExecutor) LookPath(file string) (string, error) {
	args := m.Called(file)
	return args.String(0), args.Error(1)
}
