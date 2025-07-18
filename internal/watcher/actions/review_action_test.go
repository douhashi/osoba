package actions

import (
	"context"
	"fmt"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestReviewAction_Execute(t *testing.T) {
	t.Run("正常系: 完全な実行フロー", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateReview, types.IssueStatusProcessing)

		// tmuxウィンドウへの切り替え
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)
		mockTmux.On("SelectOrCreatePaneForPhase", sessionName, fmt.Sprintf("issue-%d", issueNumber), "review-phase").Return(nil)

		// mainブランチの更新
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)

		// worktreeの新規作成
		mockWorktree.On("CreateWorktreeForIssue", ctx, int(issueNumber)).Return(nil)

		// worktreeパスの取得
		workdir := "/tmp/osoba/worktree/28-review"
		mockWorktree.On("GetWorktreePathForIssue", int(issueNumber)).Return(workdir)

		// Claude実行
		phaseConfig := &claude.PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:review {{issue-number}}",
		}
		templateVars := &claude.TemplateVariables{
			IssueNumber: int(issueNumber),
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}
		mockClaude.On("ExecuteInTmux", ctx, phaseConfig, templateVars, sessionName, "issue-28", workdir).Return(nil)

		// レビュー完了後のラベル追加
		mockLabel.On("AddLabel", ctx, int(issueNumber), "status:completed").Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, types.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockWorktree.AssertExpectations(t)
		mockClaude.AssertExpectations(t)
	})

	t.Run("異常系: 既に処理済み", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 既に処理済み
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateReview).Return(true)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err) // 処理済みはエラーではない
		mockTmux.AssertNotCalled(t, "CreateWindowForIssue")
		mockLabel.AssertNotCalled(t, "TransitionLabel")
		mockWorktree.AssertNotCalled(t, "UpdateMainBranch")
		mockClaude.AssertNotCalled(t, "ExecuteInTmux")
		mockState.AssertExpectations(t)
	})

	t.Run("異常系: tmuxウィンドウ作成失敗", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateReview, types.IssueStatusProcessing)

		// tmuxウィンドウ作成失敗
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, types.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create tmux window")
		mockWorktree.AssertNotCalled(t, "UpdateMainBranch")
		mockClaude.AssertNotCalled(t, "ExecuteInTmux")
		mockState.AssertExpectations(t)
		mockTmux.AssertExpectations(t)
	})

	t.Run("正常系: 独立したReviewフェーズのworktreeを作成", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateReview, types.IssueStatusProcessing)

		// tmuxウィンドウへの切り替え
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)
		mockTmux.On("SelectOrCreatePaneForPhase", sessionName, fmt.Sprintf("issue-%d", issueNumber), "review-phase").Return(nil)

		// mainブランチの更新
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)

		// worktreeの新規作成（独立したReviewフェーズのworktree）
		mockWorktree.On("CreateWorktreeForIssue", ctx, int(issueNumber)).Return(nil)

		// worktreeパスの取得（Reviewフェーズ用）
		workdir := "/tmp/osoba/worktree/28-review"
		mockWorktree.On("GetWorktreePathForIssue", int(issueNumber)).Return(workdir)

		// Claude実行
		phaseConfig := &claude.PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:review {{issue-number}}",
		}
		templateVars := &claude.TemplateVariables{
			IssueNumber: int(issueNumber),
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}
		mockClaude.On("ExecuteInTmux", ctx, phaseConfig, templateVars, sessionName, "issue-28", workdir).Return(nil)

		// レビュー完了後のラベル追加
		mockLabel.On("AddLabel", ctx, int(issueNumber), "status:completed").Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, types.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockWorktree.AssertExpectations(t)
		mockClaude.AssertExpectations(t)
	})

	t.Run("異常系: Claude実行失敗", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateReview, types.IssueStatusProcessing)

		// tmuxウィンドウへの切り替え
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)
		mockTmux.On("SelectOrCreatePaneForPhase", sessionName, fmt.Sprintf("issue-%d", issueNumber), "review-phase").Return(nil)

		// mainブランチの更新
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)

		// worktreeの新規作成
		mockWorktree.On("CreateWorktreeForIssue", ctx, int(issueNumber)).Return(nil)

		// worktreeパスの取得
		workdir := "/tmp/osoba/worktree/28-review"
		mockWorktree.On("GetWorktreePathForIssue", int(issueNumber)).Return(workdir)

		// Claude実行失敗
		phaseConfig := &claude.PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:review {{issue-number}}",
		}
		templateVars := &claude.TemplateVariables{
			IssueNumber: int(issueNumber),
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}
		mockClaude.On("ExecuteInTmux", ctx, phaseConfig, templateVars, sessionName, "issue-28", workdir).Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, types.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execute claude")
		mockLabel.AssertNotCalled(t, "AddLabel", ctx, int(issueNumber), "status:completed")
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockWorktree.AssertExpectations(t)
		mockClaude.AssertExpectations(t)
	})
}

func TestReviewAction_CanExecute(t *testing.T) {
	t.Run("実行可能: status:review-requestedラベルあり", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(28),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
				{Name: github.String("enhancement")},
			},
		}

		action := &ReviewAction{}

		// Act
		canExecute := action.CanExecute(issue)

		// Assert
		assert.True(t, canExecute)
	})

	t.Run("実行不可: status:review-requestedラベルなし", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(28),
			Labels: []*github.Label{
				{Name: github.String("status:ready")},
				{Name: github.String("enhancement")},
			},
		}

		action := &ReviewAction{}

		// Act
		canExecute := action.CanExecute(issue)

		// Assert
		assert.False(t, canExecute)
	})
}
