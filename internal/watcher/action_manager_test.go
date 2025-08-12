package watcher

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockActionExecutorExt はActionExecutorのモック（拡張版）
type MockActionExecutorExt struct {
	mock.Mock
}

func (m *MockActionExecutorExt) Execute(ctx context.Context, issue *github.Issue) error {
	args := m.Called(ctx, issue)
	return args.Error(0)
}

func (m *MockActionExecutorExt) CanExecute(issue *github.Issue) bool {
	args := m.Called(issue)
	return args.Bool(0)
}

// MockActionFactory はアクションファクトリのモック
type MockActionFactory struct {
	mock.Mock
}

func (m *MockActionFactory) CreatePlanAction() ActionExecutor {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ActionExecutor)
}

func (m *MockActionFactory) CreateImplementationAction() ActionExecutor {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ActionExecutor)
}

func (m *MockActionFactory) CreateReviewAction() ActionExecutor {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ActionExecutor)
}

func (m *MockActionFactory) CreateReviseAction() ActionExecutor {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ActionExecutor)
}

func (m *MockActionFactory) CreateNoOpAction() ActionExecutor {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(ActionExecutor)
}

func TestActionManagerExtended_ExecuteAction(t *testing.T) {
	t.Run("正常系: PlanActionの実行", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := 28
		issue := &github.Issue{
			Number: github.Int(issueNumber),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockFactory := new(MockActionFactory)
		mockAction := new(MockActionExecutorExt)

		// ファクトリからアクションを取得
		mockFactory.On("CreatePlanAction").Return(mockAction)

		// アクションの実行
		mockAction.On("CanExecute", issue).Return(true)
		mockAction.On("Execute", ctx, issue).Return(nil)

		manager := NewActionManagerExtended(sessionName, mockFactory)

		// Act
		err := manager.ExecuteAction(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockFactory.AssertExpectations(t)
		mockAction.AssertExpectations(t)
	})

	t.Run("正常系: ImplementationActionの実行", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := 28
		issue := &github.Issue{
			Number: github.Int(issueNumber),
			Labels: []*github.Label{
				{Name: github.String("status:ready")},
			},
		}

		mockFactory := new(MockActionFactory)
		mockAction := new(MockActionExecutorExt)

		// ファクトリからアクションを取得
		mockFactory.On("CreateImplementationAction").Return(mockAction)

		// アクションの実行
		mockAction.On("CanExecute", issue).Return(true)
		mockAction.On("Execute", ctx, issue).Return(nil)

		manager := NewActionManagerExtended(sessionName, mockFactory)

		// Act
		err := manager.ExecuteAction(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockFactory.AssertExpectations(t)
		mockAction.AssertExpectations(t)
	})

	t.Run("正常系: ReviewActionの実行", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := 28
		issue := &github.Issue{
			Number: github.Int(issueNumber),
			Labels: []*github.Label{
				{Name: github.String("status:review-requested")},
			},
		}

		mockFactory := new(MockActionFactory)
		mockAction := new(MockActionExecutorExt)

		// ファクトリからアクションを取得
		mockFactory.On("CreateReviewAction").Return(mockAction)

		// アクションの実行
		mockAction.On("CanExecute", issue).Return(true)
		mockAction.On("Execute", ctx, issue).Return(nil)

		manager := NewActionManagerExtended(sessionName, mockFactory)

		// Act
		err := manager.ExecuteAction(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockFactory.AssertExpectations(t)
		mockAction.AssertExpectations(t)
	})

	t.Run("異常系: 適切なアクションが見つからない", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := 28
		issue := &github.Issue{
			Number: github.Int(issueNumber),
			Labels: []*github.Label{
				{Name: github.String("bug")}, // アクション対象外のラベル
			},
		}

		mockFactory := new(MockActionFactory)
		manager := NewActionManagerExtended(sessionName, mockFactory)

		// Act
		err := manager.ExecuteAction(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no action found")
		mockFactory.AssertNotCalled(t, "CreatePlanAction")
		mockFactory.AssertNotCalled(t, "CreateImplementationAction")
		mockFactory.AssertNotCalled(t, "CreateReviewAction")
	})

	t.Run("異常系: アクションが実行不可", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		sessionName := "osoba-test"
		issueNumber := 28
		issue := &github.Issue{
			Number: github.Int(issueNumber),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockFactory := new(MockActionFactory)
		mockAction := new(MockActionExecutorExt)

		// ファクトリからアクションを取得
		mockFactory.On("CreatePlanAction").Return(mockAction)

		// アクションが実行不可
		mockAction.On("CanExecute", issue).Return(false)

		manager := NewActionManagerExtended(sessionName, mockFactory)

		// Act
		err := manager.ExecuteAction(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "action cannot be executed")
		mockFactory.AssertExpectations(t)
		mockAction.AssertExpectations(t)
		mockAction.AssertNotCalled(t, "Execute")
	})
}
