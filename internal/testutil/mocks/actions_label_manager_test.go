package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/watcher/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockLabelManager(t *testing.T) {
	t.Run("TransitionLabel", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager)

		// 成功ケース
		mockLabelManager.On("TransitionLabel", ctx, 123, "status:needs-plan", "status:planning").Return(nil)

		err := mockLabelManager.TransitionLabel(ctx, 123, "status:needs-plan", "status:planning")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("AddLabel", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager)

		// 成功ケース
		mockLabelManager.On("AddLabel", ctx, 456, "status:implementing").Return(nil)

		err := mockLabelManager.AddLabel(ctx, 456, "status:implementing")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("RemoveLabel", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager)

		// 成功ケース
		mockLabelManager.On("RemoveLabel", ctx, 789, "status:needs-plan").Return(nil)

		err := mockLabelManager.RemoveLabel(ctx, 789, "status:needs-plan")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithSuccessfulTransition", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager).
			WithSuccessfulTransition(ctx, 100, "from", "to")

		err := mockLabelManager.TransitionLabel(ctx, 100, "from", "to")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithTransitionError", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("transition failed")
		mockLabelManager := new(mocks.MockLabelManager).
			WithTransitionError(ctx, 200, "from", "to", expectedErr)

		err := mockLabelManager.TransitionLabel(ctx, 200, "from", "to")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithAddLabelSuccess", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager).
			WithAddLabelSuccess(ctx, 300, "new-label")

		err := mockLabelManager.AddLabel(ctx, 300, "new-label")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithAddLabelError", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("add label failed")
		mockLabelManager := new(mocks.MockLabelManager).
			WithAddLabelError(ctx, 400, "bad-label", expectedErr)

		err := mockLabelManager.AddLabel(ctx, 400, "bad-label")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithRemoveLabelSuccess", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager).
			WithRemoveLabelSuccess(ctx, 500, "old-label")

		err := mockLabelManager.RemoveLabel(ctx, 500, "old-label")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithRemoveLabelError", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("remove label failed")
		mockLabelManager := new(mocks.MockLabelManager).
			WithRemoveLabelError(ctx, 600, "stuck-label", expectedErr)

		err := mockLabelManager.RemoveLabel(ctx, 600, "stuck-label")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockLabelManager.AssertExpectations(t)
	})

	t.Run("WithDefaultBehavior", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager).
			WithDefaultBehavior()

		// すべてのメソッドがデフォルトで成功を返すことを確認
		err := mockLabelManager.TransitionLabel(ctx, 700, "any", "label")
		assert.NoError(t, err)

		err = mockLabelManager.AddLabel(ctx, 800, "any-label")
		assert.NoError(t, err)

		err = mockLabelManager.RemoveLabel(ctx, 900, "any-label")
		assert.NoError(t, err)

		// AssertExpectationsは呼ばない（Maybeを使用しているため）
	})

	t.Run("interface compliance", func(t *testing.T) {
		// インターフェースの実装を静的にチェック
		var _ actions.ActionsLabelManager = (*mocks.MockLabelManager)(nil)
	})

	t.Run("chaining support", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager).
			WithSuccessfulTransition(ctx, 100, "from1", "to1").
			WithSuccessfulTransition(ctx, 200, "from2", "to2").
			WithAddLabelSuccess(ctx, 300, "label1").
			WithRemoveLabelSuccess(ctx, 400, "label2")

		// 複数の期待値が正しく設定されていることを確認
		assert.NoError(t, mockLabelManager.TransitionLabel(ctx, 100, "from1", "to1"))
		assert.NoError(t, mockLabelManager.TransitionLabel(ctx, 200, "from2", "to2"))
		assert.NoError(t, mockLabelManager.AddLabel(ctx, 300, "label1"))
		assert.NoError(t, mockLabelManager.RemoveLabel(ctx, 400, "label2"))

		mockLabelManager.AssertExpectations(t)
	})

	t.Run("any argument matching", func(t *testing.T) {
		ctx := context.Background()
		mockLabelManager := new(mocks.MockLabelManager)

		// mock.Anythingを使用した柔軟なマッチング
		mockLabelManager.On("TransitionLabel", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)

		err := mockLabelManager.TransitionLabel(ctx, 999, "any-from", "any-to")
		assert.NoError(t, err)
		mockLabelManager.AssertExpectations(t)
	})
}
