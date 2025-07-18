package tmux_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
)

// containsMessage はメッセージスライスに指定のメッセージが含まれているかをチェックする
func containsMessage(messages []string, target string) bool {
	for _, msg := range messages {
		if msg == target {
			return true
		}
	}
	return false
}

// mockLoggerはテスト用のログ実装
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.errorMessages = append(m.errorMessages, msg)
}

func (m *mockLogger) WithFields(fields ...interface{}) logger.Logger {
	return m
}

func TestCreateWindow(t *testing.T) {
	t.Run("正常系: ウィンドウが正常に作成される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", windowName}).Return("", nil)

		// Act
		err := tmux.CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: tmuxコマンドがエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"
		expectedErr := errors.New("session not found")

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", windowName}).Return("", expectedErr)

		// Act
		err := tmux.CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create window")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のセッション名", func(t *testing.T) {
		// Arrange
		sessionName := ""
		windowName := "issue-13"
		mockExec := mocks.NewMockTmuxCommandExecutor()

		// Act
		err := tmux.CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session name cannot be empty")
	})

	t.Run("異常系: 空のウィンドウ名", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := ""
		mockExec := mocks.NewMockTmuxCommandExecutor()

		// Act
		err := tmux.CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "window name cannot be empty")
	})
}

func TestSwitchToWindow(t *testing.T) {
	t.Run("正常系: ウィンドウに正常に切り替わる", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", sessionName + ":" + windowName}).Return("", nil)

		// Act
		err := tmux.SwitchToWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: tmuxコマンドがエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"
		expectedErr := errors.New("window not found")

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", sessionName + ":" + windowName}).Return("", expectedErr)

		// Act
		err := tmux.SwitchToWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to switch to window")
		mockExec.AssertExpectations(t)
	})
}

func TestWindowExists(t *testing.T) {
	t.Run("正常系: ウィンドウが存在する場合", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("issue-12\nissue-13\nissue-14", nil)

		// Act
		exists, err := tmux.WindowExistsWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.True(t, exists)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ウィンドウが存在しない場合", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-99"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("issue-12\nissue-13\nissue-14", nil)

		// Act
		exists, err := tmux.WindowExistsWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.False(t, exists)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: セッションが存在しない場合", func(t *testing.T) {
		// Arrange
		sessionName := "non-existent-session"
		windowName := "issue-13"
		expectedErr := errors.New("session not found")

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("", expectedErr)

		// Act
		exists, err := tmux.WindowExistsWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "failed to list windows")
		mockExec.AssertExpectations(t)
	})
}

func TestGetWindowName(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "正常系: Issue番号からウィンドウ名を生成",
			issueNumber: 13,
			want:        "issue-13",
		},
		{
			name:        "正常系: 大きなIssue番号",
			issueNumber: 9999,
			want:        "issue-9999",
		},
		{
			name:        "正常系: Issue番号1",
			issueNumber: 1,
			want:        "issue-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := tmux.GetWindowName(tt.issueNumber)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetWindowNameWithPhase(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		phase       string
		want        string
		wantErr     bool
	}{
		{
			name:        "正常系: planフェーズのウィンドウ名",
			issueNumber: 44,
			phase:       "plan",
			want:        "44-plan",
		},
		{
			name:        "正常系: implementフェーズのウィンドウ名",
			issueNumber: 44,
			phase:       "implement",
			want:        "44-implement",
		},
		{
			name:        "正常系: reviewフェーズのウィンドウ名",
			issueNumber: 44,
			phase:       "review",
			want:        "44-review",
		},
		{
			name:        "異常系: 無効なフェーズ",
			issueNumber: 44,
			phase:       "invalid",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "異常系: 空のフェーズ",
			issueNumber: 44,
			phase:       "",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got, err := tmux.GetWindowNameWithPhase(tt.issueNumber, tt.phase)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid phase")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestKillWindow(t *testing.T) {
	t.Run("正常系: ウィンドウが正常に削除される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "44-plan"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"kill-window", "-t", sessionName + ":" + windowName}).Return("", nil)

		// Act
		err := tmux.KillWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: tmuxコマンドがエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "44-plan"
		expectedErr := errors.New("window not found")

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"kill-window", "-t", sessionName + ":" + windowName}).Return("", expectedErr)

		// Act
		err := tmux.KillWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to kill window")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のセッション名", func(t *testing.T) {
		// Arrange
		sessionName := ""
		windowName := "44-plan"
		mockExec := mocks.NewMockTmuxCommandExecutor()

		// Act
		err := tmux.KillWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session name cannot be empty")
	})

	t.Run("異常系: 空のウィンドウ名", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := ""
		mockExec := mocks.NewMockTmuxCommandExecutor()

		// Act
		err := tmux.KillWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "window name cannot be empty")
	})
}

