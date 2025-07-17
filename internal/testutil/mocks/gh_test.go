package mocks_test

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockGHExecutor(t *testing.T) {
	t.Run("基本的な動作", func(t *testing.T) {
		executor := mocks.NewMockGHExecutor()
		ctx := context.Background()

		// モックの設定
		executor.On("Execute", ctx, []string{"api", "repos/owner/repo"}).
			Return([]byte(`{"name": "repo", "full_name": "owner/repo"}`), nil)

		// 実行
		result, err := executor.Execute(ctx, []string{"api", "repos/owner/repo"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"name": "repo", "full_name": "owner/repo"}`, string(result))

		executor.AssertExpectations(t)
	})

	t.Run("エラーケース", func(t *testing.T) {
		executor := mocks.NewMockGHExecutor()
		ctx := context.Background()

		// エラーを返すモック設定
		executor.On("Execute", ctx, []string{"api", "invalid"}).
			Return([]byte(nil), assert.AnError)

		// 実行
		result, err := executor.Execute(ctx, []string{"api", "invalid"})
		assert.Error(t, err)
		assert.Nil(t, result)

		executor.AssertExpectations(t)
	})

	t.Run("デフォルト動作の確認", func(t *testing.T) {
		executor := mocks.NewMockGHExecutor().WithDefaultBehavior()
		ctx := context.Background()

		t.Run("API呼び出しのデフォルト", func(t *testing.T) {
			result, err := executor.Execute(ctx, []string{"api", "rate_limit"})
			require.NoError(t, err)
			assert.Equal(t, "{}", string(result))
		})

		t.Run("issue listのデフォルト", func(t *testing.T) {
			result, err := executor.Execute(ctx, []string{"issue", "list", "--repo", "owner/repo"})
			require.NoError(t, err)
			assert.Equal(t, "[]", string(result))
		})

		t.Run("issue commentのデフォルト", func(t *testing.T) {
			result, err := executor.Execute(ctx, []string{"issue", "comment", "123", "--body", "test"})
			require.NoError(t, err)
			assert.Equal(t, "", string(result))
		})
	})

	t.Run("引数のマッチング", func(t *testing.T) {
		executor := mocks.NewMockGHExecutor()
		ctx := context.Background()

		// 引数の部分一致を使ったモック設定
		executor.On("Execute", ctx, mock.MatchedBy(func(args []string) bool {
			return len(args) >= 2 && args[0] == "pr" && args[1] == "create"
		})).Return([]byte(`{"number": 123}`), nil)

		// 実行
		result, err := executor.Execute(ctx, []string{"pr", "create", "--title", "Test PR"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"number": 123}`, string(result))

		executor.AssertExpectations(t)
	})
}
