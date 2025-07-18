package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestWorktree_Create(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-worktree-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	testLogger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := NewCommand(testLogger)
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 初期コミットを作成
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)

	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	tests := []struct {
		name          string
		path          string
		branch        string
		expectError   bool
		expectLogMsgs []string
	}{
		{
			name:        "正常系: 新しいworktreeを作成",
			path:        filepath.Join(tmpDir, "worktree1"),
			branch:      "feature/test1",
			expectError: false,
			expectLogMsgs: []string{
				"Creating git worktree",
				"Git worktree created successfully",
			},
		},
		{
			name:        "正常系: 別のworktreeを作成",
			path:        filepath.Join(tmpDir, "worktree2"),
			branch:      "feature/test2",
			expectError: false,
			expectLogMsgs: []string{
				"Creating git worktree",
				"Git worktree created successfully",
			},
		},
		{
			name:        "異常系: 既存のパスにworktreeを作成",
			path:        filepath.Join(tmpDir, "worktree1"),
			branch:      "feature/test3",
			expectError: true,
			expectLogMsgs: []string{
				"Creating git worktree",
				"Failed to create git worktree",
			},
		},
		{
			name:        "異常系: 既存のブランチ名でworktreeを作成",
			path:        filepath.Join(tmpDir, "worktree3"),
			branch:      "feature/test1",
			expectError: true,
			expectLogMsgs: []string{
				"Creating git worktree",
				"Failed to create git worktree",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ログ出力をキャプチャ
			testLogger, recorded := helpers.NewObservableLogger(zapcore.InfoLevel)

			wt := &Worktree{
				logger:  testLogger,
				command: NewCommand(testLogger),
			}

			// ブランチが存在しない場合は作成（-bフラグを削除したため）
			if !tt.expectError || tt.name != "異常系: 既存のブランチ名でworktreeを作成" {
				// ブランチを作成
				_, err := cmd.Run(context.Background(), "git", []string{"branch", tt.branch}, tmpDir)
				// エラーは無視（既存のブランチの場合もあるため）
				_ = err
			}

			// worktree作成を実行
			err := wt.Create(context.Background(), tmpDir, tt.path, tt.branch)

			// エラーチェック
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// worktreeが作成されたことを確認
				assert.DirExists(t, tt.path)
			}

			// ログメッセージの検証
			entries := recorded.All()
			for _, expectedMsg := range tt.expectLogMsgs {
				found := false
				for _, entry := range entries {
					if strings.Contains(entry.Message, expectedMsg) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected log message not found: %s", expectedMsg)
			}
		})
	}
}

func TestWorktree_Remove(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-worktree-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化して準備
	testLogger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := NewCommand(testLogger)
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 初期コミット
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// テスト用のworktreeを作成
	worktreePath := filepath.Join(tmpDir, "test-worktree")
	_, err = cmd.Run(context.Background(), "git", []string{"worktree", "add", worktreePath, "-b", "test-branch"}, tmpDir)
	require.NoError(t, err)

	// ログ出力をキャプチャ
	testLogger, recorded := helpers.NewObservableLogger(zapcore.InfoLevel)

	wt := &Worktree{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// worktree削除を実行
	err = wt.Remove(context.Background(), tmpDir, worktreePath)
	assert.NoError(t, err)

	// worktreeが削除されたことを確認
	assert.NoDirExists(t, worktreePath)

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Removing git worktree",
		"Git worktree removed successfully",
	}
	for _, expectedMsg := range expectedMsgs {
		found := false
		for _, entry := range entries {
			if strings.Contains(entry.Message, expectedMsg) {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected log message not found: %s", expectedMsg)
	}
}

func TestWorktree_List(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-worktree-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	testLogger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := NewCommand(testLogger)
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 初期コミット
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// 複数のworktreeを作成
	worktrees := []struct {
		path   string
		branch string
	}{
		{filepath.Join(tmpDir, "worktree1"), "feature/test1"},
		{filepath.Join(tmpDir, "worktree2"), "feature/test2"},
	}

	for _, wt := range worktrees {
		_, err = cmd.Run(context.Background(), "git", []string{"worktree", "add", wt.path, "-b", wt.branch}, tmpDir)
		require.NoError(t, err)
	}

	// ログ出力をキャプチャ
	testLogger, recorded := helpers.NewObservableLogger(zapcore.InfoLevel)

	wt := &Worktree{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// worktree一覧を取得
	list, err := wt.List(context.Background(), tmpDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, list)

	// メインのworktreeと追加した2つのworktreeがあることを確認
	assert.GreaterOrEqual(t, len(list), 3)

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Listing git worktrees",
		"Git worktrees listed successfully",
	}
	for _, expectedMsg := range expectedMsgs {
		found := false
		for _, entry := range entries {
			if strings.Contains(entry.Message, expectedMsg) {
				found = true
				// worktree数がログに記録されていることを確認
				if strings.Contains(entry.Message, "listed successfully") {
					fields := helpers.GetZapFieldsAsMap(entry.Context)
					if count, ok := fields["count"].(float64); ok {
						assert.Equal(t, float64(len(list)), count)
					}
				}
				break
			}
		}
		assert.True(t, found, "Expected log message not found: %s", expectedMsg)
	}

	// 各worktreeの情報が正しく取得できているか確認
	for _, wtInfo := range list {
		assert.NotEmpty(t, wtInfo.Path)
		assert.NotEmpty(t, wtInfo.Branch)
		assert.NotEmpty(t, wtInfo.Commit)
	}
}
