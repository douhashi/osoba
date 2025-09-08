package tmux

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockCommandExecutorは既存のpane_test.goで定義済みのため、このファイルでは削除

func TestSessionDiagnostics_Basic(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windows     int
		attached    bool
		created     string
		wantValid   bool
	}{
		{
			name:        "valid session diagnostics",
			sessionName: "osoba-test",
			windows:     3,
			attached:    true,
			created:     "2025-01-08T10:00:00Z",
			wantValid:   true,
		},
		{
			name:        "empty session name",
			sessionName: "",
			windows:     1,
			attached:    false,
			created:     "2025-01-08T10:00:00Z",
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diag := &SessionDiagnostics{
				Name:      tt.sessionName,
				Windows:   tt.windows,
				Attached:  tt.attached,
				Created:   tt.created,
				Timestamp: time.Now(),
			}

			if tt.wantValid {
				assert.NotEmpty(t, diag.Name)
				assert.GreaterOrEqual(t, diag.Windows, 0)
			} else {
				assert.Empty(t, diag.Name)
			}
		})
	}
}

func TestWindowDiagnostics_Basic(t *testing.T) {
	tests := []struct {
		name       string
		windowName string
		exists     bool
		panes      int
		active     bool
		wantValid  bool
	}{
		{
			name:       "valid window diagnostics",
			windowName: "330-plan",
			exists:     true,
			panes:      2,
			active:     true,
			wantValid:  true,
		},
		{
			name:       "non-existent window",
			windowName: "999-missing",
			exists:     false,
			panes:      0,
			active:     false,
			wantValid:  true, // still valid diagnostics
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diag := &WindowDiagnostics{
				Name:      tt.windowName,
				Exists:    tt.exists,
				Panes:     tt.panes,
				Active:    tt.active,
				Timestamp: time.Now(),
			}

			if tt.wantValid {
				assert.NotEmpty(t, diag.Name)
				assert.GreaterOrEqual(t, diag.Panes, 0)
			}
		})
	}
}

func TestDefaultManager_DiagnoseSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		mockOutput  string
		mockError   error
		wantError   bool
	}{
		{
			name:        "successful session diagnosis",
			sessionName: "osoba-test",
			mockOutput:  "osoba-test:3:1641641600:1",
			mockError:   nil,
			wantError:   false,
		},
		{
			name:        "session not found",
			sessionName: "nonexistent",
			mockOutput:  "",
			mockError:   &MockExitError{ExitCode: 1},
			wantError:   false, // should handle gracefully
		},
		{
			name:        "empty session name",
			sessionName: "",
			mockOutput:  "",
			mockError:   nil,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := new(MockCommandExecutor)
			manager := NewDefaultManagerWithExecutor(mockExec)

			if tt.sessionName != "" {
				args := []string{"list-sessions", "-t", tt.sessionName, 
					"-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}"}
				mockExec.On("Execute", "tmux", args).
					Return(tt.mockOutput, tt.mockError)
			}

			result, err := manager.DiagnoseSession(tt.sessionName)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				if tt.mockError == nil {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.Equal(t, tt.sessionName, result.Name)
				} else if _, isExit := IsExitError(tt.mockError); isExit {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.Equal(t, tt.sessionName, result.Name)
				}
			}

			mockExec.AssertExpectations(t)
		})
	}
}

func TestDefaultManager_DiagnoseWindow(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		windowName  string
		mockOutput  string
		mockError   error
		wantError   bool
	}{
		{
			name:        "successful window diagnosis",
			sessionName: "osoba-test",
			windowName:  "330-plan",
			mockOutput:  "0:330-plan:1:2",
			mockError:   nil,
			wantError:   false,
		},
		{
			name:        "window not found",
			sessionName: "osoba-test",
			windowName:  "nonexistent",
			mockOutput:  "",
			mockError:   &MockExitError{ExitCode: 1},
			wantError:   false, // should handle gracefully
		},
		{
			name:        "empty session name",
			sessionName: "",
			windowName:  "test-window",
			mockOutput:  "",
			mockError:   nil,
			wantError:   true,
		},
		{
			name:        "empty window name",
			sessionName: "osoba-test",
			windowName:  "",
			mockOutput:  "",
			mockError:   nil,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := new(MockCommandExecutor)
			manager := NewDefaultManagerWithExecutor(mockExec)

			if tt.sessionName != "" && tt.windowName != "" {
				args := []string{"list-windows", "-t", tt.sessionName,
					"-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}
				mockExec.On("Execute", "tmux", args).
					Return(tt.mockOutput, tt.mockError)
			}

			result, err := manager.DiagnoseWindow(tt.sessionName, tt.windowName)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				if tt.mockError == nil {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.Equal(t, tt.windowName, result.Name)
				} else if _, isExit := IsExitError(tt.mockError); isExit {
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.Equal(t, tt.windowName, result.Name)
				}
			}

			if tt.sessionName != "" && tt.windowName != "" {
				mockExec.AssertExpectations(t)
			}
		})
	}
}

func TestDefaultManager_ListSessionDiagnostics(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		mockOutput string
		mockError  error
		wantError  bool
		wantCount  int
	}{
		{
			name:       "multiple sessions found",
			prefix:     "osoba-",
			mockOutput: "osoba-test:2:1641641600:0\nosoba-prod:3:1641641700:1",
			mockError:  nil,
			wantError:  false,
			wantCount:  2,
		},
		{
			name:       "no sessions found",
			prefix:     "nonexistent-",
			mockOutput: "",
			mockError:  &MockExitError{ExitCode: 1},
			wantError:  false,
			wantCount:  0,
		},
		{
			name:       "tmux command error",
			prefix:     "osoba-",
			mockOutput: "",
			mockError:  assert.AnError,
			wantError:  true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := new(MockCommandExecutor)
			manager := NewDefaultManagerWithExecutor(mockExec)

			args := []string{"list-sessions", 
				"-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}"}
			mockExec.On("Execute", "tmux", args).
				Return(tt.mockOutput, tt.mockError)

			result, err := manager.ListSessionDiagnostics(tt.prefix)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.wantCount)
				
				for _, diag := range result {
					assert.Contains(t, diag.Name, tt.prefix)
				}
			}

			mockExec.AssertExpectations(t)
		})
	}
}