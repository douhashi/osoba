package tmux

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCommandExecutor はコマンド実行のモック
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(cmd string, args ...string) (string, error) {
	argsList := []interface{}{cmd}
	for _, arg := range args {
		argsList = append(argsList, arg)
	}
	results := m.Called(argsList...)
	return results.String(0), results.Error(1)
}

func TestCreateWindow(t *testing.T) {
	t.Run("正常系: ウィンドウが正常に作成される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "new-window", "-t", sessionName, "-n", windowName).Return("", nil)

		// Act
		err := CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: tmuxコマンドがエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"
		expectedErr := errors.New("session not found")

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "new-window", "-t", sessionName, "-n", windowName).Return("", expectedErr)

		// Act
		err := CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create window")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のセッション名", func(t *testing.T) {
		// Arrange
		sessionName := ""
		windowName := "issue-13"
		mockExec := new(MockCommandExecutor)

		// Act
		err := CreateWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session name cannot be empty")
	})

	t.Run("異常系: 空のウィンドウ名", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := ""
		mockExec := new(MockCommandExecutor)

		// Act
		err := CreateWindowWithExecutor(sessionName, windowName, mockExec)

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

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "select-window", "-t", sessionName+":"+windowName).Return("", nil)

		// Act
		err := SwitchToWindowWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: tmuxコマンドがエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-13"
		expectedErr := errors.New("window not found")

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "select-window", "-t", sessionName+":"+windowName).Return("", expectedErr)

		// Act
		err := SwitchToWindowWithExecutor(sessionName, windowName, mockExec)

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

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("issue-12\nissue-13\nissue-14", nil)

		// Act
		exists, err := WindowExistsWithExecutor(sessionName, windowName, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.True(t, exists)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ウィンドウが存在しない場合", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-99"

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("issue-12\nissue-13\nissue-14", nil)

		// Act
		exists, err := WindowExistsWithExecutor(sessionName, windowName, mockExec)

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

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("", expectedErr)

		// Act
		exists, err := WindowExistsWithExecutor(sessionName, windowName, mockExec)

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
			got := GetWindowName(tt.issueNumber)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}
