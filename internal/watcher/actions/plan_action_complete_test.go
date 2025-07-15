package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/types"
	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLabelManager はラベル管理のモック
type MockLabelManager struct {
	mock.Mock
}

func (m *MockLabelManager) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	args := m.Called(ctx, issueNumber, from, to)
	return args.Error(0)
}

func (m *MockLabelManager) AddLabel(ctx context.Context, issueNumber int, label string) error {
	args := m.Called(ctx, issueNumber, label)
	return args.Error(0)
}

func (m *MockLabelManager) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	args := m.Called(ctx, issueNumber, label)
	return args.Error(0)
}

// MockGitManager はGit操作のモック
type MockGitManager struct {
	mock.Mock
}

func (m *MockGitManager) CreateWorktreeForIssue(issueNumber int, branchName string) (string, error) {
	args := m.Called(issueNumber, branchName)
	return args.String(0), args.Error(1)
}

func (m *MockGitManager) DeleteWorktree(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// MockClaudeManager はClaude実行のモック
type MockClaudeManager struct {
	mock.Mock
}

func (m *MockClaudeManager) ExecutePlanPrompt(ctx context.Context, issueNumber int, workdir string) error {
	args := m.Called(ctx, issueNumber, workdir)
	return args.Error(0)
}

func (m *MockClaudeManager) ExecuteImplementationPrompt(ctx context.Context, issueNumber int, workdir string) error {
	args := m.Called(ctx, issueNumber, workdir)
	return args.Error(0)
}

func (m *MockClaudeManager) ExecuteReviewPrompt(ctx context.Context, issueNumber int, workdir string) error {
	args := m.Called(ctx, issueNumber, workdir)
	return args.Error(0)
}

func TestPlanActionComplete_Execute(t *testing.T) {
	t.Run("正常系: 完全な実行フロー", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:needs-plan", "status:planning").Return(nil)

		// tmuxウィンドウ作成
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)

		// git worktree作成
		workdir := "/tmp/osoba/worktree/28"
		mockGit.On("CreateWorktreeForIssue", int(issueNumber), "feat/#28-phase-action-execution").Return(workdir, nil)

		// claudeプロンプト実行
		mockClaude.On("ExecutePlanPrompt", ctx, int(issueNumber), workdir).Return(nil)

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, types.IssueStatePlan)

		action := NewPlanActionComplete(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockGit.AssertExpectations(t)
		mockClaude.AssertExpectations(t)
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
				{Name: github.String("status:needs-plan")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

		// ラベル遷移失敗
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:needs-plan", "status:planning").Return(assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, types.IssueStatePlan)

		action := NewPlanActionComplete(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transition label")
		mockTmux.AssertNotCalled(t, "CreateWindowForIssue")
		mockGit.AssertNotCalled(t, "CreateWorktreeForIssue")
		mockClaude.AssertNotCalled(t, "ExecutePlanPrompt")
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
	})

	t.Run("異常系: git worktree作成失敗", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := int64(28)
		issue := &github.Issue{
			Number: github.Int(int(issueNumber)),
			Title:  github.String("Test Issue"),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)
		mockLabel := new(MockLabelManager)
		mockGit := new(MockGitManager)
		mockClaude := new(MockClaudeManager)

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, types.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

		// ラベル遷移
		mockLabel.On("TransitionLabel", ctx, int(issueNumber), "status:needs-plan", "status:planning").Return(nil)

		// tmuxウィンドウ作成
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)

		// git worktree作成失敗
		mockGit.On("CreateWorktreeForIssue", int(issueNumber), "feat/#28-phase-action-execution").Return("", assert.AnError)

		// 処理失敗
		mockState.On("MarkAsFailed", issueNumber, types.IssueStatePlan)

		action := NewPlanActionComplete(sessionName, mockTmux, mockState, mockLabel, mockGit, mockClaude)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "create worktree")
		mockClaude.AssertNotCalled(t, "ExecutePlanPrompt")
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
		mockLabel.AssertExpectations(t)
		mockGit.AssertExpectations(t)
	})
}
