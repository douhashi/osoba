package mocks_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockTmuxManager_SessionOperations(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockTmuxManager)
		testFunc  func(*mocks.MockTmuxManager) error
		wantErr   bool
	}{
		{
			name: "check tmux installed",
			setupMock: func(m *mocks.MockTmuxManager) {
				m.On("CheckTmuxInstalled").Return(nil)
			},
			testFunc: func(m *mocks.MockTmuxManager) error {
				return m.CheckTmuxInstalled()
			},
			wantErr: false,
		},
		{
			name: "session exists - true",
			setupMock: func(m *mocks.MockTmuxManager) {
				m.On("SessionExists", "test-session").Return(true, nil)
			},
			testFunc: func(m *mocks.MockTmuxManager) error {
				exists, err := m.SessionExists("test-session")
				if err != nil {
					return err
				}
				if !exists {
					return errors.New("expected session to exist")
				}
				return nil
			},
			wantErr: false,
		},
		{
			name: "create session",
			setupMock: func(m *mocks.MockTmuxManager) {
				m.On("CreateSession", "new-session").Return(nil)
			},
			testFunc: func(m *mocks.MockTmuxManager) error {
				return m.CreateSession("new-session")
			},
			wantErr: false,
		},
		{
			name: "ensure session - creates new",
			setupMock: func(m *mocks.MockTmuxManager) {
				m.On("EnsureSession", "test-session").Return(nil)
			},
			testFunc: func(m *mocks.MockTmuxManager) error {
				return m.EnsureSession("test-session")
			},
			wantErr: false,
		},
		{
			name: "list sessions",
			setupMock: func(m *mocks.MockTmuxManager) {
				m.On("ListSessions", "osoba-").Return([]string{"osoba-repo1", "osoba-repo2"}, nil)
			},
			testFunc: func(m *mocks.MockTmuxManager) error {
				sessions, err := m.ListSessions("osoba-")
				if err != nil {
					return err
				}
				if len(sessions) != 2 {
					return errors.New("expected 2 sessions")
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTmux := mocks.NewMockTmuxManager()
			tt.setupMock(mockTmux)

			err := tt.testFunc(mockTmux)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockTmux.AssertExpectations(t)
		})
	}
}

func TestMockTmuxManager_WindowOperations(t *testing.T) {
	mockTmux := mocks.NewMockTmuxManager()

	// CreateWindow
	mockTmux.On("CreateWindow", "session", "window").Return(nil)
	err := mockTmux.CreateWindow("session", "window")
	assert.NoError(t, err)

	// SwitchToWindow
	mockTmux.On("SwitchToWindow", "session", "window").Return(nil)
	err = mockTmux.SwitchToWindow("session", "window")
	assert.NoError(t, err)

	// WindowExists
	mockTmux.On("WindowExists", "session", "window").Return(true, nil)
	exists, err := mockTmux.WindowExists("session", "window")
	assert.NoError(t, err)
	assert.True(t, exists)

	// KillWindow
	mockTmux.On("KillWindow", "session", "window").Return(nil)
	err = mockTmux.KillWindow("session", "window")
	assert.NoError(t, err)

	// CreateOrReplaceWindow
	mockTmux.On("CreateOrReplaceWindow", "session", "window").Return(nil)
	err = mockTmux.CreateOrReplaceWindow("session", "window")
	assert.NoError(t, err)

	// ListWindows
	mockTmux.On("ListWindows", "session").Return([]string{"window1", "window2"}, nil)
	windows, err := mockTmux.ListWindows("session")
	assert.NoError(t, err)
	assert.Len(t, windows, 2)

	mockTmux.AssertExpectations(t)
}

