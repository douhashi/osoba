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

func TestClient_RemoveLabel(t *testing.T) {
	tests := []struct {
		name          string
		setupExecutor func() CommandExecutor
		owner         string
		repo          string
		issueNumber   int
		label         string
		wantErr       bool
		expectedError string
	}{
		{
			name: "正常系: ラベルが正常に削除される",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 7 && args[0] == "issue" && args[1] == "edit" && args[2] == "123" && args[3] == "--repo" && args[4] == "owner/repo" && args[5] == "--remove-label" && args[6] == "bug" {
							return "", nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			owner:       "owner",
			repo:        "repo",
			issueNumber: 123,
			label:       "bug",
			wantErr:     false,
		},
		{
			name: "異常系: ghコマンドが失敗",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) >= 3 && args[0] == "issue" && args[1] == "edit" {
							return "", &ExecError{
								Command:  "gh",
								Args:     args,
								ExitCode: 1,
								Stderr:   "issue not found",
							}
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			owner:       "owner",
			repo:        "repo",
			issueNumber: 123,
			label:       "bug",
			wantErr:     true,
		},
		{
			name: "異常系: ownerが空",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "",
			repo:          "repo",
			issueNumber:   123,
			label:         "bug",
			wantErr:       true,
			expectedError: "owner is required",
		},
		{
			name: "異常系: repoが空",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "owner",
			repo:          "",
			issueNumber:   123,
			label:         "bug",
			wantErr:       true,
			expectedError: "repo is required",
		},
		{
			name: "異常系: issueNumberが無効",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "owner",
			repo:          "repo",
			issueNumber:   0,
			label:         "bug",
			wantErr:       true,
			expectedError: "issue number must be positive",
		},
		{
			name: "異常系: labelが空",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "owner",
			repo:          "repo",
			issueNumber:   123,
			label:         "",
			wantErr:       true,
			expectedError: "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			client, err := NewClient(executor)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.RemoveLabel(context.Background(), tt.owner, tt.repo, tt.issueNumber, tt.label)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.expectedError != "" && err.Error() != tt.expectedError {
				t.Errorf("RemoveLabel() error = %v, expectedError %v", err.Error(), tt.expectedError)
			}
		})
	}
}

func TestClient_AddLabel(t *testing.T) {
	tests := []struct {
		name          string
		setupExecutor func() CommandExecutor
		owner         string
		repo          string
		issueNumber   int
		label         string
		wantErr       bool
		expectedError string
	}{
		{
			name: "正常系: ラベルが正常に追加される",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) == 7 && args[0] == "issue" && args[1] == "edit" && args[2] == "123" && args[3] == "--repo" && args[4] == "owner/repo" && args[5] == "--add-label" && args[6] == "bug" {
							return "", nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			owner:       "owner",
			repo:        "repo",
			issueNumber: 123,
			label:       "bug",
			wantErr:     false,
		},
		{
			name: "異常系: ghコマンドが失敗",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) >= 3 && args[0] == "issue" && args[1] == "edit" {
							return "", &ExecError{
								Command:  "gh",
								Args:     args,
								ExitCode: 1,
								Stderr:   "issue not found",
							}
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			owner:       "owner",
			repo:        "repo",
			issueNumber: 123,
			label:       "bug",
			wantErr:     true,
		},
		{
			name: "異常系: ownerが空",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "",
			repo:          "repo",
			issueNumber:   123,
			label:         "bug",
			wantErr:       true,
			expectedError: "owner is required",
		},
		{
			name: "異常系: repoが空",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "owner",
			repo:          "",
			issueNumber:   123,
			label:         "bug",
			wantErr:       true,
			expectedError: "repo is required",
		},
		{
			name: "異常系: issueNumberが無効",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "owner",
			repo:          "repo",
			issueNumber:   0,
			label:         "bug",
			wantErr:       true,
			expectedError: "issue number must be positive",
		},
		{
			name: "異常系: labelが空",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{}
			},
			owner:         "owner",
			repo:          "repo",
			issueNumber:   123,
			label:         "",
			wantErr:       true,
			expectedError: "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			client, err := NewClient(executor)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.AddLabel(context.Background(), tt.owner, tt.repo, tt.issueNumber, tt.label)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.expectedError != "" && err.Error() != tt.expectedError {
				t.Errorf("AddLabel() error = %v, expectedError %v", err.Error(), tt.expectedError)
			}
		})
	}
}
