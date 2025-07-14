package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestBranch_Create(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-branch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
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
		branchName    string
		baseBranch    string
		expectError   bool
		expectLogMsgs []string
	}{
		{
			name:        "正常系: 新しいブランチを作成",
			branchName:  "feature/test1",
			baseBranch:  "",
			expectError: false,
			expectLogMsgs: []string{
				"Creating git branch",
				"Git branch created successfully",
			},
		},
		{
			name:        "正常系: ベースブランチを指定して作成",
			branchName:  "feature/test2",
			baseBranch:  "main",
			expectError: false,
			expectLogMsgs: []string{
				"Creating git branch",
				"Git branch created successfully",
			},
		},
		{
			name:        "異常系: 既存のブランチ名で作成",
			branchName:  "feature/test1",
			baseBranch:  "",
			expectError: true,
			expectLogMsgs: []string{
				"Creating git branch",
				"Failed to create git branch",
			},
		},
		{
			name:        "異常系: 無効なブランチ名",
			branchName:  "feature//invalid",
			baseBranch:  "",
			expectError: true,
			expectLogMsgs: []string{
				"Creating git branch",
				"Failed to create git branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ログ出力をキャプチャ
			core, recorded := observer.New(zap.InfoLevel)
			testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

			br := &Branch{
				logger:  testLogger,
				command: NewCommand(testLogger),
			}

			// ブランチ作成を実行
			err := br.Create(context.Background(), tmpDir, tt.branchName, tt.baseBranch)

			// エラーチェック
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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

func TestBranch_Checkout(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-branch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// 初期コミット
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// テスト用のブランチを作成
	_, err = cmd.Run(context.Background(), "git", []string{"branch", "test-branch"}, tmpDir)
	require.NoError(t, err)

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	br := &Branch{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// ブランチ切り替えを実行
	err = br.Checkout(context.Background(), tmpDir, "test-branch", false)
	assert.NoError(t, err)

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Checking out git branch",
		"Git branch checked out successfully",
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

	// 現在のブランチが切り替わったことを確認
	current, err := br.GetCurrent(context.Background(), tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, "test-branch", current)
}

func TestBranch_List(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-branch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// 初期コミット
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// 複数のブランチを作成
	branches := []string{"feature/test1", "feature/test2", "bugfix/test1"}
	for _, branch := range branches {
		_, err = cmd.Run(context.Background(), "git", []string{"branch", branch}, tmpDir)
		require.NoError(t, err)
	}

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	br := &Branch{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// ブランチ一覧を取得
	list, err := br.List(context.Background(), tmpDir, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, list)

	// 作成したブランチが全て含まれていることを確認
	for _, expectedBranch := range branches {
		found := false
		for _, branch := range list {
			if branch.Name == expectedBranch {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected branch not found: %s", expectedBranch)
	}

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Listing git branches",
		"Git branches listed successfully",
	}
	for _, expectedMsg := range expectedMsgs {
		found := false
		for _, entry := range entries {
			if strings.Contains(entry.Message, expectedMsg) {
				found = true
				// ブランチ数がログに記録されていることを確認
				if strings.Contains(entry.Message, "listed successfully") {
					fields := getFieldsAsMap(entry.Context)
					if count, ok := fields["count"].(float64); ok {
						assert.Equal(t, float64(len(list)), count)
					}
				}
				break
			}
		}
		assert.True(t, found, "Expected log message not found: %s", expectedMsg)
	}
}

func TestBranch_Delete(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-branch-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// 初期コミット
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// 削除用のブランチを作成
	_, err = cmd.Run(context.Background(), "git", []string{"branch", "to-delete"}, tmpDir)
	require.NoError(t, err)

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	br := &Branch{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// ブランチ削除を実行
	err = br.Delete(context.Background(), tmpDir, "to-delete", false)
	assert.NoError(t, err)

	// ブランチが削除されたことを確認
	list, err := br.List(context.Background(), tmpDir, false)
	assert.NoError(t, err)
	for _, branch := range list {
		assert.NotEqual(t, "to-delete", branch.Name)
	}

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Deleting git branch",
		"Git branch deleted successfully",
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