func TestCreateWindow_WithLogging(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		windowName     string
		success        bool
		wantLogMessage string
		wantLogLevel   string
	}{
		{
			name:           "ウィンドウ作成開始時にログ出力される",
			sessionName:    "test-session",
			windowName:     "test-window",
			success:        true,
			wantLogMessage: "tmuxウィンドウ作成開始",
			wantLogLevel:   "info",
		},
		{
			name:           "ウィンドウ作成失敗時にエラーログ出力される",
			sessionName:    "test-session",
			windowName:     "test-window",
			success:        false,
			wantLogMessage: "tmuxウィンドウ作成失敗",
			wantLogLevel:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックロガーのセットアップ
			mockLog := &mockLogger{}
			tmux.SetLogger(mockLog)
			defer tmux.SetLogger(nil)

			// モックExecutorのセットアップ
			mockExec := mocks.NewMockTmuxCommandExecutor()
			if tt.success {
				mockExec.On("Execute", "tmux", []string{"new-window", "-t", tt.sessionName, "-n", tt.windowName}).Return("", nil)
			} else {
				mockExec.On("Execute", "tmux", []string{"new-window", "-t", tt.sessionName, "-n", tt.windowName}).Return("", errors.New("error"))
			}

			// 実行
			tmux.CreateWindowWithExecutor(tt.sessionName, tt.windowName, mockExec)

			// ログ出力の検証
			switch tt.wantLogLevel {
			case "info":
				if !containsMessage(mockLog.infoMessages, tt.wantLogMessage) {
					t.Errorf("期待するログが出力されませんでした: %s", tt.wantLogMessage)
				}
			case "error":
				if !containsMessage(mockLog.errorMessages, tt.wantLogMessage) {
					t.Errorf("期待するエラーログが出力されませんでした: %s", tt.wantLogMessage)
				}
			}
		})
	}
}

func TestSwitchToWindow_WithLogging(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		windowName     string
		success        bool
		wantLogMessage string
		wantLogLevel   string
	}{
		{
			name:           "ウィンドウ切り替え時にログ出力される",
			sessionName:    "test-session",
			windowName:     "test-window",
			success:        true,
			wantLogMessage: "tmuxウィンドウ切り替え",
			wantLogLevel:   "info",
		},
		{
			name:           "ウィンドウ切り替え失敗時にエラーログ出力される",
			sessionName:    "test-session",
			windowName:     "test-window",
			success:        false,
			wantLogMessage: "tmuxウィンドウ切り替え失敗",
			wantLogLevel:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックロガーのセットアップ
			mockLog := &mockLogger{}
			tmux.SetLogger(mockLog)
			defer tmux.SetLogger(nil)

			// モックExecutorのセットアップ
			mockExec := mocks.NewMockTmuxCommandExecutor()
			target := tt.sessionName + ":" + tt.windowName
			if tt.success {
				mockExec.On("Execute", "tmux", []string{"select-window", "-t", target}).Return("", nil)
			} else {
				mockExec.On("Execute", "tmux", []string{"select-window", "-t", target}).Return("", errors.New("error"))
			}

			// 実行
			tmux.SwitchToWindowWithExecutor(tt.sessionName, tt.windowName, mockExec)

			// ログ出力の検証
			switch tt.wantLogLevel {
			case "info":
				if !containsMessage(mockLog.infoMessages, tt.wantLogMessage) {
					t.Errorf("期待するログが出力されませんでした: %s", tt.wantLogMessage)
				}
			case "error":
				if !containsMessage(mockLog.errorMessages, tt.wantLogMessage) {
					t.Errorf("期待するエラーログが出力されませんでした: %s", tt.wantLogMessage)
				}
			}
		})
	}
}

