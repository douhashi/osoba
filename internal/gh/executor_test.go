package gh

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCommandExecutor_Execute(t *testing.T) {
	tests := []struct {
		name             string
		setupExecutor    func() CommandExecutor
		ctx              context.Context
		command          string
		args             []string
		expectedOutput   string
		expectedError    error
		expectedExitCode int
	}{
		{
			name: "正常系: コマンドが正常に実行される",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "echo" && len(args) == 1 && args[0] == "hello" {
							return "hello\n", nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			ctx:            context.Background(),
			command:        "echo",
			args:           []string{"hello"},
			expectedOutput: "hello\n",
			expectedError:  nil,
		},
		{
			name: "異常系: コマンドが失敗する",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", &ExecError{
							Command:  command,
							Args:     args,
							ExitCode: 1,
							Stderr:   "command not found",
						}
					},
				}
			},
			ctx:            context.Background(),
			command:        "invalid-command",
			args:           []string{},
			expectedOutput: "",
			expectedError:  &ExecError{Command: "invalid-command", Args: []string{}, ExitCode: 1, Stderr: "command not found"},
		},
		{
			name: "異常系: コンテキストがキャンセルされる",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						select {
						case <-ctx.Done():
							return "", ctx.Err()
						case <-time.After(1 * time.Second):
							return "should not reach here", nil
						}
					},
				}
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			command:        "sleep",
			args:           []string{"10"},
			expectedOutput: "",
			expectedError:  context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			output, err := executor.Execute(tt.ctx, tt.command, tt.args...)

			if output != tt.expectedOutput {
				t.Errorf("Execute() output = %v, want %v", output, tt.expectedOutput)
			}

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("Execute() error = nil, want %v", tt.expectedError)
				} else if execErr, ok := tt.expectedError.(*ExecError); ok {
					gotExecErr, ok := err.(*ExecError)
					if !ok {
						t.Errorf("Execute() error type = %T, want *ExecError", err)
					} else if gotExecErr.ExitCode != execErr.ExitCode {
						t.Errorf("Execute() ExitCode = %v, want %v", gotExecErr.ExitCode, execErr.ExitCode)
					}
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("Execute() error = %v, want %v", err, tt.expectedError)
				}
			} else if err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
			}
		})
	}
}

func TestRealCommandExecutor_Execute(t *testing.T) {
	executor := NewRealCommandExecutor()
	ctx := context.Background()

	t.Run("正常系: echoコマンドの実行", func(t *testing.T) {
		output, err := executor.Execute(ctx, "echo", "test")
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output != "test\n" {
			t.Errorf("Execute() = %v, want %v", output, "test\n")
		}
	})

	t.Run("異常系: 存在しないコマンド", func(t *testing.T) {
		_, err := executor.Execute(ctx, "this-command-does-not-exist")
		if err == nil {
			t.Error("Execute() error = nil, want error")
		}
		execErr, ok := err.(*ExecError)
		if !ok {
			t.Errorf("Execute() error type = %T, want *ExecError", err)
		} else if execErr.ExitCode == 0 {
			t.Error("Execute() ExitCode = 0, want non-zero")
		}
	})
}

func TestExecError_Error(t *testing.T) {
	err := &ExecError{
		Command:  "test-cmd",
		Args:     []string{"arg1", "arg2"},
		ExitCode: 127,
		Stderr:   "command not found",
	}

	expected := "command 'test-cmd arg1 arg2' failed with exit code 127: command not found"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}
