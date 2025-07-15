package tmux

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateWindowForIssue(t *testing.T) {
	t.Run("正常系: Issue番号とフェーズからウィンドウが作成される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"
		expectedWindowName := "33-plan"

		mockExec := new(MockCommandExecutor)
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("other-window", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", "new-window", "-t", sessionName, "-n", expectedWindowName).Return("", nil)

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: implementフェーズのウィンドウ作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "implement"
		expectedWindowName := "33-implement"

		mockExec := new(MockCommandExecutor)
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("33-plan\nother-window", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", "new-window", "-t", sessionName, "-n", expectedWindowName).Return("", nil)

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: reviewフェーズのウィンドウ作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "review"
		expectedWindowName := "33-review"

		mockExec := new(MockCommandExecutor)
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("33-plan\n33-implement", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", "new-window", "-t", sessionName, "-n", expectedWindowName).Return("", nil)

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ウィンドウが既に存在する場合はスキップ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"

		mockExec := new(MockCommandExecutor)
		// WindowExists: ウィンドウが既に存在する
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("33-plan\nother-window", nil)
		// CreateWindowは呼ばれない

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 無効なフェーズ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "invalid"

		mockExec := new(MockCommandExecutor)
		// WindowExistsは呼ばれない

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid phase")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のフェーズ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := ""

		mockExec := new(MockCommandExecutor)

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid phase")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: ウィンドウ作成に失敗", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"
		expectedWindowName := "33-plan"

		mockExec := new(MockCommandExecutor)
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}").Return("other-window", nil)
		// CreateWindow: 失敗
		mockExec.On("Execute", "tmux", "new-window", "-t", sessionName, "-n", expectedWindowName).Return("", errors.New("creation failed"))

		// Act
		err := CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "creation failed")
		mockExec.AssertExpectations(t)
	})
}

func TestSwitchToIssueWindow(t *testing.T) {
	t.Run("正常系: Issue番号とフェーズからウィンドウに切り替える", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"
		expectedWindowName := "33-plan"
		expectedTarget := sessionName + ":" + expectedWindowName

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "select-window", "-t", expectedTarget).Return("", nil)

		// Act
		err := SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: implementフェーズのウィンドウに切り替え", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "implement"
		expectedWindowName := "33-implement"
		expectedTarget := sessionName + ":" + expectedWindowName

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "select-window", "-t", expectedTarget).Return("", nil)

		// Act
		err := SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: reviewフェーズのウィンドウに切り替え", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "review"
		expectedWindowName := "33-review"
		expectedTarget := sessionName + ":" + expectedWindowName

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "select-window", "-t", expectedTarget).Return("", nil)

		// Act
		err := SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 無効なフェーズ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "invalid"

		mockExec := new(MockCommandExecutor)

		// Act
		err := SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid phase")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のフェーズ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := ""

		mockExec := new(MockCommandExecutor)

		// Act
		err := SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid phase")
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: ウィンドウ切り替えに失敗", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"
		expectedWindowName := "33-plan"
		expectedTarget := sessionName + ":" + expectedWindowName

		mockExec := new(MockCommandExecutor)
		mockExec.On("Execute", "tmux", "select-window", "-t", expectedTarget).Return("", errors.New("window not found"))

		// Act
		err := SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to switch to window")
		mockExec.AssertExpectations(t)
	})
}
