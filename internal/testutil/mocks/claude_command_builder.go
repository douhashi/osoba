package mocks

import (
	"github.com/stretchr/testify/mock"
)

// MockClaudeCommandBuilder is a mock implementation of ClaudeCommandBuilder interface
type MockClaudeCommandBuilder struct {
	mock.Mock
}

// NewMockClaudeCommandBuilder creates a new instance of MockClaudeCommandBuilder
func NewMockClaudeCommandBuilder() *MockClaudeCommandBuilder {
	return &MockClaudeCommandBuilder{}
}

// BuildCommand mocks the BuildCommand method
func (m *MockClaudeCommandBuilder) BuildCommand(promptPath string, outputPath string, workdir string, vars interface{}) string {
	args := m.Called(promptPath, outputPath, workdir, vars)
	return args.String(0)
}
