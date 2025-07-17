package tmux_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
)

func TestDefaultManager_CheckTmuxInstalled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockCommandExecutor)
		wantErr bool
	}{
		{
			name: "tmuxがインストールされている場合",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "which", []string{"tmux"}).Return("/usr/bin/tmux", nil)
			},
			wantErr: false,
		},
		{
			name: "tmuxがインストールされていない場合",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "which", []string{"tmux"}).Return("", errors.New("command not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockExecutor := mocks.NewMockCommandExecutor()
			tt.setup(mockExecutor)
			manager := tmux.NewDefaultManagerWithExecutor(mockExecutor)

			// Act
			err := manager.CheckTmuxInstalled()

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tmux.ErrTmuxNotInstalled, err)
			} else {
				assert.NoError(t, err)
			}
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_SessionExists(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*mocks.MockCommandExecutor)
		want        bool
		wantErr     bool
	}{
		{
			name:        "セッションが存在する場合",
			sessionName: "test-session",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"has-session", "-t", "test-session"}).Return("", nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:        "セッションが存在しない場合",
			sessionName: "non-existent",
			setup: func(m *mocks.MockCommandExecutor) {
				exitErr := &tmux.MockExitError{ExitCode: 1}
				m.On("Execute", "tmux", []string{"has-session", "-t", "non-existent"}).Return("", exitErr)
			},
			want:    false,
			wantErr: false,
		},
		{
			name:        "その他のエラーが発生する場合",
			sessionName: "error-session",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"has-session", "-t", "error-session"}).Return("", errors.New("tmux error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockExecutor := mocks.NewMockCommandExecutor()
			tt.setup(mockExecutor)
			manager := tmux.NewDefaultManagerWithExecutor(mockExecutor)

			// Act
			got, err := manager.SessionExists(tt.sessionName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_CreateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*mocks.MockCommandExecutor)
		wantErr     bool
	}{
		{
			name:        "セッションを正常に作成",
			sessionName: "new-session",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"new-session", "-d", "-s", "new-session"}).Return("", nil)
			},
			wantErr: false,
		},
		{
			name:        "セッション作成に失敗",
			sessionName: "fail-session",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"new-session", "-d", "-s", "fail-session"}).Return("", errors.New("creation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockExecutor := mocks.NewMockCommandExecutor()
			tt.setup(mockExecutor)
			manager := tmux.NewDefaultManagerWithExecutor(mockExecutor)

			// Act
			err := manager.CreateSession(tt.sessionName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_CreateWindow(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		setup       func(*mocks.MockCommandExecutor)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "ウィンドウを正常に作成",
			sessionName: "test-session",
			windowName:  "test-window",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"new-window", "-t", "test-session", "-n", "test-window"}).Return("", nil)
			},
			wantErr: false,
		},
		{
			name:        "セッション名が空の場合",
			sessionName: "",
			windowName:  "test-window",
			setup:       func(m *mocks.MockCommandExecutor) {},
			wantErr:     true,
			errMsg:      "session name cannot be empty",
		},
		{
			name:        "ウィンドウ名が空の場合",
			sessionName: "test-session",
			windowName:  "",
			setup:       func(m *mocks.MockCommandExecutor) {},
			wantErr:     true,
			errMsg:      "window name cannot be empty",
		},
		{
			name:        "ウィンドウ作成に失敗",
			sessionName: "test-session",
			windowName:  "fail-window",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"new-window", "-t", "test-session", "-n", "fail-window"}).Return("", errors.New("creation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockExecutor := mocks.NewMockCommandExecutor()
			tt.setup(mockExecutor)
			manager := tmux.NewDefaultManagerWithExecutor(mockExecutor)

			// Act
			err := manager.CreateWindow(tt.sessionName, tt.windowName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_WindowExists(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		setup       func(*mocks.MockCommandExecutor)
		want        bool
		wantErr     bool
	}{
		{
			name:        "ウィンドウが存在する場合",
			sessionName: "test-session",
			windowName:  "test-window",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"list-windows", "-t", "test-session", "-F", "#{window_name}"}).Return("test-window\nother-window", nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:        "ウィンドウが存在しない場合",
			sessionName: "test-session",
			windowName:  "non-existent",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"list-windows", "-t", "test-session", "-F", "#{window_name}"}).Return("test-window\nother-window", nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name:        "エラーが発生する場合",
			sessionName: "error-session",
			windowName:  "test-window",
			setup: func(m *mocks.MockCommandExecutor) {
				m.On("Execute", "tmux", []string{"list-windows", "-t", "error-session", "-F", "#{window_name}"}).Return("", errors.New("list failed"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockExecutor := mocks.NewMockCommandExecutor()
			tt.setup(mockExecutor)
			manager := tmux.NewDefaultManagerWithExecutor(mockExecutor)

			// Act
			got, err := manager.WindowExists(tt.sessionName, tt.windowName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_GetIssueWindow(t *testing.T) {
	manager := tmux.NewDefaultManager()

	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "Issue番号1",
			issueNumber: 1,
			want:        "issue-1",
		},
		{
			name:        "Issue番号123",
			issueNumber: 123,
			want:        "issue-123",
		},
		{
			name:        "Issue番号9999",
			issueNumber: 9999,
			want:        "issue-9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.GetIssueWindow(tt.issueNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultManager_FindIssueWindow(t *testing.T) {
	manager := tmux.NewDefaultManager()

	tests := []struct {
		name       string
		windowName string
		wantNum    int
		wantFound  bool
	}{
		{
			name:       "issue-123形式",
			windowName: "issue-123",
			wantNum:    123,
			wantFound:  true,
		},
		{
			name:       "123-plan形式",
			windowName: "123-plan",
			wantNum:    123,
			wantFound:  true,
		},
		{
			name:       "456-implement形式",
			windowName: "456-implement",
			wantNum:    456,
			wantFound:  true,
		},
		{
			name:       "789-review形式",
			windowName: "789-review",
			wantNum:    789,
			wantFound:  true,
		},
		{
			name:       "無効なフェーズ",
			windowName: "123-invalid",
			wantNum:    0,
			wantFound:  false,
		},
		{
			name:       "数字でない",
			windowName: "abc-plan",
			wantNum:    0,
			wantFound:  false,
		},
		{
			name:       "無関係な名前",
			windowName: "random-window",
			wantNum:    0,
			wantFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNum, gotFound := manager.FindIssueWindow(tt.windowName)
			assert.Equal(t, tt.wantNum, gotNum)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}
