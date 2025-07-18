package tmux_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
)

func TestCreateWindowForIssue(t *testing.T) {
	t.Run("正常系: Issue番号とフェーズからウィンドウが作成される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"
		expectedWindowName := "33-plan"

		mockExec := mocks.NewMockCommandExecutor()
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("other-window", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", expectedWindowName}).Return("", nil)

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("33-plan\nother-window", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", expectedWindowName}).Return("", nil)

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("33-plan\n33-implement", nil)
		// CreateWindow
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", expectedWindowName}).Return("", nil)

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ウィンドウが既に存在する場合はスキップ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "plan"

		mockExec := mocks.NewMockCommandExecutor()
		// WindowExists: ウィンドウが既に存在する
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("33-plan\nother-window", nil)
		// CreateWindowは呼ばれない

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 無効なフェーズ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "invalid"

		mockExec := mocks.NewMockCommandExecutor()
		// WindowExistsは呼ばれない

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		// WindowExists: ウィンドウが存在しない
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("other-window", nil)
		// CreateWindow: 失敗
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", expectedWindowName}).Return("", errors.New("creation failed"))

		// Act
		err := tmux.CreateWindowForIssueWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", expectedTarget}).Return("", nil)

		// Act
		err := tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", expectedTarget}).Return("", nil)

		// Act
		err := tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", expectedTarget}).Return("", nil)

		// Act
		err := tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 無効なフェーズ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 33
		phase := "invalid"

		mockExec := mocks.NewMockCommandExecutor()

		// Act
		err := tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()

		// Act
		err := tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

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

		mockExec := mocks.NewMockCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"select-window", "-t", expectedTarget}).Return("", errors.New("window not found"))

		// Act
		err := tmux.SwitchToIssueWindowWithExecutor(sessionName, issueNumber, phase, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to switch to window")
		mockExec.AssertExpectations(t)
	})
}

func TestCreateWindowForIssueWithNewWindowDetection(t *testing.T) {
	t.Run("正常系: 新規ウィンドウを作成し、isNewWindowがtrueを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 149
		windowName := "issue-149"

		mockExec := mocks.NewMockCommandExecutor()

		// WindowExists check - returns false (window doesn't exist)
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("issue-1\nissue-2\n", nil)

		// Create new window
		mockExec.On("Execute", "tmux", []string{"new-window", "-t", sessionName, "-n", windowName}).Return("", nil)

		// Act
		actualWindowName, isNewWindow, err := tmux.CreateWindowForIssueWithNewWindowDetection(sessionName, issueNumber, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "issue-149", actualWindowName)
		assert.True(t, isNewWindow)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: 既存ウィンドウが存在し、isNewWindowがfalseを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 149
		expectedWindowName := "issue-149"

		mockExec := mocks.NewMockCommandExecutor()

		// WindowExists check - returns true (window exists)
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("issue-1\nissue-149\nissue-2\n", nil)

		// Act
		actualWindowName, isNewWindow, err := tmux.CreateWindowForIssueWithNewWindowDetection(sessionName, issueNumber, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedWindowName, actualWindowName)
		assert.False(t, isNewWindow)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: tmuxコマンドがエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		issueNumber := 149
		expectedErr := errors.New("session not found")

		mockExec := mocks.NewMockCommandExecutor()
		mockExec.On("Execute", "tmux", []string{"list-windows", "-t", sessionName, "-F", "#{window_name}"}).Return("", expectedErr)

		// Act
		actualWindowName, isNewWindow, err := tmux.CreateWindowForIssueWithNewWindowDetection(sessionName, issueNumber, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "", actualWindowName)
		assert.False(t, isNewWindow)
		mockExec.AssertExpectations(t)
	})
}

func TestSelectOrCreatePaneForPhaseWithNewWindowFlag(t *testing.T) {
	t.Run("正常系: 新規ウィンドウの場合、pane分割をスキップ", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-149"
		paneTitle := "plan-phase"
		isNewWindow := true

		mockExec := mocks.NewMockCommandExecutor()

		// List panes - return single pane (default pane)
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:\n", nil)

		// Select the first pane and set title
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName + ".0"}).Return("", nil)
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		paneTarget, err := tmux.SelectOrCreatePaneForPhaseWithNewWindowFlag(sessionName, windowName, paneTitle, isNewWindow, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, sessionName+":"+windowName+".0", paneTarget)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: 既存ウィンドウの場合、縦分割でpaneを作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-149"
		paneTitle := "implement-phase"
		isNewWindow := false

		mockExec := mocks.NewMockCommandExecutor()

		// List panes - return single pane (no matching pane found)
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n", nil)

		// Create new pane with horizontal split (-h)
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "50"}).Return("", nil)

		// Set pane title
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		paneTarget, err := tmux.SelectOrCreatePaneForPhaseWithNewWindowFlag(sessionName, windowName, paneTitle, isNewWindow, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, sessionName+":"+windowName, paneTarget)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: 既存ウィンドウで既存paneが見つかった場合、そのpaneを選択", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-149"
		paneTitle := "plan-phase"
		isNewWindow := false

		mockExec := mocks.NewMockCommandExecutor()

		// List panes - return pane with matching title
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase\n", nil)

		// Select existing pane
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName + ".0"}).Return("", nil)

		// Act
		paneTarget, err := tmux.SelectOrCreatePaneForPhaseWithNewWindowFlag(sessionName, windowName, paneTitle, isNewWindow, mockExec)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, sessionName+":"+windowName+".0", paneTarget)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: 空のセッション名", func(t *testing.T) {
		// Arrange
		sessionName := ""
		windowName := "issue-149"
		paneTitle := "plan-phase"
		isNewWindow := false
		mockExec := mocks.NewMockCommandExecutor()

		// Act
		paneTarget, err := tmux.SelectOrCreatePaneForPhaseWithNewWindowFlag(sessionName, windowName, paneTitle, isNewWindow, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "", paneTarget)
		assert.Contains(t, err.Error(), "session name cannot be empty")
	})

	t.Run("異常系: 空のウィンドウ名", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := ""
		paneTitle := "plan-phase"
		isNewWindow := false
		mockExec := mocks.NewMockCommandExecutor()

		// Act
		paneTarget, err := tmux.SelectOrCreatePaneForPhaseWithNewWindowFlag(sessionName, windowName, paneTitle, isNewWindow, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "", paneTarget)
		assert.Contains(t, err.Error(), "window name cannot be empty")
	})

	t.Run("異常系: 空のpaneタイトル", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-149"
		paneTitle := ""
		isNewWindow := false
		mockExec := mocks.NewMockCommandExecutor()

		// Act
		paneTarget, err := tmux.SelectOrCreatePaneForPhaseWithNewWindowFlag(sessionName, windowName, paneTitle, isNewWindow, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, "", paneTarget)
		assert.Contains(t, err.Error(), "pane title cannot be empty")
	})
}
