package gh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandExecutor はコマンド実行の抽象化インターフェース
type CommandExecutor interface {
	Execute(ctx context.Context, command string, args ...string) (string, error)
}

// ExecError はコマンド実行エラーを表す
type ExecError struct {
	Command  string
	Args     []string
	ExitCode int
	Stderr   string
}

// Error はエラーメッセージを返す
func (e *ExecError) Error() string {
	cmdStr := e.Command
	if len(e.Args) > 0 {
		cmdStr = fmt.Sprintf("%s %s", e.Command, strings.Join(e.Args, " "))
	}
	return fmt.Sprintf("command '%s' failed with exit code %d: %s", cmdStr, e.ExitCode, e.Stderr)
}

// RealCommandExecutor は実際のコマンドを実行する実装
type RealCommandExecutor struct{}

// NewRealCommandExecutor は新しいRealCommandExecutorを作成する
func NewRealCommandExecutor() CommandExecutor {
	return &RealCommandExecutor{}
}

// Execute はコマンドを実行し、標準出力を返す
func (r *RealCommandExecutor) Execute(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		exitCode := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
		return "", &ExecError{
			Command:  command,
			Args:     args,
			ExitCode: exitCode,
			Stderr:   stderr.String(),
		}
	}

	return stdout.String(), nil
}

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

// Executor はghコマンドの実行インターフェース
type Executor interface {
	Execute(ctx context.Context, args []string) ([]byte, error)
}

// GHExecutor はghコマンドを実行する構造体
type GHExecutor struct {
	cmdExecutor CommandExecutor
}

// NewExecutor は新しいGHExecutorを作成する
func NewExecutor() Executor {
	return &GHExecutor{
		cmdExecutor: NewRealCommandExecutor(),
	}
}

// Execute はghコマンドを実行する
func (e *GHExecutor) Execute(ctx context.Context, args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	output, err := e.cmdExecutor.Execute(ctx, "gh", args...)
	if err != nil {
		return nil, err
	}

	return []byte(output), nil
}
