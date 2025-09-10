package mocks

import (
	"time"

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

	// Pane operations
	m.On("ResizePanesEvenly", mock.Anything, mock.Anything).Maybe().Return(nil)

	// Pane operations
	m.On("ResizePanesEvenly", mock.Anything, mock.Anything).Maybe().Return(nil)

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

// PaneManager methods

// CreatePane mocks the CreatePane method
func (m *MockTmuxManager) CreatePane(sessionName, windowName string, opts tmux.PaneOptions) (*tmux.PaneInfo, error) {
	args := m.Called(sessionName, windowName, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tmux.PaneInfo), args.Error(1)
}

// SelectPane mocks the SelectPane method
func (m *MockTmuxManager) SelectPane(sessionName, windowName string, paneIndex int) error {
	args := m.Called(sessionName, windowName, paneIndex)
	return args.Error(0)
}

// SetPaneTitle mocks the SetPaneTitle method
func (m *MockTmuxManager) SetPaneTitle(sessionName, windowName string, paneIndex int, title string) error {
	args := m.Called(sessionName, windowName, paneIndex, title)
	return args.Error(0)
}

// ListPanes mocks the ListPanes method
func (m *MockTmuxManager) ListPanes(sessionName, windowName string) ([]*tmux.PaneInfo, error) {
	args := m.Called(sessionName, windowName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*tmux.PaneInfo), args.Error(1)
}

// GetPaneByTitle mocks the GetPaneByTitle method
func (m *MockTmuxManager) GetPaneByTitle(sessionName, windowName string, title string) (*tmux.PaneInfo, error) {
	args := m.Called(sessionName, windowName, title)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tmux.PaneInfo), args.Error(1)
}

// ResizePanesEvenly mocks the ResizePanesEvenly method
func (m *MockTmuxManager) ResizePanesEvenly(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// ResizePanesEvenlyWithRetry mocks the ResizePanesEvenlyWithRetry method
func (m *MockTmuxManager) ResizePanesEvenlyWithRetry(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

// GetWindowSize mocks the GetWindowSize method
func (m *MockTmuxManager) GetWindowSize(sessionName, windowName string) (width, height int, err error) {
	args := m.Called(sessionName, windowName)
	return args.Int(0), args.Int(1), args.Error(2)
}

// KillPane mocks the KillPane method
func (m *MockTmuxManager) KillPane(sessionName, windowName string, paneIndex int) error {
	args := m.Called(sessionName, windowName, paneIndex)
	return args.Error(0)
}

// GetPaneBaseIndex mocks the GetPaneBaseIndex method
func (m *MockTmuxManager) GetPaneBaseIndex() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// CreateWindowForIssueWithNewWindowDetection mocks the CreateWindowForIssueWithNewWindowDetection method
func (m *MockTmuxManager) CreateWindowForIssueWithNewWindowDetection(sessionName string, issueNumber int) (string, bool, error) {
	args := m.Called(sessionName, issueNumber)
	return args.String(0), args.Bool(1), args.Error(2)
}

// DiagnosticManager methods

// DiagnoseSession mocks the DiagnoseSession method
func (m *MockTmuxManager) DiagnoseSession(sessionName string) (*tmux.SessionDiagnostics, error) {
	args := m.Called(sessionName)
	return args.Get(0).(*tmux.SessionDiagnostics), args.Error(1)
}

// DiagnoseWindow mocks the DiagnoseWindow method
func (m *MockTmuxManager) DiagnoseWindow(sessionName, windowName string) (*tmux.WindowDiagnostics, error) {
	args := m.Called(sessionName, windowName)
	return args.Get(0).(*tmux.WindowDiagnostics), args.Error(1)
}

// ListSessionDiagnostics mocks the ListSessionDiagnostics method
func (m *MockTmuxManager) ListSessionDiagnostics(prefix string) ([]*tmux.SessionDiagnostics, error) {
	args := m.Called(prefix)
	return args.Get(0).([]*tmux.SessionDiagnostics), args.Error(1)
}

// ListWindowDiagnostics mocks the ListWindowDiagnostics method
func (m *MockTmuxManager) ListWindowDiagnostics(sessionName string) ([]*tmux.WindowDiagnostics, error) {
	args := m.Called(sessionName)
	return args.Get(0).([]*tmux.WindowDiagnostics), args.Error(1)
}

// WithDiagnosticDefaults sets up default behaviors for diagnostic methods
func (m *MockTmuxManager) WithDiagnosticDefaults() *MockTmuxManager {
	// Session診断のデフォルト
	m.On("DiagnoseSession", mock.Anything).Maybe().Return(&tmux.SessionDiagnostics{
		Name:      "test-session",
		Windows:   1,
		Attached:  false,
		Created:   "1641641600",
		Errors:    []string{},
		Metadata:  map[string]string{"exists": "true", "mock": "true"},
		Timestamp: time.Now(),
	}, nil)

	// Window診断のデフォルト
	m.On("DiagnoseWindow", mock.Anything, mock.Anything).Maybe().Return(&tmux.WindowDiagnostics{
		Name:        "test-window",
		SessionName: "test-session",
		Index:       0,
		Exists:      true,
		Active:      false,
		Panes:       1,
		IssueNumber: 0,
		Phase:       "",
		Errors:      []string{},
		Metadata:    map[string]string{"exists": "true", "mock": "true"},
		Timestamp:   time.Now(),
	}, nil)

	// セッション一覧診断のデフォルト
	m.On("ListSessionDiagnostics", mock.Anything).Maybe().Return([]*tmux.SessionDiagnostics{
		{
			Name:      "test-session",
			Windows:   1,
			Attached:  false,
			Created:   "1641641600",
			Errors:    []string{},
			Metadata:  map[string]string{"exists": "true", "mock": "true"},
			Timestamp: time.Now(),
		},
	}, nil)

	// ウィンドウ一覧診断のデフォルト
	m.On("ListWindowDiagnostics", mock.Anything).Maybe().Return([]*tmux.WindowDiagnostics{
		{
			Name:        "test-window",
			SessionName: "test-session",
			Index:       0,
			Exists:      true,
			Active:      false,
			Panes:       1,
			IssueNumber: 0,
			Phase:       "",
			Errors:      []string{},
			Metadata:    map[string]string{"exists": "true", "mock": "true"},
			Timestamp:   time.Now(),
		},
	}, nil)

	return m
}

// Ensure MockTmuxManager implements tmux.Manager interface
var _ tmux.Manager = (*MockTmuxManager)(nil)