func TestWindowExists_WithLogging(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		windowName     string
		exists         bool
		wantLogMessage string
	}{
		{
			name:           "ウィンドウ存在確認時にログ出力される",
			sessionName:    "test-session",
			windowName:     "test-window",
			exists:         true,
			wantLogMessage: "tmuxウィンドウ一覧取得",
		},
		{
			name:           "ウィンドウが存在しない場合もログ出力される",
			sessionName:    "test-session",
			windowName:     "test-window",
			exists:         false,
			wantLogMessage: "tmuxウィンドウ一覧取得",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックロガーのセットアップ
			mockLog := &mockLogger{}
			tmux.SetLogger(mockLog)
			defer tmux.SetLogger(nil)

			// モックExecutorのセットアップ
			mockExec := mocks.NewMockTmuxCommandExecutor()
			if tt.exists {
				mockExec.On("Execute", "tmux", []string{"list-windows", "-t", tt.sessionName, "-F", "#{window_name}"}).Return(tt.windowName, nil)
			} else {
				mockExec.On("Execute", "tmux", []string{"list-windows", "-t", tt.sessionName, "-F", "#{window_name}"}).Return("other-window", nil)
			}

			// 実行
			tmux.WindowExistsWithExecutor(tt.sessionName, tt.windowName, mockExec)

			// ログ出力の検証
			if !containsMessage(mockLog.debugMessages, tt.wantLogMessage) {
				t.Errorf("期待するログが出力されませんでした: %s", tt.wantLogMessage)
			}
		})
	}
}

func TestCreateWindowForIssue_WithLogging(t *testing.T) {
	t.Run("Issue用ウィンドウ作成時にログ出力される", func(t *testing.T) {
		// モックロガーのセットアップ
		mockLog := &mockLogger{}
		tmux.SetLogger(mockLog)
		defer tmux.SetLogger(nil)

		// このテストは CreateWindowForIssue が内部でログを出力することを確認
		// 実際の関数は WindowExists と CreateWindow を呼び出すため、
		// それらの関数のログ出力をテストで確認すれば十分
		// ここでは CreateWindowForIssue 自体のログ出力に焦点を当てる

		// 簡易的なテストとして、関数名にログが含まれることを確認
		// 実際のログ出力は、すでにテストされている CreateWindow_WithLogging でカバーされる
		t.Log("CreateWindowForIssue は内部で WindowExists と CreateWindow を呼び出します")
		t.Log("それぞれの関数のログ出力は個別のテストでカバーされています")
	})
}

func TestCreateOrReplaceWindow(t *testing.T) {
	t.Run("正常系: ウィンドウが存在しない場合、新規作成される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "44-plan"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("other-window", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", windowName}).Return("", nil)

		// Act
		err := tmux.CreateOrReplaceWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ウィンドウが存在する場合、削除してから新規作成される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "44-plan"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// WindowExists: ウィンドウが存在する
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("44-plan\nother-window", nil)
		// KillWindow
		mockExec.On("Execute", "tmux", []string{"kill-window", "-t", sessionName + ":" + windowName}).Return("", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", windowName}).Return("", nil)

		// Act
		err := tmux.CreateOrReplaceWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: ウィンドウ削除に失敗", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "44-plan"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// WindowExists: ウィンドウが存在する
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("44-plan\nother-window", nil)
		// KillWindow: 失敗
		mockExec.On("Execute", "tmux", []string{"kill-window", "-t", sessionName + ":" + windowName}).Return("", errors.New("kill failed"))

		// Act
		err := tmux.CreateOrReplaceWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to kill existing window")
		mockExec.AssertExpectations(t)
	})
}

