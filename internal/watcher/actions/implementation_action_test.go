package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
)

func TestImplementationAction_Execute(t *testing.T) {
	t.Run("正常系: 完全な実行フロー", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:ready")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStateImplementation, watcher.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("SwitchToIssueWindow", sessionName, int(issueNumber)).Return(nil)

		// 既存のworktreeパスを取得（実装フェーズでは既存のworktreeを使用）
		workdir := "/tmp/osoba/worktree/28"

		// claudeプロンプト実行
		mockClaude.On("ExecuteImplementationPrompt", ctx, int(issueNumber), workdir).Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, watcher.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		// GitManagerは実装フェーズでは使用されない
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
				{Name: github.String("status:ready")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 既に処理済み
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateImplementation).Return(true)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err) // 処理済みはエラーではない
		mockTmux.AssertNotCalled(t, "SwitchToIssueWindow")
		mockLabel.AssertNotCalled(t, "TransitionLabel")
		mockClaude.AssertNotCalled(t, "ExecuteImplementationPrompt")
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
				{Name: github.String("status:ready")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStateImplementation, watcher.IssueStatusProcessing)

		// ラベル遷移失敗
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, watcher.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition label")
		mockTmux.AssertNotCalled(t, "SwitchToIssueWindow")
		mockClaude.AssertNotCalled(t, "ExecuteImplementationPrompt")
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
				{Name: github.String("status:ready")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStateImplementation, watcher.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("SwitchToIssueWindow", sessionName, int(issueNumber)).Return(nil)

		// claudeプロンプト実行失敗
		workdir := "/tmp/osoba/worktree/28"
		mockClaude.On("ExecuteImplementationPrompt", ctx, int(issueNumber), workdir).Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, watcher.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execute claude prompt")
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockClaude.AssertExpectations(t)
	})
}

func TestImplementationAction_CanExecute(t *testing.T) {
	t.Run("実行可能: status:readyラベルあり", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(28),
			Labels: []*github.Label{
				{Name: github.String("status:ready")},
				{Name: github.String("enhancement")},
			},
		}

		action := &ImplementationAction{}

		// Act
		canExecute := action.CanExecute(issue)

		// Assert
		assert.True(t, canExecute)
	})

	t.Run("実行不可: status:readyラベルなし", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(28),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
				{Name: github.String("enhancement")},
			},
		}

		action := &ImplementationAction{}

		// Act
		canExecute := action.CanExecute(issue)

		// Assert
		assert.False(t, canExecute)
	})
}
