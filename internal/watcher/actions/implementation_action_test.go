package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/types"
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
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber), "implement").Return(nil)

		// mainブランチの更新
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)

		// worktreeの新規作成
		mockWorktree.On("CreateWorktree", ctx, int(issueNumber), git.PhaseImplementation).Return(nil)

		// worktreeパスの取得
		workdir := "/tmp/osoba/worktree/28-implementation"
		mockWorktree.On("GetWorktreePath", int(issueNumber), git.PhaseImplementation).Return(workdir)

		// Claude実行
		phaseConfig := &claude.PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:implement {{issue-number}}",
		}
		templateVars := &claude.TemplateVariables{
			IssueNumber: int(issueNumber),
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}
		mockClaude.On("ExecuteInTmux", ctx, phaseConfig, templateVars, sessionName, "28-implement", workdir).Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, types.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

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
				{Name: github.String("status:ready")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 既に処理済み
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateImplementation).Return(true)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

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
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

		// ラベル遷移失敗
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, types.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition label")
		mockTmux.AssertNotCalled(t, "CreateWindowForIssue")
		mockWorktree.AssertNotCalled(t, "UpdateMainBranch")
		mockClaude.AssertNotCalled(t, "ExecuteInTmux")
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
	})

	t.Run("正常系: 独立したImplementationフェーズのworktreeを作成", func(t *testing.T) {
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
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber), "implement").Return(nil)

		// mainブランチの更新
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)

		// worktreeの新規作成（独立したImplementationフェーズのworktree）
		mockWorktree.On("CreateWorktree", ctx, int(issueNumber), git.PhaseImplementation).Return(nil)

		// worktreeパスの取得（Implementationフェーズ用）
		workdir := "/tmp/osoba/worktree/28-implementation"
		mockWorktree.On("GetWorktreePath", int(issueNumber), git.PhaseImplementation).Return(workdir)

		// Claude実行
		phaseConfig := &claude.PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:implement {{issue-number}}",
		}
		templateVars := &claude.TemplateVariables{
			IssueNumber: int(issueNumber),
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}
		mockClaude.On("ExecuteInTmux", ctx, phaseConfig, templateVars, sessionName, "28-implement", workdir).Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, types.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

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
				{Name: github.String("status:ready")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		claudeConfig := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStateImplementation).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:ready", "status:implementing").Return(nil)

		// tmuxウィンドウへの切り替え
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber), "implement").Return(nil)

		// mainブランチの更新
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)

		// worktreeの新規作成
		mockWorktree.On("CreateWorktree", ctx, int(issueNumber), git.PhaseImplementation).Return(nil)

		// worktreeパスの取得
		workdir := "/tmp/osoba/worktree/28-implementation"
		mockWorktree.On("GetWorktreePath", int(issueNumber), git.PhaseImplementation).Return(workdir)

		// Claude実行失敗
		phaseConfig := &claude.PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:implement {{issue-number}}",
		}
		templateVars := &claude.TemplateVariables{
			IssueNumber: int(issueNumber),
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}
		mockClaude.On("ExecuteInTmux", ctx, phaseConfig, templateVars, sessionName, "28-implement", workdir).Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, types.IssueStateImplementation)

		action := NewImplementationAction(sessionName, mockTmux, mockState, mockLabel, mockWorktree, mockClaude, claudeConfig)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "execute claude")
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockWorktree.AssertExpectations(t)
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
