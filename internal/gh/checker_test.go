package gh

import (
	"context"
	"errors"
	"testing"
)

func TestCheckInstalled(t *testing.T) {
	tests := []struct {
		name          string
		setupExecutor func() CommandExecutor
		want          bool
		wantErr       bool
	}{
		{
			name: "正常系: ghコマンドがインストールされている",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 1 && args[0] == "--version" {
							return "gh version 2.32.1 (2023-07-24)\nhttps://github.com/cli/cli/releases/tag/v2.32.1\n", nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "異常系: ghコマンドがインストールされていない",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", &ExecError{
							Command:  "gh",
							Args:     []string{"--version"},
							ExitCode: 127,
							Stderr:   "command not found: gh",
						}
					},
				}
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "異常系: コマンド実行でエラーが発生（ExecError以外）",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", errors.New("unexpected error")
					},
				}
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			got, err := CheckInstalled(context.Background(), executor)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckAuth(t *testing.T) {
	tests := []struct {
		name          string
		setupExecutor func() CommandExecutor
		want          bool
		wantErr       bool
	}{
		{
			name: "正常系: 認証済み",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 2 && args[0] == "auth" && args[1] == "status" {
							return "github.com\n  ✓ Logged in to github.com as username (/Users/username/.config/gh/hosts.yml)\n", nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "正常系: 未認証",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", &ExecError{
							Command:  "gh",
							Args:     []string{"auth", "status"},
							ExitCode: 1,
							Stderr:   "You are not logged into any GitHub hosts. Run gh auth login to authenticate.",
						}
					},
				}
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "異常系: コマンド実行でエラーが発生（ExecError以外）",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", errors.New("unexpected error")
					},
				}
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			got, err := CheckAuth(context.Background(), executor)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckAuth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}
