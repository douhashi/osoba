package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestCommand_Run(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		args        []string
		workDir     string
		expectError bool
		expectLog   []string
	}{
		{
			name:        "正常系: git versionコマンドの実行",
			cmd:         "git",
			args:        []string{"version"},
			expectError: false,
			expectLog: []string{
				"Executing git command",
				"Git command completed successfully",
			},
		},
		{
			name:        "正常系: 作業ディレクトリ指定でのgit statusコマンド",
			cmd:         "git",
			args:        []string{"status", "--porcelain"},
			workDir:     ".",
			expectError: false,
			expectLog: []string{
				"Executing git command",
				"Git command completed successfully",
			},
		},
		{
			name:        "異常系: 存在しないgitサブコマンド",
			cmd:         "git",
			args:        []string{"nonexistent-command"},
			expectError: true,
			expectLog: []string{
				"Executing git command",
				"Git command failed",
			},
		},
		{
			name:        "異常系: 存在しないディレクトリでの実行",
			cmd:         "git",
			args:        []string{"status"},
			workDir:     "/nonexistent/directory",
			expectError: true,
			expectLog: []string{
				"Executing git command",
				"Git command failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ログ出力をキャプチャするための設定（DEBUGレベル）
			testLogger, recorded := helpers.NewObservableLogger(zapcore.DebugLevel)

			cmd := &Command{
				logger: testLogger,
			}

			// コマンド実行
			output, err := cmd.Run(context.Background(), tt.cmd, tt.args, tt.workDir)

			// エラーチェック
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// git versionコマンドの場合のみ出力があることを確認
				if tt.name == "正常系: git versionコマンドの実行" {
					assert.NotEmpty(t, output)
				}
			}

			// ログ出力の検証
			entries := recorded.All()
			for _, expectedLog := range tt.expectLog {
				found := false
				for _, entry := range entries {
					if strings.Contains(entry.Message, expectedLog) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected log message not found: %s", expectedLog)
			}

			// 構造化フィールドの検証
			if len(entries) > 0 {
				// コマンド実行開始ログの検証
				startEntry := entries[0]
				assert.Equal(t, "Executing git command", startEntry.Message)

				// フィールドの存在確認（commandフィールドのみ検証）
				fields := helpers.GetZapFieldsAsMap(startEntry.Context)
				if cmdField, ok := fields["command"]; ok {
					assert.Equal(t, tt.cmd, cmdField)
				}

				// workDirフィールドの検証
				if tt.workDir != "" {
					if workDirField, ok := fields["workDir"]; ok {
						assert.Equal(t, tt.workDir, workDirField)
					}
				}
			}
		})
	}
}

func TestCommand_RunWithTimeout(t *testing.T) {
	// ログ出力をキャプチャ
	testLogger, recorded := helpers.NewObservableLogger(zapcore.InfoLevel)

	cmd := &Command{
		logger: testLogger,
	}

	// タイムアウトを設定した短いコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// 長時間実行されるコマンドを実行
	_, err := cmd.Run(ctx, "sleep", []string{"1"}, "")

	// タイムアウトエラーを確認（signal: killedまたはcontext deadline exceeded）
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "signal: killed") || strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected timeout error, got: %v", err)

	// ログにタイムアウトが記録されていることを確認
	entries := recorded.All()
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, "Git command failed") {
			fields := helpers.GetZapFieldsAsMap(entry.Context)
			if errField, ok := fields["error"].(string); ok &&
				(strings.Contains(errField, "signal: killed") || strings.Contains(errField, "context deadline exceeded")) {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Timeout error not found in logs")
}

func TestCommand_RunWithLargeOutput(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	testLogger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := &Command{
		logger: testLogger,
	}

	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 大量のファイルを作成
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(filename, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	// ログ出力をキャプチャ
	testLogger, recorded := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd.logger = testLogger

	// git statusを実行（大量の出力が期待される）
	output, err := cmd.Run(context.Background(), "git", []string{"status", "--porcelain"}, tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, output)

	// ログに出力が要約されていることを確認
	entries := recorded.All()
	for _, entry := range entries {
		if entry.Message == "Git command completed successfully" {
			fields := helpers.GetZapFieldsAsMap(entry.Context)
			if outputField, ok := fields["output"].(string); ok {
				// 出力が適切に記録されていることを確認
				assert.True(t, len(outputField) > 0)
			}
		}
	}
}