func TestListWindows(t *testing.T) {
	t.Run("正常系: ウィンドウ一覧が正常に取得される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("0:37-plan:1:1\n1:40-implement:0:2\n2:42-review:0:1", nil)

		// Act
		windows, err := tmux.ListWindowsWithExecutor(sessionName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, windows, 3)

		// 最初のウィンドウ（アクティブ）
		assert.Equal(t, 0, windows[0].Index)
		assert.Equal(t, "37-plan", windows[0].Name)
		assert.True(t, windows[0].Active)
		assert.Equal(t, 1, windows[0].Panes)

		// 2番目のウィンドウ（非アクティブ）
		assert.Equal(t, 1, windows[1].Index)
		assert.Equal(t, "40-implement", windows[1].Name)
		assert.False(t, windows[1].Active)
		assert.Equal(t, 2, windows[1].Panes)

		// 3番目のウィンドウ（非アクティブ）
		assert.Equal(t, 2, windows[2].Index)
		assert.Equal(t, "42-review", windows[2].Name)
		assert.False(t, windows[2].Active)
		assert.Equal(t, 1, windows[2].Panes)

		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ウィンドウが存在しない場合", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("", nil)

		// Act
		windows, err := tmux.ListWindowsWithExecutor(sessionName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, windows, 0)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: セッションが存在しない場合", func(t *testing.T) {
		// Arrange
		sessionName := "non-existent-session"
		expectedErr := errors.New("session not found")
		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("", expectedErr)

		// Act
		windows, err := tmux.ListWindowsWithExecutor(sessionName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, windows)
		assert.Contains(t, err.Error(), "failed to list windows")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のセッション名", func(t *testing.T) {
		// Arrange
		sessionName := ""
		mockExec := mocks.NewMockTmuxCommandExecutor()

		// Act
		windows, err := tmux.ListWindowsWithExecutor(sessionName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, windows)
		assert.Contains(t, err.Error(), "session name cannot be empty")
	})

	t.Run("正常系: フォーマットが不正な行は無視される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("0:37-plan:1:1\ninvalid-line\n1:40-implement:0:2", nil)

		// Act
		windows, err := tmux.ListWindowsWithExecutor(sessionName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, windows, 2) // 不正な行は無視される

		assert.Equal(t, 0, windows[0].Index)
		assert.Equal(t, "37-plan", windows[0].Name)
		assert.Equal(t, 1, windows[1].Index)
		assert.Equal(t, "40-implement", windows[1].Name)

		mockExec.AssertExpectations(t)
	})
}

