package tmux_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
)

func TestSessionManager_CheckTmuxInstalled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockTmuxManager)
		wantErr bool
		errMsg  string
	}{
		{
			name: "tmuxがインストールされている場合",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("CheckTmuxInstalled").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "tmuxがインストールされていない場合",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("CheckTmuxInstalled").Return(tmux.ErrTmuxNotInstalled)
			},
			wantErr: true,
			errMsg:  "tmuxがインストールされていません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockManager := mocks.NewMockTmuxManager()
			tt.setup(mockManager)

			// Act
			err := mockManager.CheckTmuxInstalled()

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

func TestSessionManager_SessionExists(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*mocks.MockTmuxManager)
		want        bool
		wantErr     bool
	}{
		{
			name:        "セッションが存在する場合",
			sessionName: "test-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("SessionExists", "test-session").Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name:        "セッションが存在しない場合",
			sessionName: "non-existent",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("SessionExists", "non-existent").Return(false, nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name:        "エラーが発生する場合",
			sessionName: "error-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("SessionExists", "error-session").Return(false, errors.New("tmux error"))
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockManager := mocks.NewMockTmuxManager()
			tt.setup(mockManager)

			// Act
			got, err := mockManager.SessionExists(tt.sessionName)

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

func TestSessionManager_CreateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*mocks.MockTmuxManager)
		wantErr     bool
	}{
		{
			name:        "セッションを正常に作成",
			sessionName: "new-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("CreateSession", "new-session").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "セッション作成に失敗",
			sessionName: "fail-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("CreateSession", "fail-session").Return(errors.New("creation failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockManager := mocks.NewMockTmuxManager()
			tt.setup(mockManager)

			// Act
			err := mockManager.CreateSession(tt.sessionName)

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

func TestSessionManager_EnsureSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setup       func(*mocks.MockTmuxManager)
		wantErr     bool
	}{
		{
			name:        "既存のセッションの場合",
			sessionName: "existing-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("EnsureSession", "existing-session").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "新規セッション作成の場合",
			sessionName: "new-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("EnsureSession", "new-session").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "エラーが発生する場合",
			sessionName: "error-session",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("EnsureSession", "error-session").Return(errors.New("ensure failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockManager := mocks.NewMockTmuxManager()
			tt.setup(mockManager)

			// Act
			err := mockManager.EnsureSession(tt.sessionName)

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

func TestSessionManager_ListSessions(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		setup   func(*mocks.MockTmuxManager)
		want    []string
		wantErr bool
	}{
		{
			name:   "セッション一覧を取得",
			prefix: "test-",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("ListSessions", "test-").Return([]string{"test-1", "test-2"}, nil)
			},
			want:    []string{"test-1", "test-2"},
			wantErr: false,
		},
		{
			name:   "空のセッション一覧",
			prefix: "empty-",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("ListSessions", "empty-").Return([]string{}, nil)
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name:   "エラーが発生する場合",
			prefix: "error-",
			setup: func(m *mocks.MockTmuxManager) {
				m.On("ListSessions", "error-").Return(nil, errors.New("list failed"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockManager := mocks.NewMockTmuxManager()
			tt.setup(mockManager)

			// Act
			got, err := mockManager.ListSessions(tt.prefix)

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
