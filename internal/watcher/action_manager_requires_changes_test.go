package watcher

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestActionManagerExtended_RequiresChanges(t *testing.T) {
	t.Run("returns ReviseAction for status:requires-changes label", func(t *testing.T) {
		// モックファクトリーを作成
		mockFactory := new(MockActionFactory)
		mockReviseAction := new(MockActionExecutorExt)
		mockFactory.On("CreateReviseAction").Return(mockReviseAction)

		// ActionManagerを作成
		manager := NewActionManagerExtended("test-session", mockFactory)

		// status:requires-changesラベルを持つIssueを作成
		issue := &github.Issue{
			Number: intPtr(222),
			Labels: []*github.Label{
				{Name: stringPtr("status:requires-changes")},
			},
		}

		// アクションを取得
		action := manager.GetActionForIssue(issue)

		// ReviseActionが返されることを確認
		assert.NotNil(t, action)
		assert.Equal(t, mockReviseAction, action)

		mockFactory.AssertExpectations(t)
	})

	t.Run("executes ReviseAction successfully", func(t *testing.T) {
		// モックファクトリーを作成
		mockFactory := new(MockActionFactory)
		mockReviseAction := new(MockActionExecutorExt)
		mockReviseAction.On("CanExecute", mock.Anything).Return(true)
		mockReviseAction.On("Execute", mock.Anything, mock.Anything).Return(nil)
		mockFactory.On("CreateReviseAction").Return(mockReviseAction)

		// ActionManagerを作成
		manager := NewActionManagerExtended("test-session", mockFactory)

		// status:requires-changesラベルを持つIssueを作成
		issue := &github.Issue{
			Number: intPtr(222),
			Labels: []*github.Label{
				{Name: stringPtr("status:requires-changes")},
			},
		}

		// アクションを実行
		err := manager.ExecuteAction(context.Background(), issue)

		// エラーなく実行されることを確認
		assert.NoError(t, err)

		mockFactory.AssertExpectations(t)
		mockReviseAction.AssertExpectations(t)
	})
}
