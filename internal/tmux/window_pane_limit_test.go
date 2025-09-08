package tmux_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
)

func TestSelectOrCreatePaneForPhaseWithPaneLimit(t *testing.T) {
	t.Run("正常系: ペイン数が上限以下の場合は通常通り新しいペインを作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-123"
		paneTitle := "new-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// 既存のpaneが2個存在
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase", nil)
		// 新しいpaneを作成（ペイン数制限に引っかからない）
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "33"}).Return("", nil)
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: ペイン数が上限を超える場合は最古のペインを削除してから新しいペインを作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-123"
		paneTitle := "new-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// 既存のpaneが3個存在（上限）
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase\n2:review-phase", nil)
		// ペイン数制限チェック用：アクティブペインを確認
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_active}:#{pane_title}"}).Return("0:0:plan-phase\n1:0:implement-phase\n2:1:review-phase", nil)
		// 最古のペイン（インデックス0）を削除
		mockExec.On("Execute", "tmux", []string{"kill-pane", "-t", sessionName + ":" + windowName + ".0"}).Return("", nil)
		// 新しいpaneを作成
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "33"}).Return("", nil)
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("正常系: アクティブペインは削除対象から除外される", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-123"
		paneTitle := "new-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// 既存のpaneが3個存在し、インデックス0がアクティブ
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase\n2:review-phase", nil)
		// アクティブペイン確認：インデックス0がアクティブ
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_active}:#{pane_title}"}).Return("0:1:plan-phase\n1:0:implement-phase\n2:0:review-phase", nil)
		// インデックス0はアクティブなので、次に古いインデックス1を削除
		mockExec.On("Execute", "tmux", []string{"kill-pane", "-t", sessionName + ":" + windowName + ".1"}).Return("", nil)
		// 新しいpaneを作成
		mockExec.On("Execute", "tmux", []string{"split-window", "-t", sessionName + ":" + windowName, "-h", "-p", "33"}).Return("", nil)
		mockExec.On("Execute", "tmux", []string{"select-pane", "-t", sessionName + ":" + windowName, "-T", paneTitle}).Return("", nil)

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.NoError(t, err)
		mockExec.AssertExpectations(t)
	})

	t.Run("異常系: ペイン削除に失敗した場合はエラーを返す", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		windowName := "issue-123"
		paneTitle := "new-phase"

		mockExec := mocks.NewMockTmuxCommandExecutor()
		// 既存のpaneが3個存在
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_title}"}).Return("0:plan-phase\n1:implement-phase\n2:review-phase", nil)
		mockExec.On("Execute", "tmux", []string{"list-panes", "-t", sessionName + ":" + windowName, "-F", "#{pane_index}:#{pane_active}:#{pane_title}"}).Return("0:0:plan-phase\n1:0:implement-phase\n2:1:review-phase", nil)
		// ペイン削除に失敗
		mockExec.On("Execute", "tmux", []string{"kill-pane", "-t", sessionName + ":" + windowName + ".0"}).Return("", errors.New("failed to kill pane"))

		// Act
		err := tmux.SelectOrCreatePaneForPhaseWithExecutor(sessionName, windowName, paneTitle, mockExec)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to kill pane")
		mockExec.AssertExpectations(t)
	})
}