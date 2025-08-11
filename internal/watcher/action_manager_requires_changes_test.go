package watcher

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestActionManagerExtended_RequiresChanges(t *testing.T) {
	t.Run("returns NoOpAction for status:requires-changes label", func(t *testing.T) {
		// モックファクトリーを作成
		mockFactory := new(MockActionFactory)
		noOpAction := NewNoOpAction(NewMockLogger())
		mockFactory.On("CreateNoOpAction").Return(noOpAction)

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

		// NoOpActionが返されることを確認
		assert.NotNil(t, action)
		// NoOpActionの型であることを確認
		_, ok := action.(*NoOpAction)
		assert.True(t, ok, "Expected NoOpAction type")

		mockFactory.AssertExpectations(t)
	})

	t.Run("executes NoOpAction successfully", func(t *testing.T) {
		// モックファクトリーを作成
		mockFactory := new(MockActionFactory)
		noOpAction := NewNoOpAction(NewMockLogger())
		mockFactory.On("CreateNoOpAction").Return(noOpAction)

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
	})
}
