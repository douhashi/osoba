package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v67/github"
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
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStateReview, watcher.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:review-requested", "status:reviewing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("SwitchToIssueWindow", sessionName, int(issueNumber)).Return(nil)

		// 既存のworktreeパスを取得（レビューフェーズでは既存のworktreeを使用）
		workdir := "/tmp/osoba/worktree/28"

		// claudeプロンプト実行
		mockClaude.On("ExecuteReviewPrompt", ctx, int(issueNumber), workdir).Return(nil)

		// レビュー完了後のラベル追加
		mockLabel.On("AddLabel", ctx, int(issueNumber), "status:completed").Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, watcher.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		// GitManagerはレビューフェーズでは使用されない
		mockGit.AssertNotCalled(t, "CreateWorktreeForIssue")
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
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 既に処理済み
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateReview).Return(true)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err) // 処理済みはエラーではない
		mockTmux.AssertNotCalled(t, "SwitchToIssueWindow")
		mockLabel.AssertNotCalled(t, "TransitionLabel")
		mockClaude.AssertNotCalled(t, "ExecuteReviewPrompt")
		mockState.AssertExpectations(t)
	})

	t.Run("異常系: ラベル遷移失敗", func(t *testing.T) {
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
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStateReview, watcher.IssueStatusProcessing)

		// ラベル遷移失敗
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:review-requested", "status:reviewing").Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, watcher.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition label")
		mockTmux.AssertNotCalled(t, "SwitchToIssueWindow")
		mockClaude.AssertNotCalled(t, "ExecuteReviewPrompt")
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
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
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateReview).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStateReview, watcher.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:review-requested", "status:reviewing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("SwitchToIssueWindow", sessionName, int(issueNumber)).Return(nil)

		// claudeプロンプト実行失敗
		workdir := "/tmp/osoba/worktree/28"
		mockClaude.On("ExecuteReviewPrompt", ctx, int(issueNumber), workdir).Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, watcher.IssueStateReview)

		action := NewReviewAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execute claude prompt")
		mockLabel.AssertNotCalled(t, "AddLabel", ctx, int(issueNumber), "status:completed")
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
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
