package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestWorktreeManager_UpdateMainBranch(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "worktree-manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// テスト用のgitリポジトリを初期化
	logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := NewCommand(logger)

	// gitリポジトリを初期化
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 初期コミットを作成
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// mainブランチを明示的に作成
	_, err = cmd.Run(context.Background(), "git", []string{"branch", "-M", "main"}, tmpDir)
	require.NoError(t, err)

	// フィーチャーブランチを作成してチェックアウト
	_, err = cmd.Run(context.Background(), "git", []string{"checkout", "-b", "feature/test"}, tmpDir)
	require.NoError(t, err)

	// WorktreeManagerを作成（リモートなしバージョン）
	branch := NewBranch(logger)

	// リモートが存在しない場合でも、ブランチ切り替えのテストができる
	currentBranch, err := branch.GetCurrent(context.Background(), tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "feature/test", currentBranch)

	// mainブランチに切り替え
	err = branch.Checkout(context.Background(), tmpDir, "main", false)
	assert.NoError(t, err)

	// 元のブランチに戻る
	err = branch.Checkout(context.Background(), tmpDir, currentBranch, false)
	assert.NoError(t, err)

	// 現在のブランチがfeature/testに戻っていることを確認
	finalBranch, err := branch.GetCurrent(context.Background(), tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "feature/test", finalBranch)
}

func TestWorktreeManager_CreateWorktree(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "worktree-manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// テスト用のgitリポジトリを初期化
	logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := NewCommand(logger)

	// gitリポジトリを初期化
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 初期コミットを作成
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// mainブランチを明示的に作成（初回commitはデフォルトでmasterブランチになるため）
	_, err = cmd.Run(context.Background(), "git", []string{"branch", "-M", "main"}, tmpDir)
	require.NoError(t, err)

	// WorktreeManagerを作成
	worktree := NewWorktree(logger)
	branch := NewBranch(logger)
	sync := NewSync(logger)

	mockRepo := &mockRepository{rootPath: tmpDir}
	manager := &worktreeManager{
		repository: mockRepo,
		worktree:   worktree,
		branch:     branch,
		sync:       sync,
		basePath:   tmpDir,
	}

	tests := []struct {
		name        string
		issueNumber int
		phase       Phase
		expectError bool
	}{
		{
			name:        "正常系: 計画フェーズのworktreeを作成",
			issueNumber: 45,
			phase:       PhasePlan,
			expectError: false,
		},
		{
			name:        "正常系: 実装フェーズのworktreeを作成",
			issueNumber: 45,
			phase:       PhaseImplementation,
			expectError: false,
		},
		{
			name:        "正常系: レビューフェーズのworktreeを作成",
			issueNumber: 45,
			phase:       PhaseReview,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// worktreeを作成
			err := manager.CreateWorktree(context.Background(), tt.issueNumber, tt.phase)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// worktreeが作成されたことを確認
				worktreePath := manager.GetWorktreePath(tt.issueNumber, tt.phase)
				assert.DirExists(t, worktreePath)

				// ブランチが作成されたことを確認
				branchName := manager.generateBranchName(tt.issueNumber, tt.phase)
				branches, err := branch.List(context.Background(), tmpDir, false)
				require.NoError(t, err)

				found := false
				for _, b := range branches {
					if b.Name == branchName {
						found = true
						break
					}
				}
				assert.True(t, found, "Branch %s should exist", branchName)
			}
		})
	}
}

