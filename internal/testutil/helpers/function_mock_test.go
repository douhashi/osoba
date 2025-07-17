package helpers_test

import (
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/stretchr/testify/assert"
)

func TestFunctionMocker(t *testing.T) {
	t.Run("単一の関数をモック", func(t *testing.T) {
		// オリジナルの関数
		var testFunc func() string
		testFunc = func() string {
			return "original"
		}

		// モッカーを作成
		mocker := helpers.NewFunctionMocker()

		// 関数をモック
		mocker.MockFunc(&testFunc, func() string {
			return "mocked"
		})

		// モックされた関数の動作を確認
		assert.Equal(t, "mocked", testFunc())

		// 元に戻す
		mocker.Restore()
		assert.Equal(t, "original", testFunc())
	})

	t.Run("複数の関数をモック", func(t *testing.T) {
		// オリジナルの関数
		var func1 func() string
		var func2 func(int) int

		func1 = func() string {
			return "original1"
		}
		func2 = func(n int) int {
			return n * 2
		}

		// モッカーを作成
		mocker := helpers.NewFunctionMocker()

		// 複数の関数をモック
		mocker.MockFunc(&func1, func() string {
			return "mocked1"
		})
		mocker.MockFunc(&func2, func(n int) int {
			return n * 3
		})

		// モックされた関数の動作を確認
		assert.Equal(t, "mocked1", func1())
		assert.Equal(t, 15, func2(5))

		// 元に戻す
		mocker.Restore()
		assert.Equal(t, "original1", func1())
		assert.Equal(t, 10, func2(5))
	})

	t.Run("エラーを返す関数のモック", func(t *testing.T) {
		// オリジナルの関数
		var errorFunc func() error
		errorFunc = func() error {
			return nil
		}

		// モッカーを作成
		mocker := helpers.NewFunctionMocker()

		// エラーを返すようにモック
		expectedErr := errors.New("mocked error")
		mocker.MockFunc(&errorFunc, func() error {
			return expectedErr
		})

		// モックされた関数の動作を確認
		assert.Equal(t, expectedErr, errorFunc())

		// 元に戻す
		mocker.Restore()
		assert.NoError(t, errorFunc())
	})

	t.Run("defer でのRestore", func(t *testing.T) {
		var testFunc func() string
		testFunc = func() string {
			return "original"
		}

		// サブ関数内でモックを使用
		func() {
			mocker := helpers.NewFunctionMocker()
			defer mocker.Restore() // deferで自動的に元に戻す

			mocker.MockFunc(&testFunc, func() string {
				return "mocked"
			})

			assert.Equal(t, "mocked", testFunc())
		}()

		// サブ関数を抜けた後は元に戻っている
		assert.Equal(t, "original", testFunc())
	})

	t.Run("RestoreAllの使用", func(t *testing.T) {
		var func1 func() string
		var func2 func() string

		func1 = func() string { return "original1" }
		func2 = func() string { return "original2" }

		// 複数のモッカーを作成
		mocker1 := helpers.NewFunctionMocker()
		mocker2 := helpers.NewFunctionMocker()

		mocker1.MockFunc(&func1, func() string { return "mocked1" })
		mocker2.MockFunc(&func2, func() string { return "mocked2" })

		// モックされていることを確認
		assert.Equal(t, "mocked1", func1())
		assert.Equal(t, "mocked2", func2())

		// すべてのモッカーを元に戻す
		helpers.RestoreAll()

		// 元に戻っていることを確認
		assert.Equal(t, "original1", func1())
		assert.Equal(t, "original2", func2())
	})

	t.Run("チェイン可能な操作", func(t *testing.T) {
		var func1 func() string
		var func2 func() int

		func1 = func() string { return "original1" }
		func2 = func() int { return 1 }

		// チェインでモック
		mocker := helpers.NewFunctionMocker().
			MockFunc(&func1, func() string { return "mocked1" }).
			MockFunc(&func2, func() int { return 2 })

		// モックされていることを確認
		assert.Equal(t, "mocked1", func1())
		assert.Equal(t, 2, func2())

		// 元に戻す
		mocker.Restore()
		assert.Equal(t, "original1", func1())
		assert.Equal(t, 1, func2())
	})

	t.Run("nilチェック", func(t *testing.T) {
		var testFunc func() string

		// モッカーを作成
		mocker := helpers.NewFunctionMocker()

		// nilの関数をモックしようとしてもパニックしない
		assert.NotPanics(t, func() {
			mocker.MockFunc(&testFunc, func() string {
				return "mocked"
			})
		})

		// モック後は関数が使える
		assert.Equal(t, "mocked", testFunc())
	})
}
