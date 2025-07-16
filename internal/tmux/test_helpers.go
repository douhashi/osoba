package tmux

import "os/exec"

// MockExitError はテスト用のExitErrorモック
type MockExitError struct {
	ExitCode int
}

// Error はerrorインターフェースを満たす
func (e *MockExitError) Error() string {
	return "exit status 1"
}

// GetExitCode は終了コードを返す
func (e *MockExitError) GetExitCode() int {
	return e.ExitCode
}

// IsExitError はerrorがExitErrorかどうかをチェックする関数
// テストで置き換え可能にするため
var IsExitError = func(err error) (int, bool) {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), true
	}
	// テスト用のMockExitErrorもチェック
	if mockErr, ok := err.(*MockExitError); ok {
		return mockErr.GetExitCode(), true
	}
	return 0, false
}
