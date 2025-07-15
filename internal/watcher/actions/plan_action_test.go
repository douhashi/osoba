package actions

import (
	"context"
	"os/exec"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/types"
	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTmuxClient はtmux操作のモック
type MockTmuxClient struct {
	mock.Mock
}

func (m *MockTmuxClient) CreateWindowForIssue(sessionName string, issueNumber int, phase string) error {
	args := m.Called(sessionName, issueNumber, phase)
	return args.Error(0)
}

func (m *MockTmuxClient) SwitchToIssueWindow(sessionName string, issueNumber int, phase string) error {
	args := m.Called(sessionName, issueNumber, phase)
	return args.Error(0)
}

func (m *MockTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	args := m.Called(sessionName, windowName)
	return args.Bool(0), args.Error(1)
}

// MockStateManager は状態管理のモック
type MockStateManager struct {
	mock.Mock
}

func (m *MockStateManager) GetState(issueNumber int64) (*types.IssueState, bool) {
	args := m.Called(issueNumber)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*types.IssueState), args.Bool(1)
}

func (m *MockStateManager) SetState(issueNumber int64, phase types.IssuePhase, status types.IssueStatus) {
	m.Called(issueNumber, phase, status)
}

func (m *MockStateManager) IsProcessing(issueNumber int64) bool {
	args := m.Called(issueNumber)
	return args.Bool(0)
}

func (m *MockStateManager) HasBeenProcessed(issueNumber int64, phase types.IssuePhase) bool {
	args := m.Called(issueNumber, phase)
	return args.Bool(0)
}

func (m *MockStateManager) MarkAsCompleted(issueNumber int64, phase types.IssuePhase) {
	m.Called(issueNumber, phase)
}

func (m *MockStateManager) MarkAsFailed(issueNumber int64, phase types.IssuePhase) {
	m.Called(issueNumber, phase)
}

// MockWorktreeManager はWorktreeManagerのモック
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) UpdateMainBranch(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockWorktreeManager) CreateWorktree(ctx context.Context, issueNumber int, phase git.Phase) error {
	args := m.Called(ctx, issueNumber, phase)
	return args.Error(0)
}

func (m *MockWorktreeManager) RemoveWorktree(ctx context.Context, issueNumber int, phase git.Phase) error {
	args := m.Called(ctx, issueNumber, phase)
	return args.Error(0)
}

func (m *MockWorktreeManager) GetWorktreePath(issueNumber int, phase git.Phase) string {
	args := m.Called(issueNumber, phase)
	return args.String(0)
}

func (m *MockWorktreeManager) WorktreeExists(ctx context.Context, issueNumber int, phase git.Phase) (bool, error) {
	args := m.Called(ctx, issueNumber, phase)
	return args.Bool(0), args.Error(1)
}

// MockClaudeExecutor はClaudeExecutorのモック
type MockClaudeExecutor struct {
	mock.Mock
}

func (m *MockClaudeExecutor) CheckClaudeExists() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClaudeExecutor) BuildCommand(ctx context.Context, args []string, prompt string, workdir string) *exec.Cmd {
	argList := m.Called(ctx, args, prompt, workdir)
	if cmd := argList.Get(0); cmd != nil {
		return cmd.(*exec.Cmd)
	}
	return nil
}

func (m *MockClaudeExecutor) ExecuteInWorktree(ctx context.Context, config *claude.PhaseConfig, vars *claude.TemplateVariables, workdir string) error {
	args := m.Called(ctx, config, vars, workdir)
	return args.Error(0)
}

func (m *MockClaudeExecutor) ExecuteInTmux(ctx context.Context, config *claude.PhaseConfig, vars *claude.TemplateVariables, sessionName, windowName, workdir string) error {
	args := m.Called(ctx, config, vars, sessionName, windowName, workdir)
	return args.Error(0)
}

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
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber), "plan").Return(nil)

		// mainブランチ更新とworktree作成
		mockWorktree.On("UpdateMainBranch", ctx).Return(nil)
		mockWorktree.On("CreateWorktree", ctx, int(issueNumber), git.PhasePlan).Return(nil)
		mockWorktree.On("GetWorktreePath", int(issueNumber), git.PhasePlan).Return("/tmp/worktree/13-plan")

		// Claude実行
		mockClaude.On("ExecuteInTmux", ctx, config.Phases["plan"], mock.AnythingOfType("*claude.TemplateVariables"), sessionName, "13-plan", "/tmp/worktree/13-plan").Return(nil)

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