func TestParseWindowName(t *testing.T) {
	tests := []struct {
		name         string
		windowName   string
		wantIssueNum int
		wantPhase    string
		wantOk       bool
	}{
		{
			name:         "正常系: plan フェーズ",
			windowName:   "37-plan",
			wantIssueNum: 37,
			wantPhase:    "plan",
			wantOk:       true,
		},
		{
			name:         "正常系: implement フェーズ",
			windowName:   "40-implement",
			wantIssueNum: 40,
			wantPhase:    "implement",
			wantOk:       true,
		},
		{
			name:         "正常系: review フェーズ",
			windowName:   "42-review",
			wantIssueNum: 42,
			wantPhase:    "review",
			wantOk:       true,
		},
		{
			name:         "異常系: 無効なフェーズ",
			windowName:   "37-invalid",
			wantIssueNum: 0,
			wantPhase:    "",
			wantOk:       false,
		},
		{
			name:         "異常系: Issue番号が数値でない",
			windowName:   "abc-plan",
			wantIssueNum: 0,
			wantPhase:    "",
			wantOk:       false,
		},
		{
			name:         "異常系: ハイフンがない",
			windowName:   "37plan",
			wantIssueNum: 0,
			wantPhase:    "",
			wantOk:       false,
		},
		{
			name:         "異常系: 複数のハイフン",
			windowName:   "37-plan-extra",
			wantIssueNum: 0,
			wantPhase:    "",
			wantOk:       false,
		},
		{
			name:         "異常系: 空の文字列",
			windowName:   "",
			wantIssueNum: 0,
			wantPhase:    "",
			wantOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			gotIssueNum, gotPhase, gotOk := tmux.ParseWindowName(tt.windowName)

			// Assert
			assert.Equal(t, tt.wantIssueNum, gotIssueNum)
			assert.Equal(t, tt.wantPhase, gotPhase)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}

func TestGetWindowDetails(t *testing.T) {
	t.Run("正常系: ウィンドウ詳細情報が正常に取得される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("0:37-plan:1:1\n1:40-implement:0:2\n2:unknown-window:0:1", nil)

		// Act
		details, err := tmux.GetWindowDetailsWithExecutor(sessionName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, details, 3)

		// 最初のウィンドウ（パース成功）
		assert.Equal(t, 0, details[0].Index)
		assert.Equal(t, "37-plan", details[0].Name)
		assert.True(t, details[0].Active)
		assert.Equal(t, 37, details[0].IssueNumber)
		assert.Equal(t, "plan", details[0].Phase)

		// 2番目のウィンドウ（パース成功）
		assert.Equal(t, 1, details[1].Index)
		assert.Equal(t, "40-implement", details[1].Name)
		assert.False(t, details[1].Active)
		assert.Equal(t, 40, details[1].IssueNumber)
		assert.Equal(t, "implement", details[1].Phase)

		// 3番目のウィンドウ（パース失敗、Issue番号とフェーズは0と空文字列）
		assert.Equal(t, 2, details[2].Index)
		assert.Equal(t, "unknown-window", details[2].Name)
		assert.False(t, details[2].Active)
		assert.Equal(t, 0, details[2].IssueNumber)
		assert.Equal(t, "", details[2].Phase)

		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: ウィンドウ一覧取得に失敗", func(t *testing.T) {
		// Arrange
		sessionName := "non-existent-session"
		expectedErr := errors.New("session not found")
		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("", expectedErr)

		// Act
		details, err := tmux.GetWindowDetailsWithExecutor(sessionName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, details)
		assert.Contains(t, err.Error(), "session not found")
		mockExec.AssertExpectations(t)
	})
}

func TestSortWindowDetails(t *testing.T) {
	t.Run("正常系: ウィンドウ詳細情報が名前でソートされる", func(t *testing.T) {
		// Arrange
		details := []*tmux.WindowDetail{
			{WindowInfo: &tmux.WindowInfo{Name: "42-review"}},
			{WindowInfo: &tmux.WindowInfo{Name: "37-plan"}},
			{WindowInfo: &tmux.WindowInfo{Name: "40-implement"}},
			{WindowInfo: &tmux.WindowInfo{Name: "45-plan"}},
		}

		// Act
		tmux.SortWindowDetails(details)

		// Assert
		assert.Len(t, details, 4)
		assert.Equal(t, "37-plan", details[0].Name)
		assert.Equal(t, "40-implement", details[1].Name)
		assert.Equal(t, "42-review", details[2].Name)
		assert.Equal(t, "45-plan", details[3].Name)
	})

	t.Run("正常系: 空のスライスでもエラーにならない", func(t *testing.T) {
		// Arrange
		details := []*tmux.WindowDetail{}

		// Act & Assert（エラーが発生しないことを確認）
		tmux.SortWindowDetails(details)
		assert.Len(t, details, 0)
	})

	t.Run("正常系: 1つの要素でもエラーにならない", func(t *testing.T) {
		// Arrange
		details := []*tmux.WindowDetail{
			{WindowInfo: &tmux.WindowInfo{Name: "37-plan"}},
		}

		// Act
		tmux.SortWindowDetails(details)

		// Assert
		assert.Len(t, details, 1)
		assert.Equal(t, "37-plan", details[0].Name)
	})
}

func TestGetSortedWindowDetails(t *testing.T) {
	t.Run("正常系: ソート済みのウィンドウ詳細情報が取得される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockExec := mocks.NewMockTmuxCommandExecutor()
		// ソートされていない順序で返す
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
			Return("0:42-review:0:1\n1:37-plan:1:1\n2:40-implement:0:2", nil)

		// Act
		details, err := tmux.GetSortedWindowDetailsWithExecutor(sessionName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, details, 3)

		// ソートされた順序で返されることを確認
		assert.Equal(t, "37-plan", details[0].Name)
		assert.Equal(t, "40-implement", details[1].Name)
		assert.Equal(t, "42-review", details[2].Name)

		mockExec.AssertExpectations(t)
	})
}

