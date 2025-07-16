package tmux_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockManager はManagerインターフェースのモック実装（既存のManagerテスト用）
type MockManager struct {
	mock.Mock
}

// SessionManager methods
func (m *MockManager) CheckTmuxInstalled() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockManager) SessionExists(sessionName string) (bool, error) {
	args := m.Called(sessionName)
	return args.Bool(0), args.Error(1)
}

func (m *MockManager) CreateSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockManager) EnsureSession(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockManager) ListSessions(prefix string) ([]string, error) {
	args := m.Called(prefix)
	if result := args.Get(0); result == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// WindowManager methods
func (m *MockManager) CreateWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockManager) SwitchToWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockManager) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

func (m *MockManager) KillWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockManager) CreateOrReplaceWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockManager) ListWindows(sessionName string) ([]string, error) {
	args := m.Called(sessionName)
	if result := args.Get(0); result == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockManager) SendKeys(sessionName, windowName, keys string) error {
	args := m.Called(sessionName, windowName, keys)
	return args.Error(0)
}

func (m *MockManager) ClearWindow(sessionName, windowName string) error {
	args := m.Called(sessionName, windowName)
	return args.Error(0)
}

func (m *MockManager) RunInWindow(sessionName, windowName, command string) error {
	args := m.Called(sessionName, windowName, command)
	return args.Error(0)
}

func (m *MockManager) GetIssueWindow(issueNumber int) string {
	args := m.Called(issueNumber)
	return args.String(0)
}

func (m *MockManager) MatchIssueWindow(windowName string) bool {
	args := m.Called(windowName)
	return args.Bool(0)
}

func (m *MockManager) FindIssueWindow(windowName string) (int, bool) {
	args := m.Called(windowName)
	return args.Int(0), args.Bool(1)
}

func TestCheckTmuxInstalledWithMock(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockManager)
		wantErr bool
		errMsg  string
	}{
		{
			name: "tmuxがインストールされている場合",
			setup: func(m *MockManager) {
				m.On("CheckTmuxInstalled").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "tmuxがインストールされていない場合",
			setup: func(m *MockManager) {
				m.On("CheckTmuxInstalled").Return(tmux.ErrTmuxNotInstalled)
			},
			wantErr: true,
			errMsg:  "tmuxがインストールされていません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			oldManager := tmux.GetGlobalManager()
			defer tmux.SetGlobalManager(oldManager)

			mockManager := new(MockManager)
			tt.setup(mockManager)
			tmux.SetGlobalManager(mockManager)

			// Act
			err := tmux.CheckTmuxInstalled()

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
			mockManager.AssertExpectations(t)
		})
	}
}

func TestSessionExistsWithMock(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*MockManager)
		want        bool
		wantErr     bool
	}{
		{
			name:        "セッションが存在する場合",
			sessionName: "test-session",
			setup: func(m *MockManager) {
				m.On("SessionExists", "test-session").Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:        "セッションが存在しない場合",
			sessionName: "non-existent",
			setup: func(m *MockManager) {
				m.On("SessionExists", "non-existent").Return(false, nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name:        "エラーが発生する場合",
			sessionName: "error-session",
			setup: func(m *MockManager) {
				m.On("SessionExists", "error-session").Return(false, errors.New("tmux error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			oldManager := tmux.GetGlobalManager()
			defer tmux.SetGlobalManager(oldManager)

			mockManager := new(MockManager)
			tt.setup(mockManager)
			tmux.SetGlobalManager(mockManager)

			// Act
			got, err := tmux.SessionExists(tt.sessionName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			mockManager.AssertExpectations(t)
		})
	}
}

func TestCreateSessionWithMock(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*MockManager)
		wantErr     bool
	}{
		{
			name:        "セッションを正常に作成",
			sessionName: "new-session",
			setup: func(m *MockManager) {
				m.On("CreateSession", "new-session").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "セッション作成に失敗",
			sessionName: "fail-session",
			setup: func(m *MockManager) {
				m.On("CreateSession", "fail-session").Return(errors.New("creation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			oldManager := tmux.GetGlobalManager()
			defer tmux.SetGlobalManager(oldManager)

			mockManager := new(MockManager)
			tt.setup(mockManager)
			tmux.SetGlobalManager(mockManager)

			// Act
			err := tmux.CreateSession(tt.sessionName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockManager.AssertExpectations(t)
		})
	}
}

func TestEnsureSessionWithMock(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*MockManager)
		wantErr     bool
	}{
		{
			name:        "既存のセッションの場合",
			sessionName: "existing-session",
			setup: func(m *MockManager) {
				m.On("EnsureSession", "existing-session").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "新規セッション作成の場合",
			sessionName: "new-session",
			setup: func(m *MockManager) {
				m.On("EnsureSession", "new-session").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "エラーが発生する場合",
			sessionName: "error-session",
			setup: func(m *MockManager) {
				m.On("EnsureSession", "error-session").Return(errors.New("ensure failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			oldManager := tmux.GetGlobalManager()
			defer tmux.SetGlobalManager(oldManager)

			mockManager := new(MockManager)
			tt.setup(mockManager)
			tmux.SetGlobalManager(mockManager)

			// Act
			err := tmux.EnsureSession(tt.sessionName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockManager.AssertExpectations(t)
		})
	}
}

func TestListSessionsWithMock(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		setup   func(*MockManager)
		want    []string
		wantErr bool
	}{
		{
			name:   "セッション一覧を取得",
			prefix: "test-",
			setup: func(m *MockManager) {
				m.On("ListSessions", "test-").Return([]string{"test-1", "test-2"}, nil)
			},
			want:    []string{"test-1", "test-2"},
			wantErr: false,
		},
		{
			name:   "空のセッション一覧",
			prefix: "empty-",
			setup: func(m *MockManager) {
				m.On("ListSessions", "empty-").Return([]string{}, nil)
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name:   "エラーが発生する場合",
			prefix: "error-",
			setup: func(m *MockManager) {
				m.On("ListSessions", "error-").Return(nil, errors.New("list failed"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			oldManager := tmux.GetGlobalManager()
			defer tmux.SetGlobalManager(oldManager)

			mockManager := new(MockManager)
			tt.setup(mockManager)
			tmux.SetGlobalManager(mockManager)

			// Act
			got, err := tmux.ListSessions(tt.prefix)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			mockManager.AssertExpectations(t)
		})
	}
}
