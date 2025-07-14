package watcher

import (
	"context"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockActionExecutor はActionExecutorインターフェースのモック実装
type MockActionExecutor struct {
	mock.Mock
}

func (m *MockActionExecutor) Execute(ctx context.Context, issue *github.Issue) error {
	args := m.Called(ctx, issue)
	return args.Error(0)
}

func (m *MockActionExecutor) CanExecute(issue *github.Issue) bool {
	args := m.Called(issue)
	return args.Bool(0)
}

func TestActionExecutor_Execute(t *testing.T) {
	t.Run("正常系: Executeが正しく呼び出される", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		issue := &github.Issue{
			Number: github.Int(13),
			Title:  github.String("Test Issue"),
		}

		mockExecutor := new(MockActionExecutor)
		mockExecutor.On("Execute", ctx, issue).Return(nil)

		// Act
		err := mockExecutor.Execute(ctx, issue)

		// Assert
		assert.NoError(t, err)
		mockExecutor.AssertExpectations(t)
	})

	t.Run("異常系: Executeがエラーを返す", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		issue := &github.Issue{
			Number: github.Int(13),
			Title:  github.String("Test Issue"),
		}
		expectedErr := assert.AnError

		mockExecutor := new(MockActionExecutor)
		mockExecutor.On("Execute", ctx, issue).Return(expectedErr)

		// Act
		err := mockExecutor.Execute(ctx, issue)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockExecutor.AssertExpectations(t)
	})
}

func TestActionExecutor_CanExecute(t *testing.T) {
	t.Run("正常系: 実行可能な場合", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(13),
			Labels: []*github.Label{
				{Name: github.String("status:needs-plan")},
			},
		}

		mockExecutor := new(MockActionExecutor)
		mockExecutor.On("CanExecute", issue).Return(true)

		// Act
		canExecute := mockExecutor.CanExecute(issue)

		// Assert
		assert.True(t, canExecute)
		mockExecutor.AssertExpectations(t)
	})

	t.Run("正常系: 実行不可能な場合", func(t *testing.T) {
		// Arrange
		issue := &github.Issue{
			Number: github.Int(13),
			Labels: []*github.Label{
				{Name: github.String("unrelated-label")},
			},
		}

		mockExecutor := new(MockActionExecutor)
		mockExecutor.On("CanExecute", issue).Return(false)

		// Act
		canExecute := mockExecutor.CanExecute(issue)

		// Assert
		assert.False(t, canExecute)
		mockExecutor.AssertExpectations(t)
	})
}

// ラベルベースアクション判定のテスト
func TestGetActionForIssue(t *testing.T) {
	tests := []struct {
		name     string
		issue    *github.Issue
		wantType string
		wantNil  bool
	}{
		{
			name: "status:needs-planラベルの場合",
			issue: &github.Issue{
				Number: github.Int(1),
				Labels: []*github.Label{
					{Name: github.String("status:needs-plan")},
				},
			},
			wantType: "plan",
			wantNil:  false,
		},
		{
			name: "status:readyラベルの場合",
			issue: &github.Issue{
				Number: github.Int(2),
				Labels: []*github.Label{
					{Name: github.String("status:ready")},
				},
			},
			wantType: "implementation",
			wantNil:  false,
		},
		{
			name: "status:review-requestedラベルの場合",
			issue: &github.Issue{
				Number: github.Int(3),
				Labels: []*github.Label{
					{Name: github.String("status:review-requested")},
				},
			},
			wantType: "review",
			wantNil:  false,
		},
		{
			name: "対応するラベルがない場合",
			issue: &github.Issue{
				Number: github.Int(4),
				Labels: []*github.Label{
					{Name: github.String("bug")},
					{Name: github.String("enhancement")},
				},
			},
			wantNil: true,
		},
		{
			name: "ラベルが空の場合",
			issue: &github.Issue{
				Number: github.Int(5),
				Labels: []*github.Label{},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			action := GetActionForIssue(tt.issue)

			// Assert
			if tt.wantNil {
				assert.Nil(t, action)
			} else {
				assert.NotNil(t, action)
				// ActionTypeメソッドがあることを確認
				if typeGetter, ok := action.(interface{ ActionType() string }); ok {
					assert.Equal(t, tt.wantType, typeGetter.ActionType())
				}
			}
		})
	}
}