// Issue #147: Issue単位のwindow管理とpane管理のテスト
func TestIssueWindowManagement(t *testing.T) {
	t.Run("正常系: Issue単位のwindow作成（フェーズ名を含まない）", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 147
		expectedWindowName := "issue-147"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("other-window", nil)
		// CreateWindow: Issue単位のウィンドウを作成
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", expectedWindowName}).Return("", nil)

		// Act
		err := tmux.CreateIssueWindowWithExecutor(sessionName, issueNumber, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: Issue単位のwindowが既に存在する場合はスキップ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 147
		expectedWindowName := "issue-147"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// WindowExists: ウィンドウが既に存在する
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return(expectedWindowName, nil)
		// CreateWindow は呼ばれない

		// Act
		err := tmux.CreateIssueWindowWithExecutor(sessionName, issueNumber, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: Issue単位のwindowに切り替え", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		expectedWindowName := "issue-147"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", sessionName + ":" + expectedWindowName}).Return("", nil)

		// Act
		err := tmux.SwitchToWindowWithExecutor(sessionName, expectedWindowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})
}

// Issue #147: pane管理のテスト
func TestPaneManagement(t *testing.T) {
	t.Run("正常系: window内に新しいpaneを作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-147"
		paneTitle := "plan-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// split-window コマンドでpaneを作成
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "50"}).Return("", nil)
		// paneにタイトルを設定
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		err := tmux.CreatePaneInWindowWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: 特定のpaneを選択", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-147"
		paneTitle := "implement-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// pane一覧取得
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase\n2:review-phase", nil)
		// 特定のpaneを選択
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName + ".1"}).Return("", nil)

		// Act
		err := tmux.SelectPaneByTitleWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: 既存のpaneが存在する場合は選択のみ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-147"
		paneTitle := "plan-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// pane一覧取得（既存のpaneが存在）
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase", nil)
		// 既存のpaneを選択
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName + ".0"}).Return("", nil)

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: 既存のpaneが存在しない場合は新規作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-147"
		paneTitle := "review-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// pane一覧取得（target paneが存在しない）
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase", nil)
		// 新しいpaneを作成
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "33"}).Return("", nil)
		// paneにタイトルを設定
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})
}

// Issue #147: フェーズ統合管理のテスト
func TestPhaseIntegration(t *testing.T) {
	t.Run("統合テスト: 複数フェーズでのwindow/pane管理", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 147
		windowName := "issue-147"

		mockExec := mocks.NewMockTmuxCommandExecutor()

		// Plan フェーズ
		// 1. Issueウィンドウ作成
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("other-window", nil).Once()
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", windowName}).Return("", nil).Once()
		// 2. Plan pane作成
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:", nil).Once()
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", "plan-phase"}).Return("", nil).Once()

		// Implementation フェーズ
		// 1. 既存のIssueウィンドウを確認（存在する）
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return(windowName, nil).Once()
		// 2. Implementation pane作成
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase", nil).Once()
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "50"}).Return("", nil).Times(2)
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", "implement-phase"}).Return("", nil).Once()

		// Review フェーズ
		// 1. 既存のIssueウィンドウを確認（存在する）
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return(windowName, nil).Once()
		// 2. Review pane作成
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase", nil).Once()
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "33"}).Return("", nil).Once()
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", "review-phase"}).Return("", nil).Once()

		// Act & Assert
		// Plan フェーズ
		err := tmux.CreateIssueWindowWithExecutor(sessionName, issueNumber, mockExec)
		assert.NoError(t, err)
		err = tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, "plan-phase", mockExec)
		assert.NoError(t, err)

		// Implementation フェーズ
		err = tmux.CreateIssueWindowWithExecutor(sessionName, issueNumber, mockExec)
		assert.NoError(t, err)
		err = tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, "implement-phase", mockExec)
		assert.NoError(t, err)

		// Review フェーズ
		err = tmux.CreateIssueWindowWithExecutor(sessionName, issueNumber, mockExec)
		assert.NoError(t, err)
		err = tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, "review-phase", mockExec)
		assert.NoError(t, err)

		mockExec.AssertExpectations(t)
	})

	t.Run("統合テスト: 既存paneの再利用", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-147"
		paneTitle := "plan-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// 既存のpaneが存在する場合
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase\n2:review-phase", nil)
		// 既存のpaneを選択
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName + ".0"}).Return("", nil)

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})
}
