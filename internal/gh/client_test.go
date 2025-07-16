package gh

import (
	"context"
	"errors"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name          string
		executor      CommandExecutor
		wantErr       bool
		expectedError string
	}{
		{
			name:     "正常系: Clientが作成される",
			executor: NewRealCommandExecutor(),
			wantErr:  false,
		},
		{
			name:          "異常系: executorがnil",
			executor:      nil,
			wantErr:       true,
			expectedError: "executor is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.executor)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedError {
				t.Errorf("NewClient() error = %v, expectedError %v", err.Error(), tt.expectedError)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewClient() returned nil client")
			}
			if !tt.wantErr && got.executor != tt.executor {
				t.Error("NewClient() did not set executor correctly")
			}
		})
	}
}

func TestClient_ValidatePrerequisites(t *testing.T) {
	tests := []struct {
		name          string
		setupExecutor func() CommandExecutor
		wantErr       bool
		expectedError string
	}{
		{
			name: "正常系: ghコマンドがインストール済みかつ認証済み",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 1 && args[0] == "--version" {
							return "gh version 2.32.1\n", nil
						}
						if command == "gh" && len(args) == 2 && args[0] == "auth" && args[1] == "status" {
							return "Logged in to github.com\n", nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: ghコマンドがインストールされていない",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 1 && args[0] == "--version" {
							return "", &ExecError{
								Command:  "gh",
								Args:     []string{"--version"},
								ExitCode: 127,
								Stderr:   "command not found",
							}
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			wantErr:       true,
			expectedError: "gh command is not installed",
		},
		{
			name: "異常系: ghコマンドが未認証",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 1 && args[0] == "--version" {
							return "gh version 2.32.1\n", nil
						}
						if command == "gh" && len(args) == 2 && args[0] == "auth" && args[1] == "status" {
							return "", &ExecError{
								Command:  "gh",
								Args:     []string{"auth", "status"},
								ExitCode: 1,
								Stderr:   "You are not logged into any GitHub hosts.",
							}
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			wantErr:       true,
			expectedError: "gh command is not authenticated. Run 'gh auth login' first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			client, err := NewClient(executor)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.ValidatePrerequisites(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrerequisites() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedError {
				t.Errorf("ValidatePrerequisites() error = %v, expectedError %v", err.Error(), tt.expectedError)
			}
		})
	}
}
