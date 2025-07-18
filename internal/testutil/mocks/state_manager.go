package mocks

import (
	"github.com/douhashi/osoba/internal/types"
	"github.com/stretchr/testify/mock"
)

// MockStateManager is a mock implementation of StateManager interface
type MockStateManager struct {
	mock.Mock
}

// NewMockStateManager creates a new instance of MockStateManager
func NewMockStateManager() *MockStateManager {
	return &MockStateManager{}
}

// SetState mocks the SetState method
func (m *MockStateManager) SetState(issueNumber int64, phase types.IssuePhase, status types.IssueStatus) {
	m.Called(issueNumber, phase, status)
}

// GetState mocks the GetState method
func (m *MockStateManager) GetState(issueNumber int64, phase types.IssuePhase) types.IssueStatus {
	args := m.Called(issueNumber, phase)
	return args.Get(0).(types.IssueStatus)
}

// HasBeenProcessed mocks the HasBeenProcessed method
func (m *MockStateManager) HasBeenProcessed(issueNumber int64, phase types.IssuePhase) bool {
	args := m.Called(issueNumber, phase)
	return args.Bool(0)
}

// IsProcessing mocks the IsProcessing method
func (m *MockStateManager) IsProcessing(issueNumber int64) bool {
	args := m.Called(issueNumber)
	return args.Bool(0)
}

// MarkAsCompleted mocks the MarkAsCompleted method
func (m *MockStateManager) MarkAsCompleted(issueNumber int64, phase types.IssuePhase) {
	m.Called(issueNumber, phase)
}

// MarkAsFailed mocks the MarkAsFailed method
func (m *MockStateManager) MarkAsFailed(issueNumber int64, phase types.IssuePhase) {
	m.Called(issueNumber, phase)
}

// Clear mocks the Clear method
func (m *MockStateManager) Clear(issueNumber int64) {
	m.Called(issueNumber)
}

// GetAllStates mocks the GetAllStates method
func (m *MockStateManager) GetAllStates() map[int64]map[types.IssuePhase]types.IssueStatus {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[int64]map[types.IssuePhase]types.IssueStatus)
}
