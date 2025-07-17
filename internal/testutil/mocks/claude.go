package mocks

import (
	"context"
	"os/exec"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/stretchr/testify/mock"
)

// MockClaudeExecutor is a mock implementation of claude.ClaudeExecutor interface
type MockClaudeExecutor struct {
	mock.Mock
}

// NewMockClaudeExecutor creates a new instance of MockClaudeExecutor
func NewMockClaudeExecutor() *MockClaudeExecutor {
	return &MockClaudeExecutor{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockClaudeExecutor) WithDefaultBehavior() *MockClaudeExecutor {
	// CheckClaudeExists returns no error by default
	m.On("CheckClaudeExists").Maybe().Return(nil)

	// BuildCommand returns a dummy command
	m.On("BuildCommand", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().
		Return(&exec.Cmd{Path: "claude", Args: []string{"claude"}})

	// ExecuteInWorktree succeeds by default
	m.On("ExecuteInWorktree", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	// ExecuteInTmux succeeds by default
	m.On("ExecuteInTmux", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	return m
}

// CheckClaudeExists mocks the CheckClaudeExists method
func (m *MockClaudeExecutor) CheckClaudeExists() error {
	args := m.Called()
	return args.Error(0)
}

// BuildCommand mocks the BuildCommand method
func (m *MockClaudeExecutor) BuildCommand(ctx context.Context, args []string, prompt string, workdir string) *exec.Cmd {
	mockArgs := m.Called(ctx, args, prompt, workdir)
	if mockArgs.Get(0) == nil {
		return nil
	}
	return mockArgs.Get(0).(*exec.Cmd)
}

// ExecuteInWorktree mocks the ExecuteInWorktree method
func (m *MockClaudeExecutor) ExecuteInWorktree(ctx context.Context, config *claude.PhaseConfig, vars *claude.TemplateVariables, workdir string) error {
	args := m.Called(ctx, config, vars, workdir)
	return args.Error(0)
}

// ExecuteInTmux mocks the ExecuteInTmux method
func (m *MockClaudeExecutor) ExecuteInTmux(ctx context.Context, config *claude.PhaseConfig, vars *claude.TemplateVariables, sessionName, windowName, workdir string) error {
	args := m.Called(ctx, config, vars, sessionName, windowName, workdir)
	return args.Error(0)
}

// Ensure MockClaudeExecutor implements claude.ClaudeExecutor interface
var _ claude.ClaudeExecutor = (*MockClaudeExecutor)(nil)
