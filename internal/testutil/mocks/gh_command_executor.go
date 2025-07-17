package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockGHCommandExecutor は gh.CommandExecutor インターフェースのモック実装
type MockGHCommandExecutor struct {
	mock.Mock
}

// Execute はコマンドを実行する
func (m *MockGHCommandExecutor) Execute(ctx context.Context, command string, args ...string) (string, error) {
	// 可変長引数を配列に変換してテストしやすくする
	argsList := make([]string, len(args))
	copy(argsList, args)

	ret := m.Called(ctx, command, argsList)
	return ret.String(0), ret.Error(1)
}

// WithCommandOutput は成功するコマンドの期待値を設定
func (m *MockGHCommandExecutor) WithCommandOutput(ctx context.Context, command string, args []string, output string) *MockGHCommandExecutor {
	m.On("Execute", ctx, command, args).Return(output, nil)
	return m
}

// WithCommandError はエラーを返すコマンドの期待値を設定
func (m *MockGHCommandExecutor) WithCommandError(ctx context.Context, command string, args []string, err error) *MockGHCommandExecutor {
	m.On("Execute", ctx, command, args).Return("", err)
	return m
}

// WithDefaultBehavior はデフォルトの動作を設定（すべてのコマンドが成功）
func (m *MockGHCommandExecutor) WithDefaultBehavior() *MockGHCommandExecutor {
	m.On("Execute", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Maybe().Return("", nil)
	return m
}