func TestMockTmuxManager_CommandExecution(t *testing.T) {
	mockTmux := mocks.NewMockTmuxManager()

	// SendKeys
	mockTmux.On("SendKeys", "session", "window", "echo 'hello'").Return(nil)
	err := mockTmux.SendKeys("session", "window", "echo 'hello'")
	assert.NoError(t, err)

	// ClearWindow
	mockTmux.On("ClearWindow", "session", "window").Return(nil)
	err = mockTmux.ClearWindow("session", "window")
	assert.NoError(t, err)

	// RunInWindow
	mockTmux.On("RunInWindow", "session", "window", "ls -la").Return(nil)
	err = mockTmux.RunInWindow("session", "window", "ls -la")
	assert.NoError(t, err)

	mockTmux.AssertExpectations(t)
}

func TestMockTmuxManager_IssueWindowFunctions(t *testing.T) {
	mockTmux := mocks.NewMockTmuxManager()

	// GetIssueWindow
	mockTmux.On("GetIssueWindow", 123).Return("issue-123")
	windowName := mockTmux.GetIssueWindow(123)
	assert.Equal(t, "issue-123", windowName)

	// MatchIssueWindow
	mockTmux.On("MatchIssueWindow", "issue-123").Return(true)
	mockTmux.On("MatchIssueWindow", "not-issue").Return(false)
	assert.True(t, mockTmux.MatchIssueWindow("issue-123"))
	assert.False(t, mockTmux.MatchIssueWindow("not-issue"))

	// FindIssueWindow
	mockTmux.On("FindIssueWindow", "issue-123").Return(123, true)
	mockTmux.On("FindIssueWindow", "not-issue").Return(0, false)

	issueNum, ok := mockTmux.FindIssueWindow("issue-123")
	assert.True(t, ok)
	assert.Equal(t, 123, issueNum)

	issueNum, ok = mockTmux.FindIssueWindow("not-issue")
	assert.False(t, ok)
	assert.Equal(t, 0, issueNum)

	mockTmux.AssertExpectations(t)
}

func TestMockTmuxManager_WithDefaultBehavior(t *testing.T) {
	mockTmux := mocks.NewMockTmuxManager().WithDefaultBehavior()

	// Session operations
	err := mockTmux.CheckTmuxInstalled()
	assert.NoError(t, err)

	exists, err := mockTmux.SessionExists("any-session")
	assert.NoError(t, err)
	assert.True(t, exists)

	err = mockTmux.CreateSession("new-session")
	assert.NoError(t, err)

	// Window operations
	err = mockTmux.CreateWindow("session", "window")
	assert.NoError(t, err)

	exists, err = mockTmux.WindowExists("session", "window")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Issue window operations
	windowName := mockTmux.GetIssueWindow(123)
	assert.Equal(t, "issue-123", windowName)
}

func TestMockTmuxManager_ComplexScenario(t *testing.T) {
	mockTmux := mocks.NewMockTmuxManager()

	// 複雑なシナリオのセットアップ
	mockTmux.On("EnsureSession", "osoba-repo").Return(nil)
	mockTmux.On("CreateOrReplaceWindow", "osoba-repo", "issue-123").Return(nil)
	mockTmux.On("ClearWindow", "osoba-repo", "issue-123").Return(nil)
	mockTmux.On("RunInWindow", "osoba-repo", "issue-123", mock.MatchedBy(func(cmd string) bool {
		return len(cmd) > 0
	})).Return(nil)
	mockTmux.On("SwitchToWindow", "osoba-repo", "issue-123").Return(nil)

	// 実行
	err := mockTmux.EnsureSession("osoba-repo")
	assert.NoError(t, err)

	err = mockTmux.CreateOrReplaceWindow("osoba-repo", "issue-123")
	assert.NoError(t, err)

	err = mockTmux.ClearWindow("osoba-repo", "issue-123")
	assert.NoError(t, err)

	err = mockTmux.RunInWindow("osoba-repo", "issue-123", "cd /path/to/repo")
	assert.NoError(t, err)

	err = mockTmux.SwitchToWindow("osoba-repo", "issue-123")
	assert.NoError(t, err)

	mockTmux.AssertExpectations(t)
}
