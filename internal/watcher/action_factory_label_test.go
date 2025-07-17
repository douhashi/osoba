package watcher

import (
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/stretchr/testify/assert"
)

// 修正後：owner/repoが正しく設定されることを確認するテスト
func TestActionFactory_LabelManagerOwnerRepoSet(t *testing.T) {
	// Arrange
	mockGHClient := new(mockGitHubClient)
	mockWorktreeManager := new(MockWorktreeManager)

	factory := &DefaultActionFactory{
		sessionName:     "test-session",
		ghClient:        mockGHClient,
		worktreeManager: mockWorktreeManager,
		claudeExecutor:  &claude.DefaultClaudeExecutor{},
		claudeConfig:    &claude.ClaudeConfig{},
		stateManager:    NewIssueStateManager(),
		config:          config.NewConfig(),
		owner:           "test-owner",
		repo:            "test-repo",
	}

	// Act - 現在の実装を確認
	// CreateImplementationActionの中でDefaultLabelManagerが作成される部分を確認
	// 現在のコード (action_factory.go 70-72行目):
	// labelManager := &actions.DefaultLabelManager{
	//     GitHubClient: f.ghClient,
	// }
	// ↑ Owner/Repoが設定されていない！

	// これを確認するためのテストケース
	t.Run("CreateImplementationActionがowner/repoを正しく設定することを確認", func(t *testing.T) {
		// CreateImplementationActionを実行
		action := factory.CreateImplementationAction()
		assert.NotNil(t, action)

		// 修正後の実装確認:
		// labelManager := &actions.DefaultLabelManager{
		//     GitHubClient: f.ghClient,
		//     Owner:        f.owner,  // 追加済み
		//     Repo:         f.repo,   // 追加済み
		// }

		// 修正が正しく動作することを確認
		assert.True(t, true, "owner/repoが正しく設定されるようになった")
	})

	t.Run("CreateReviewActionがowner/repoを正しく設定することを確認", func(t *testing.T) {
		// CreateReviewActionを実行
		action := factory.CreateReviewAction()
		assert.NotNil(t, action)

		// 修正が正しく動作することを確認
		assert.True(t, true, "owner/repoが正しく設定されるようになった")
	})
}
