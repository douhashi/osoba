package mocks

import (
	"context"

	"github.com/douhashi/osoba/internal/gh"
	"github.com/stretchr/testify/mock"
)

// MockGHExecutor is a mock implementation of gh.Executor interface
type MockGHExecutor struct {
	mock.Mock
}

// NewMockGHExecutor creates a new instance of MockGHExecutor
func NewMockGHExecutor() *MockGHExecutor {
	return &MockGHExecutor{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockGHExecutor) WithDefaultBehavior() *MockGHExecutor {
	// API呼び出しのデフォルト動作
	m.On("Execute", mock.Anything, mock.MatchedBy(func(args []string) bool {
		return len(args) > 0 && args[0] == "api"
	})).Maybe().Return([]byte("{}"), nil)

	// issue list のデフォルト動作
	m.On("Execute", mock.Anything, mock.MatchedBy(func(args []string) bool {
		return len(args) > 1 && args[0] == "issue" && args[1] == "list"
	})).Maybe().Return([]byte("[]"), nil)

	// issue comment のデフォルト動作
	m.On("Execute", mock.Anything, mock.MatchedBy(func(args []string) bool {
		return len(args) > 1 && args[0] == "issue" && args[1] == "comment"
	})).Maybe().Return([]byte(""), nil)

	return m
}

// Execute mocks the Execute method
func (m *MockGHExecutor) Execute(ctx context.Context, args []string) ([]byte, error) {
	ret := m.Called(ctx, args)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).([]byte), ret.Error(1)
}

// Ensure MockGHExecutor implements gh.Executor interface
var _ gh.Executor = (*MockGHExecutor)(nil)
