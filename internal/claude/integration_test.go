package claude

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaudeExecutor_LoggerIntegration(t *testing.T) {
	t.Run("ロガーを使用したClaude実行フロー", func(t *testing.T) {
		ml := newMockLogger()
		executor := NewClaudeExecutorWithLogger(ml)
		assert.NotNil(t, executor)

		config := &PhaseConfig{
			Args:   []string{"--test"},
			Prompt: "/osoba:test {{issue-number}}",
		}
		vars := &TemplateVariables{
			IssueNumber: 123,
		}
		workdir := "/test/dir"

		// Claudeコマンドが存在しない環境でのテスト
		// エラーが発生することを想定
		err := executor.ExecuteInWorktree(context.Background(), config, vars, workdir)
		assert.Error(t, err)

		// ログが適切に出力されているか確認
		// 1. エラーログがあるはず
		assert.Greater(t, len(ml.errorCalls), 0, "Should have error logs")

		// 2. エラーログの内容を確認
		foundCommandNotFound := false
		for _, call := range ml.errorCalls {
			if call.msg == "Claude command not found" || call.msg == "Failed to execute Claude" {
				foundCommandNotFound = true
				break
			}
		}
		assert.True(t, foundCommandNotFound, "Should log error related to Claude execution")
	})

	t.Run("tmuxでのロガー使用", func(t *testing.T) {
		ml := newMockLogger()
		executor := NewClaudeExecutorWithLogger(ml)
		assert.NotNil(t, executor)

		config := &PhaseConfig{
			Args:   []string{"--test"},
			Prompt: "/osoba:test {{issue-number}}",
		}
		vars := &TemplateVariables{
			IssueNumber: 456,
		}
		sessionName := "test-session"
		windowName := "test-window"
		workdir := "/test/dir"

		// Claudeコマンドが存在しない環境でのテスト
		err := executor.ExecuteInTmux(context.Background(), config, vars, sessionName, windowName, workdir)
		assert.Error(t, err)

		// エラーログを確認
		assert.Greater(t, len(ml.errorCalls), 0, "Should have error logs")
	})

	t.Run("構造化ログのコンテキスト情報", func(t *testing.T) {
		ml := newMockLogger()
		executor := &DefaultClaudeExecutor{
			logger: ml,
		}

		// CheckClaudeExistsのテスト
		_ = executor.CheckClaudeExists()

		// エラーログを確認
		if len(ml.errorCalls) > 0 {
			// エラーログが含まれていれば、適切なコンテキストがあることを確認
			errorCall := ml.errorCalls[0]
			assert.Equal(t, "Claude command not found", errorCall.msg)

			// key-valueペアが含まれているか確認
			assert.Greater(t, len(errorCall.keysAndValues), 0, "Should have context in error log")
		}
	})

	t.Run("機密情報のマスキング確認", func(t *testing.T) {
		ml := newMockLogger()
		executor := &DefaultClaudeExecutor{
			logger: ml,
		}

		config := &PhaseConfig{
			Args:   []string{"--test"},
			Prompt: "Execute with token ghp_1234567890abcdefghijklmnopqrstuvwxyz",
		}
		vars := &TemplateVariables{
			IssueNumber: 789,
		}
		workdir := "/test/dir"

		// 実行（エラーは無視）
		_ = executor.ExecuteInWorktree(context.Background(), config, vars, workdir)

		// デバッグログを確認
		for _, call := range ml.debugCalls {
			for i := 0; i < len(call.keysAndValues); i += 2 {
				if call.keysAndValues[i] == "prompt" {
					promptValue := call.keysAndValues[i+1].(string)
					// トークンがマスクされているか確認
					assert.Contains(t, promptValue, "[GITHUB_TOKEN]", "Token should be masked")
					assert.NotContains(t, promptValue, "ghp_1234567890abcdefghijklmnopqrstuvwxyz", "Raw token should not appear")
				}
			}
		}
	})
}

func TestClaudeExecutor_LoggerLifecycle(t *testing.T) {
	t.Run("複数フェーズでのログ出力", func(t *testing.T) {
		ml := newMockLogger()
		executor := NewClaudeExecutorWithLogger(ml)

		// 異なるフェーズの設定
		phases := []struct {
			name   string
			config *PhaseConfig
			vars   *TemplateVariables
		}{
			{
				name: "plan",
				config: &PhaseConfig{
					Args:   []string{"--dangerously-skip-permissions"},
					Prompt: "/osoba:plan {{issue-number}}",
				},
				vars: &TemplateVariables{IssueNumber: 100},
			},
			{
				name: "implement",
				config: &PhaseConfig{
					Args:   []string{"--dangerously-skip-permissions"},
					Prompt: "/osoba:implement {{issue-number}}",
				},
				vars: &TemplateVariables{IssueNumber: 100},
			},
			{
				name: "review",
				config: &PhaseConfig{
					Args:   []string{"--read-only"},
					Prompt: "/osoba:review {{issue-number}}",
				},
				vars: &TemplateVariables{IssueNumber: 100},
			},
		}

		// 各フェーズを実行
		for _, phase := range phases {
			_ = executor.ExecuteInWorktree(context.Background(), phase.config, phase.vars, "/test/dir")
		}

		// エラーログが3つ（各フェーズで1つずつ）あることを確認
		assert.GreaterOrEqual(t, len(ml.errorCalls), 3, "Should have error logs for each phase")
	})
}