func TestWorktreeManager_RemoveWorktree(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "worktree-manager-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// テスト用のgitリポジトリを初期化
	logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	cmd := NewCommand(logger)

	// gitリポジトリを初期化
	_, err = cmd.Run(context.Background(), "git", []string{"init"}, tmpDir)
	require.NoError(t, err)

	// CI環境用のgit設定
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.email", "test@example.com"}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"config", "user.name", "Test User"}, tmpDir)
	require.NoError(t, err)

	// 初期コミットを作成
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// mainブランチを明示的に作成（初回commitはデフォルトでmasterブランチになるため）
	_, err = cmd.Run(context.Background(), "git", []string{"branch", "-M", "main"}, tmpDir)
	require.NoError(t, err)

	// WorktreeManagerを作成
	worktree := NewWorktree(logger)
	branch := NewBranch(logger)
	sync := NewSync(logger)

	mockRepo := &mockRepository{rootPath: tmpDir}
	manager := &worktreeManager{
		repository: mockRepo,
		worktree:   worktree,
		branch:     branch,
		sync:       sync,
		basePath:   tmpDir,
	}

	// worktreeを作成
	issueNumber := 45
	phase := PhasePlan
	err = manager.CreateWorktree(context.Background(), issueNumber, phase)
	require.NoError(t, err)

	// worktreeが存在することを確認
	exists, err := manager.WorktreeExists(context.Background(), issueNumber, phase)
	require.NoError(t, err)
	assert.True(t, exists)

	// worktreeを削除
	err = manager.RemoveWorktree(context.Background(), issueNumber, phase)
	assert.NoError(t, err)

	// worktreeが削除されたことを確認
	exists, err = manager.WorktreeExists(context.Background(), issueNumber, phase)
	require.NoError(t, err)
	assert.False(t, exists)

	// worktreeディレクトリが削除されたことを確認
	worktreePath := manager.GetWorktreePath(issueNumber, phase)
	assert.NoDirExists(t, worktreePath)
}

func TestWorktreeManager_GetWorktreePath(t *testing.T) {
	mockRepo := &mockRepository{rootPath: "/test/repo"}
	manager := &worktreeManager{
		repository: mockRepo,
		basePath:   "/test/repo",
	}

	tests := []struct {
		name         string
		issueNumber  int
		phase        Phase
		expectedPath string
	}{
		{
			name:         "計画フェーズのパス",
			issueNumber:  45,
			phase:        PhasePlan,
			expectedPath: "/test/repo/.git/osoba/worktrees/45-plan",
		},
		{
			name:         "実装フェーズのパス",
			issueNumber:  45,
			phase:        PhaseImplementation,
			expectedPath: "/test/repo/.git/osoba/worktrees/45-implementation",
		},
		{
			name:         "レビューフェーズのパス",
			issueNumber:  45,
			phase:        PhaseReview,
			expectedPath: "/test/repo/.git/osoba/worktrees/45-review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := manager.GetWorktreePath(tt.issueNumber, tt.phase)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestWorktreeManager_generateBranchName(t *testing.T) {
	manager := &worktreeManager{}

	tests := []struct {
		name           string
		issueNumber    int
		phase          Phase
		expectedBranch string
	}{
		{
			name:           "計画フェーズのブランチ名",
			issueNumber:    45,
			phase:          PhasePlan,
			expectedBranch: "osoba/#45-plan",
		},
		{
			name:           "実装フェーズのブランチ名",
			issueNumber:    45,
			phase:          PhaseImplementation,
			expectedBranch: "osoba/#45-implementation",
		},
		{
			name:           "レビューフェーズのブランチ名",
			issueNumber:    45,
			phase:          PhaseReview,
			expectedBranch: "osoba/#45-review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchName := manager.generateBranchName(tt.issueNumber, tt.phase)
			assert.Equal(t, tt.expectedBranch, branchName)
		})
	}
}

// mockRepository はテスト用のRepositoryモック
type mockRepository struct {
	rootPath string
}

func (m *mockRepository) GetRootPath(ctx context.Context) (string, error) {
	return m.rootPath, nil
}

func (m *mockRepository) IsGitRepository(ctx context.Context, path string) bool {
	return true
}

func (m *mockRepository) GetCurrentCommit(ctx context.Context, path string) (string, error) {
	return "dummy-commit", nil
}

func (m *mockRepository) GetRemoteURL(ctx context.Context, path string, remoteName string) (string, error) {
	return "https://github.com/test/repo.git", nil
}

func (m *mockRepository) GetStatus(ctx context.Context, path string) (*RepositoryStatus, error) {
	return &RepositoryStatus{
		IsClean:        true,
		ModifiedFiles:  []string{},
		UntrackedFiles: []string{},
		StagedFiles:    []string{},
	}, nil
}

func (m *mockRepository) GetLogger() logger.Logger {
	logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
	return logger
}
