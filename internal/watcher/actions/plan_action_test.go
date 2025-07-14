package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v50/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTmuxClient はtmux操作のモック
type MockTmuxClient struct {
	mock.Mock
}

func (m *MockTmuxClient) CreateWindowForIssue(sessionName string, issueNumber int) error {
	args := m.Called(sessionName, issueNumber)
	return args.Error(0)
}

func (m *MockTmuxClient) SwitchToIssueWindow(sessionName string, issueNumber int) error {
	args := m.Called(sessionName, issueNumber)
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

func (m *MockStateManager) GetState(issueNumber int64) (*watcher.IssueState, bool) {
	args := m.Called(issueNumber)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*watcher.IssueState), args.Bool(1)
}

func (m *MockStateManager) SetState(issueNumber int64, phase watcher.IssuePhase, status watcher.IssueStatus) {
	m.Called(issueNumber, phase, status)
}

func (m *MockStateManager) IsProcessing(issueNumber int64) bool {
	args := m.Called(issueNumber)
	return args.Bool(0)
}

func (m *MockStateManager) HasBeenProcessed(issueNumber int64, phase watcher.IssuePhase) bool {
	args := m.Called(issueNumber, phase)
	return args.Bool(0)
}

func (m *MockStateManager) MarkAsCompleted(issueNumber int64, phase watcher.IssuePhase) {
	m.Called(issueNumber, phase)
}

func (m *MockStateManager) MarkAsFailed(issueNumber int64, phase watcher.IssuePhase) {
	m.Called(issueNumber, phase)
}

func TestNewPlanAction(t *testing.T) {
	t.Run("PlanActionの作成", func(t *testing.T) {
		// Arrange
		sessionName := "osoba-test"
		mockTmux := new(MockTmuxClient)
		mockState := new(MockStateManager)

		// Act
		action := NewPlanAction(sessionName, mockTmux, mockState)

		// Assert
		assert.NotNil(t, action)
		assert.Equal(t, sessionName, action.sessionName)
		assert.Equal(t, mockTmux, action.tmuxClient)
		assert.Equal(t, mockState, action.stateManager)
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

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(false)

		// 処理開始
		mockState.On("SetState", issueNumber, watcher.IssueStatePlan, watcher.IssueStatusProcessing)

		// tmuxウィンドウ作成
		mockTmux.On("CreateWindowForIssue", sessionName, int(issueNumber)).Return(nil)

		// TODO: git worktree作成のモック

		// 処理完了
		mockState.On("MarkAsCompleted", issueNumber, watcher.IssueStatePlan)

		action := NewPlanAction(sessionName, mockTmux, mockState)

		// Act
		err := action.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockTmux.AssertExpectations(t)
		mockState.AssertExpectations(t)
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

		// 既に処理済み
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStatePlan).Return(true)

		action := NewPlanAction(sessionName, mockTmux, mockState)

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

		// 状態確認
		mockState.On("HasBeenProcessed", issueNumber, watcher.IssueStatePlan).Return(false)
		mockState.On("IsProcessing", issueNumber).Return(true)

		action := NewPlanAction(sessionName, mockTmux, mockState)

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
