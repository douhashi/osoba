package gh

import (
	"context"
)

// MockCommandExecutor はテスト用のモック実装
type MockCommandExecutor struct {
	ExecuteFunc func(ctx context.Context, command string, args ...string) (string, error)
}

// Execute はモック関数を呼び出す
func (m *MockCommandExecutor) Execute(ctx context.Context, command string, args ...string) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, command, args...)
	}
	return "", nil
}
