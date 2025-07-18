package actions

import (
	"context"
	"fmt"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewPlanAction(t *testing.T) {
	t.Run("PlanActionの作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		config := claude.NewDefaultClaudeConfig()

		// Act
		action := NewPlanAction(sessionName, mockTmux, mockState, mockWorktree, mockClaude, config)

		// Assert
		assert.NotNil(t, action)
		assert.Equal(t, sessionName, action.sessionName)
		assert.Equal(t, mockTmux, action.tmuxClient)
		assert.Equal(t, mockState, action.stateManager)
		assert.Equal(t, mockWorktree, action.worktreeManager)
		assert.Equal(t, mockClaude, action.claudeExecutor)
		assert.Equal(t, config, action.claudeConfig)
	})
}

func TestPlanAction_Execute(t *testing.T) {
	t.Run("正常系: 初回実行", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(13)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		config := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

		// tmuxウィンドウ作成
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)
		mockTmux.On("SelectOrCreatePaneForPhase", sessionName, fmt.Sprintf("issue-%d", issueNumber), "plan-phase").Return(nil)

		// mainブランチ更新とworktree作成
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)
		mockWorktree.On("CreateWorktreeForIssue", ctx, int(issueNumber)).Return(nil)
		mockWorktree.On("GetWorktreePathForIssue", int(issueNumber)).Return("/tmp/worktree/13-plan")

		// Claude実行
		mockClaude.On("ExecuteInTmux", ctx, config.Phases["plan"], mock.AnythingOfType("*claude.TemplateVariables"), sessionName, "issue-13", "/tmp/worktree/13-plan").Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, types.IssueStatePlan)

		action := NewPlanAction(sessionName, mockTmux, mockState, mockWorktree, mockClaude, config)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockWorktree.AssertExpectations(t)
		mockClaude.AssertExpectations(t)
	})

	t.Run("異常系: 既に処理済み", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(13)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		config := claude.NewDefaultClaudeConfig()

		// 既に処理済み
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStatePlan).Return(true)

		action := NewPlanAction(sessionName, mockTmux, mockState, mockWorktree, mockClaude, config)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err) // 処理済みはエラーではない
		mockTmux.AssertNotCalled(t, "CreateWindowForIssue")
		mockState.AssertExpectations(t)
	})

	t.Run("異常系: 処理中", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(13)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockWorktree := new(MockWorktreeManager)
		mockClaude := new(MockClaudeExecutor)
		config := claude.NewDefaultClaudeConfig()

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(true)

		action := NewPlanAction(sessionName, mockTmux, mockState, mockWorktree, mockClaude, config)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already processing")
		mockTmux.AssertNotCalled(t, "CreateWindowForIssue")
		mockState.AssertExpectations(t)
	})
}

func TestPlanAction_CanExecute(t *testing.T) {
	t.Run("実行可能: status:needs-planラベルあり", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(13),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
				{Name: github.String("bug")},
			},
		}

		action := &PlanAction{}

		// Act
		canExecute := action.CanExecute(issue)

		// Assert
		assert.True(t, canExecute)
	})

	t.Run("実行不可: status:needs-planラベルなし", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(13),
			Labels: []*github.Label{
				{Name: github.String("status:ready")},
				{Name: github.String("bug")},
			},
		}

		action := &PlanAction{}

		// Act
		canExecute := action.CanExecute(issue)

		// Assert
		assert.False(t, canExecute)
	})
}
