package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockGHCommandExecutor(t *testing.T) {
	t.Run("Execute method", func(t *testing.T) {
		ctx := context.Background()
		mockExecutor := new(mocks.MockGHCommandExecutor)

		// 成功ケース
		mockExecutor.On("Execute", ctx, "echo", []string{"hello"}).Return("hello\n", nil)

		output, err := mockExecutor.Execute(ctx, "echo", "hello")
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", output)
		mockExecutor.AssertExpectations(t)
	})

	t.Run("Execute with error", func(t *testing.T) {
		ctx := context.Background()
		mockExecutor := new(mocks.MockGHCommandExecutor)

		expectedErr := errors.New("command failed")
		mockExecutor.On("Execute", ctx, "fail", []string{}).Return("", expectedErr)

		output, err := mockExecutor.Execute(ctx, "fail")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, output)
		mockExecutor.AssertExpectations(t)
	})

	t.Run("WithCommandOutput", func(t *testing.T) {
		ctx := context.Background()
		mockExecutor := new(mocks.MockGHCommandExecutor).
			WithCommandOutput(ctx, "ls", []string{"-la"}, "total 8\ndrwxr-xr-x  2 user\n")

		output, err := mockExecutor.Execute(ctx, "ls", "-la")
		assert.NoError(t, err)
		assert.Equal(t, "total 8\ndrwxr-xr-x  2 user\n", output)
		mockExecutor.AssertExpectations(t)
	})

	t.Run("WithCommandError", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("command not found")
		mockExecutor := new(mocks.MockGHCommandExecutor).
			WithCommandError(ctx, "invalid", []string{"arg"}, expectedErr)

		output, err := mockExecutor.Execute(ctx, "invalid", "arg")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, output)
		mockExecutor.AssertExpectations(t)
	})

	t.Run("WithDefaultBehavior", func(t *testing.T) {
		ctx := context.Background()
		mockExecutor := new(mocks.MockGHCommandExecutor).
			WithDefaultBehavior()

		// デフォルトではすべてのコマンドが成功
		output, err := mockExecutor.Execute(ctx, "any", "command", "args")
		assert.NoError(t, err)
		assert.Empty(t, output)
	})

	t.Run("interface compliance", func(t *testing.T) {
		// インターフェースの実装は、ghパッケージ側でテストされる
		// ここではメソッドの存在のみを確認
		mockExecutor := new(mocks.MockGHCommandExecutor)
		assert.NotNil(t, mockExecutor)
	})

	t.Run("chaining support", func(t *testing.T) {
		ctx := context.Background()
		mockExecutor := new(mocks.MockGHCommandExecutor).
			WithCommandOutput(ctx, "echo", []string{"test1"}, "test1\n").
			WithCommandOutput(ctx, "echo", []string{"test2"}, "test2\n").
			WithCommandError(ctx, "fail", []string{}, errors.New("error"))

		// 複数の期待値が正しく設定されていることを確認
		output1, err1 := mockExecutor.Execute(ctx, "echo", "test1")
		assert.NoError(t, err1)
		assert.Equal(t, "test1\n", output1)

		output2, err2 := mockExecutor.Execute(ctx, "echo", "test2")
		assert.NoError(t, err2)
		assert.Equal(t, "test2\n", output2)

		_, err3 := mockExecutor.Execute(ctx, "fail")
		assert.Error(t, err3)

		mockExecutor.AssertExpectations(t)
	})

	t.Run("any argument matching", func(t *testing.T) {
		mockExecutor := new(mocks.MockGHCommandExecutor)

		// mock.Anythingを使用した柔軟なマッチング
		mockExecutor.On("Execute", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return("matched\n", nil)

		output, err := mockExecutor.Execute(context.Background(), "any-command", "arg1", "arg2")
		assert.NoError(t, err)
		assert.Equal(t, "matched\n", output)
		mockExecutor.AssertExpectations(t)
	})
}
