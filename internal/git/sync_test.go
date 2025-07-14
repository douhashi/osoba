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

func TestSync_Fetch(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-sync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
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

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	sync := &Sync{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// fetchを実行（リモートがないのでエラーになるが、ログ出力を確認）
	err = sync.Fetch(context.Background(), tmpDir, "origin", false)
	// リモートが設定されていないのでエラーになるはず
	assert.Error(t, err)

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Fetching from remote",
		"Failed to fetch from remote",
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

func TestSync_Pull(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-sync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
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

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	sync := &Sync{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// pullを実行（リモートがないのでエラーになるが、ログ出力を確認）
	err = sync.Pull(context.Background(), tmpDir, "origin", "main", false)
	// リモートが設定されていないのでエラーになるはず
	assert.Error(t, err)

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Pulling from remote",
		"Failed to pull from remote",
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

func TestSync_Push(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-sync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
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

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	sync := &Sync{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// pushを実行（リモートがないのでエラーになるが、ログ出力を確認）
	err = sync.Push(context.Background(), tmpDir, "origin", "main", false, false)
	// リモートが設定されていないのでエラーになるはず
	assert.Error(t, err)

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Pushing to remote",
		"Failed to push to remote",
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

func TestSync_GetRemotes(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-sync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// テスト用のリモートを追加
	_, err = cmd.Run(context.Background(), "git", []string{"remote", "add", "origin", "https://github.com/test/test.git"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"remote", "add", "upstream", "https://github.com/upstream/test.git"}, tmpDir)
	require.NoError(t, err)

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	sync := &Sync{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// リモート一覧を取得
	remotes, err := sync.GetRemotes(context.Background(), tmpDir)
	assert.NoError(t, err)
	assert.Len(t, remotes, 2)

	// リモート情報が正しく取得できているか確認
	expectedRemotes := map[string]string{
		"origin":   "https://github.com/test/test.git",
		"upstream": "https://github.com/upstream/test.git",
	}
	for _, remote := range remotes {
		expectedURL, ok := expectedRemotes[remote.Name]
		assert.True(t, ok, "Unexpected remote: %s", remote.Name)
		assert.Equal(t, expectedURL, remote.URL)
	}

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsgs := []string{
		"Listing git remotes",
		"Git remotes listed successfully",
	}
	for _, expectedMsg := range expectedMsgs {
		found := false
		for _, entry := range entries {
			if strings.Contains(entry.Message, expectedMsg) {
				found = true
				// リモート数がログに記録されていることを確認
				if strings.Contains(entry.Message, "listed successfully") {
					fields := getFieldsAsMap(entry.Context)
					if count, ok := fields["count"].(float64); ok {
						assert.Equal(t, float64(2), count)
					}
				}
				break
			}
		}
		assert.True(t, found, "Expected log message not found: %s", expectedMsg)
	}
}

func TestSync_GetStatus(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "git-sync-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// gitリポジトリを初期化
	cmd := NewCommand(&testLoggerImpl{sugar: zap.NewNop().Sugar()})
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// ログ出力をキャプチャ
	core, recorded := observer.New(zap.InfoLevel)
	testLogger := &testLoggerImpl{sugar: zap.New(core).Sugar()}

	sync := &Sync{
		logger:  testLogger,
		command: NewCommand(testLogger),
	}

	// ステータスを取得
	status, err := sync.GetStatus(context.Background(), tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.IsClean) // 新規リポジトリなのでクリーンなはず

	// 新しいファイルを追加
	newFile := filepath.Join(tmpDir, "new.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	require.NoError(t, err)

	// 再度ステータスを取得
	status, err = sync.GetStatus(context.Background(), tmpDir)
	assert.NoError(t, err)
	assert.False(t, status.IsClean) // 未追跡ファイルがあるのでクリーンではない
	assert.Len(t, status.UntrackedFiles, 1)
	assert.Contains(t, status.UntrackedFiles, "new.txt")

	// ログメッセージの検証
	entries := recorded.All()
	expectedMsg := "Getting git status"
	found := false
	for _, entry := range entries {
		if strings.Contains(entry.Message, expectedMsg) {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected log message not found: %s", expectedMsg)
}
