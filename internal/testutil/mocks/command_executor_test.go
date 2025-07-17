package mocks_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockCommandExecutor(t *testing.T) {
	t.Run("基本的な動作", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor()

		// モックの設定
		executor.On("Execute", "echo", []string{"hello"}).
			Return("hello\n", nil)

		// 実行
		output, err := executor.Execute("echo", "hello")
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", output)

		executor.AssertExpectations(t)
	})

	t.Run("エラーケース", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor()

		// エラーを返すモック設定
		executor.On("Execute", "false", []string{}).
			Return("", errors.New("command failed"))

		// 実行
		output, err := executor.Execute("false")
		assert.Error(t, err)
		assert.Equal(t, "", output)

		executor.AssertExpectations(t)
	})

	t.Run("デフォルト動作の確認", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor().WithDefaultBehavior()

		t.Run("tmux list-sessions", func(t *testing.T) {
			output, err := executor.Execute("tmux", "list-sessions", "-F", "#{session_name}")
			assert.NoError(t, err)
			assert.Equal(t, "", output)
		})

		t.Run("tmux new-session", func(t *testing.T) {
			output, err := executor.Execute("tmux", "new-session", "-d", "-s", "test-session")
			assert.NoError(t, err)
			assert.Equal(t, "", output)
		})

		t.Run("tmux new-window", func(t *testing.T) {
			output, err := executor.Execute("tmux", "new-window", "-t", "test-session", "-n", "test-window")
			assert.NoError(t, err)
			assert.Equal(t, "", output)
		})

		t.Run("tmux kill-session", func(t *testing.T) {
			output, err := executor.Execute("tmux", "kill-session", "-t", "test-session")
			assert.NoError(t, err)
			assert.Equal(t, "", output)
		})
	})

	t.Run("WithTmuxSessions ヘルパー", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor().
			WithTmuxSessions("session1", "session2", "session3")

		output, err := executor.Execute("tmux", "list-sessions", "-F", "#{session_name}")
		assert.NoError(t, err)
		assert.Contains(t, output, "session1")
		assert.Contains(t, output, "session2")
		assert.Contains(t, output, "session3")
	})

	t.Run("WithTmuxWindows ヘルパー", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor().
			WithTmuxWindows("my-session", "window1", "window2", "window3")

		output, err := executor.Execute("tmux", "list-windows", "-t", "my-session", "-F", "#{window_index}: #{window_name}")
		assert.NoError(t, err)
		assert.Contains(t, output, "0: window1")
		assert.Contains(t, output, "1: window2")
		assert.Contains(t, output, "2: window3")
	})

	t.Run("WithCommandError ヘルパー", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor().
			WithCommandError("git", []string{"push"}, errors.New("permission denied"))

		output, err := executor.Execute("git", "push")
		assert.Error(t, err)
		assert.Equal(t, "permission denied", err.Error())
		assert.Equal(t, "", output)
	})

	t.Run("WithCommandOutput ヘルパー", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor().
			WithCommandOutput("git", []string{"status"}, "On branch main\nnothing to commit")

		output, err := executor.Execute("git", "status")
		assert.NoError(t, err)
		assert.Equal(t, "On branch main\nnothing to commit", output)
	})

	t.Run("引数のマッチング", func(t *testing.T) {
		executor := mocks.NewMockCommandExecutor()

		// 引数の部分一致を使ったモック設定
		executor.On("Execute", "tmux", mock.MatchedBy(func(args []string) bool {
			return len(args) >= 2 && args[0] == "send-keys" && args[1] == "-t"
		})).Return("", nil)

		// 実行
		output, err := executor.Execute("tmux", "send-keys", "-t", "session:window", "Enter")
		assert.NoError(t, err)
		assert.Equal(t, "", output)

		executor.AssertExpectations(t)
	})

	t.Run("MockExitError", func(t *testing.T) {
		exitErr := &mocks.MockExitError{ExitCode: 1}
		assert.Equal(t, "exit status 1", exitErr.Error())
	})
}
