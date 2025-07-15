package github

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
)

func TestLabelManager_404ErrorHandling(t *testing.T) {
	t.Run("ラベル削除時の404エラーを適切に処理する", func(t *testing.T) {
		mockService := new(MockLabelService)
		manager := NewLabelManager(mockService)

		ctx := context.Background()

		// 現在のラベル（status:needs-plan）を返すモック
		needsPlanLabel := &github.Label{Name: github.String("status:needs-plan")}
		mockService.On("ListLabelsByIssue", ctx, "test-owner", "test-repo", 123, (*github.ListOptions)(nil)).
			Return([]*github.Label{needsPlanLabel}, &github.Response{}, nil)

		// 404エラーをシミュレート
		notFoundError := errors.New("404 Not Found: Label does not exist")
		mockService.On("RemoveLabelForIssue", ctx, "test-owner", "test-repo", 123, "status:needs-plan").
			Return((*github.Response)(nil), notFoundError)

		// ラベル追加は成功するようにモック（404エラーが無視される場合に備えて）
		planningLabel := &github.Label{Name: github.String("status:planning")}
		mockService.On("AddLabelsToIssue", ctx, "test-owner", "test-repo", 123, []string{"status:planning"}).
			Return([]*github.Label{planningLabel}, &github.Response{}, nil)

		// ラベル遷移を実行
		transitioned, err := manager.TransitionLabel(ctx, "test-owner", "test-repo", 123)

		// 修正後は404エラーが無視されて処理が成功する
		if err == nil && transitioned {
			t.Logf("修正されました: 404エラーが無視されて処理が成功しました")
		} else {
			t.Errorf("404エラーが適切に処理されませんでした: transitioned=%v, err=%v", transitioned, err)
		}

		mockService.AssertExpectations(t)
	})

	t.Run("修正後: ラベル削除時の404エラーを無視して処理続行", func(t *testing.T) {
		mockService := new(MockLabelService)
		manager := NewLabelManager(mockService)

		ctx := context.Background()

		// 現在のラベル（status:needs-plan）を返すモック
		needsPlanLabel := &github.Label{Name: github.String("status:needs-plan")}
		mockService.On("ListLabelsByIssue", ctx, "test-owner", "test-repo", 123, (*github.ListOptions)(nil)).
			Return([]*github.Label{needsPlanLabel}, &github.Response{}, nil)

		// 404エラーをシミュレート
		notFoundError := errors.New("404 Not Found: Label does not exist")
		mockService.On("RemoveLabelForIssue", ctx, "test-owner", "test-repo", 123, "status:needs-plan").
			Return((*github.Response)(nil), notFoundError)

		// ラベル追加は成功するようにモック
		planningLabel := &github.Label{Name: github.String("status:planning")}
		mockService.On("AddLabelsToIssue", ctx, "test-owner", "test-repo", 123, []string{"status:planning"}).
			Return([]*github.Label{planningLabel}, &github.Response{}, nil)

		// ラベル遷移を実行
		transitioned, err := manager.TransitionLabel(ctx, "test-owner", "test-repo", 123)

		// 404エラーが無視されて処理が続行されることを期待
		assert.NoError(t, err, "404エラーは無視されて処理が続行されるべき")
		assert.True(t, transitioned, "ラベル遷移は成功するべき")

		mockService.AssertExpectations(t)
	})
}
