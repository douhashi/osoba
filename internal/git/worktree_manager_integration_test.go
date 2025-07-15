package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWorktreeManager_Integration(t *testing.T) {
	// 統合テストはCI環境でスキップ
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI")
	}

	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "worktree-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// ロガーを作成
	logger := &testLoggerImpl{sugar: zap.NewNop().Sugar()}

	// コンポーネントを初期化
	repository := NewRepository(logger)
	worktree := NewWorktree(logger)
	branch := NewBranch(logger)
	sync := NewSync(logger)
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
	testFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repository\n"), 0644)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"add", "."}, tmpDir)
	require.NoError(t, err)
	_, err = cmd.Run(context.Background(), "git", []string{"commit", "-m", "initial commit"}, tmpDir)
	require.NoError(t, err)

	// mainブランチを明示的に作成
	_, err = cmd.Run(context.Background(), "git", []string{"branch", "-M", "main"}, tmpDir)
	require.NoError(t, err)

	// WorktreeManagerを作成（実際のRepositoryを使用）
	manager, err := NewWorktreeManager(repository, worktree, branch, sync)
	// リポジトリが見つからない場合は、tmpDirを使用するモックを作成
	if err != nil {
		mockRepo := &mockRepository{rootPath: tmpDir}
		manager = &worktreeManager{
			repository: mockRepo,
			worktree:   worktree,
			branch:     branch,
			sync:       sync,
			basePath:   tmpDir,
		}
	}

	ctx := context.Background()

	// 統合テストのシナリオ
	t.Run("完全なワークフロー", func(t *testing.T) {
		issueNumber := 45

		// 1. 計画フェーズのworktreeを作成
		t.Run("計画フェーズ", func(t *testing.T) {
			err := manager.CreateWorktree(ctx, issueNumber, PhasePlan)
			assert.NoError(t, err)

			// worktreeが作成されたことを確認
			exists, err := manager.WorktreeExists(ctx, issueNumber, PhasePlan)
			require.NoError(t, err)
			assert.True(t, exists)

			// worktreeパスを取得
			worktreePath := manager.GetWorktreePath(issueNumber, PhasePlan)
			assert.DirExists(t, worktreePath)

			// ブランチが作成されたことを確認
			branchName := fmt.Sprintf("osoba/#%d-%s", issueNumber, PhasePlan)
			assert.True(t, branch.Exists(ctx, tmpDir, branchName))
		})

		// 2. 実装フェーズのworktreeを作成
		t.Run("実装フェーズ", func(t *testing.T) {
			err := manager.CreateWorktree(ctx, issueNumber, PhaseImplementation)
			assert.NoError(t, err)

			// worktreeが作成されたことを確認
			exists, err := manager.WorktreeExists(ctx, issueNumber, PhaseImplementation)
			require.NoError(t, err)
			assert.True(t, exists)

			// worktreeパスを取得
			worktreePath := manager.GetWorktreePath(issueNumber, PhaseImplementation)
			assert.DirExists(t, worktreePath)

			// ブランチが作成されたことを確認
			branchName := fmt.Sprintf("osoba/#%d-%s", issueNumber, PhaseImplementation)
			assert.True(t, branch.Exists(ctx, tmpDir, branchName))
		})

		// 3. レビューフェーズのworktreeを作成
		t.Run("レビューフェーズ", func(t *testing.T) {
			err := manager.CreateWorktree(ctx, issueNumber, PhaseReview)
			assert.NoError(t, err)

			// worktreeが作成されたことを確認
			exists, err := manager.WorktreeExists(ctx, issueNumber, PhaseReview)
			require.NoError(t, err)
			assert.True(t, exists)

			// worktreeパスを取得
			worktreePath := manager.GetWorktreePath(issueNumber, PhaseReview)
			assert.DirExists(t, worktreePath)

			// ブランチが作成されたことを確認
			branchName := fmt.Sprintf("osoba/#%d-%s", issueNumber, PhaseReview)
			assert.True(t, branch.Exists(ctx, tmpDir, branchName))
		})

		// 4. worktree一覧を確認
		t.Run("worktree一覧確認", func(t *testing.T) {
			worktrees, err := worktree.List(ctx, tmpDir)
			require.NoError(t, err)

			// メインworktree + 3つのフェーズworktreeがあることを確認
			assert.GreaterOrEqual(t, len(worktrees), 4)
		})

		// 5. worktreeの削除
		t.Run("worktree削除", func(t *testing.T) {
			// 計画フェーズのworktreeを削除
			err := manager.RemoveWorktree(ctx, issueNumber, PhasePlan)
			assert.NoError(t, err)

			// worktreeが削除されたことを確認
			exists, err := manager.WorktreeExists(ctx, issueNumber, PhasePlan)
			require.NoError(t, err)
			assert.False(t, exists)

			// ディレクトリが削除されたことを確認
			worktreePath := manager.GetWorktreePath(issueNumber, PhasePlan)
			assert.NoDirExists(t, worktreePath)

			// ブランチも削除されたことを確認
			branchName := fmt.Sprintf("osoba/#%d-%s", issueNumber, PhasePlan)
			assert.False(t, branch.Exists(ctx, tmpDir, branchName))
		})

		// 6. 再作成（冪等性のテスト）
		t.Run("worktree再作成", func(t *testing.T) {
			// 同じIssueとフェーズで再度作成
			err := manager.CreateWorktree(ctx, issueNumber, PhasePlan)
			assert.NoError(t, err)

			// worktreeが作成されたことを確認
			exists, err := manager.WorktreeExists(ctx, issueNumber, PhasePlan)
			require.NoError(t, err)
			assert.True(t, exists)
		})
	})
}
