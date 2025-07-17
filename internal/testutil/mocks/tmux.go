package mocks

import (
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/mock"
)

// MockTmuxManager is a mock implementation of tmux.Manager interface
type MockTmuxManager struct {
	mock.Mock
}

// NewMockTmuxManager creates a new instance of MockTmuxManager
func NewMockTmuxManager() *MockTmuxManager {
	return &MockTmuxManager{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockTmuxManager) WithDefaultBehavior() *MockTmuxManager {
	// Session operations
	m.On("CheckTmuxInstalled").Maybe().Return(nil)
	m.On("SessionExists", mock.Anything).Maybe().Return(true, nil)
	m.On("CreateSession", mock.Anything).Maybe().Return(nil)
	m.On("EnsureSession", mock.Anything).Maybe().Return(nil)
	m.On("ListSessions", mock.Anything).Maybe().Return([]string{}, nil)

	// Window operations
	m.On("CreateWindow", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("SwitchToWindow", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("WindowExists", mock.Anything, mock.Anything).Maybe().Return(true, nil)
	m.On("KillWindow", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("CreateOrReplaceWindow", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("ListWindows", mock.Anything).Maybe().Return([]string{}, nil)

	// Command execution
	m.On("SendKeys", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("ClearWindow", mock.Anything, mock.Anything).Maybe().Return(nil)
	m.On("RunInWindow", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	// Issue window operations - GetIssueWindowのデフォルト実装
	m.On("GetIssueWindow", mock.AnythingOfType("int")).Maybe().Return("issue-123")
	m.On("MatchIssueWindow", mock.Anything).Maybe().Return(false)
	m.On("FindIssueWindow", mock.Anything).Maybe().Return(0, false)

	return m
}

// SessionManager methods

// CheckTmuxInstalled mocks the CheckTmuxInstalled method
func (m *MockTmuxManager) CheckTmuxInstalled() error {
	args := m.Called()
	return args.Error(0)
}

// SessionExists mocks the SessionExists method
func (m *MockTmuxManager) SessionExists(sessionName string) (bool, error) {
	args := m.Called(sessionName)
	return args.Bool(0), args.Error(1)
}

// CreateSession mocks the CreateSession method
func (m *MockTmuxManager) CreateSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

// EnsureSession mocks the EnsureSession method
func (m *MockTmuxManager) EnsureSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

// ListSessions mocks the ListSessions method
func (m *MockTmuxManager) ListSessions(prefix string) ([]string, error) {
	args := m.Called(prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// WindowManager methods

// CreateWindow mocks the CreateWindow method
func (m *MockTmuxManager) CreateWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// SwitchToWindow mocks the SwitchToWindow method
func (m *MockTmuxManager) SwitchToWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// WindowExists mocks the WindowExists method
func (m *MockTmuxManager) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

// KillWindow mocks the KillWindow method
func (m *MockTmuxManager) KillWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// CreateOrReplaceWindow mocks the CreateOrReplaceWindow method
func (m *MockTmuxManager) CreateOrReplaceWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// ListWindows mocks the ListWindows method
func (m *MockTmuxManager) ListWindows(sessionName string) ([]string, error) {
	args := m.Called(sessionName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// SendKeys mocks the SendKeys method
func (m *MockTmuxManager) SendKeys(sessionName, windowName, keys string) error {
	args := m.Called(sessionName, windowName, keys)
	return args.Error(0)
}

// ClearWindow mocks the ClearWindow method
func (m *MockTmuxManager) ClearWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// RunInWindow mocks the RunInWindow method
func (m *MockTmuxManager) RunInWindow(sessionName, windowName, command string) error {
	args := m.Called(sessionName, windowName, command)
	return args.Error(0)
}

// GetIssueWindow mocks the GetIssueWindow method
func (m *MockTmuxManager) GetIssueWindow(issueNumber int) string {
	args := m.Called(issueNumber)
	return args.String(0)
}

// MatchIssueWindow mocks the MatchIssueWindow method
func (m *MockTmuxManager) MatchIssueWindow(windowName string) bool {
	args := m.Called(windowName)
	return args.Bool(0)
}

// FindIssueWindow mocks the FindIssueWindow method
func (m *MockTmuxManager) FindIssueWindow(windowName string) (int, bool) {
	args := m.Called(windowName)
	return args.Int(0), args.Bool(1)
}

// Ensure MockTmuxManager implements tmux.Manager interface
var _ tmux.Manager = (*MockTmuxManager)(nil)
