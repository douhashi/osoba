package gh

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestClient_CreateIssueComment(t *testing.T) {
	tests := []struct {
		name          string
		owner         string
		repo          string
		issueNumber   int
		comment       string
		setupExecutor func() CommandExecutor
		wantErr       bool
		expectedError string
	}{
		{
			name:        "正常系: コメントが投稿できる",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			comment:     "テストコメントです。",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) >= 5 &&
							args[0] == "issue" && args[1] == "comment" &&
							args[2] == "123" && args[3] == "--body" && args[4] == "テストコメントです。" &&
							args[5] == "--repo" && args[6] == "douhashi/osoba" {
							return "https://github.com/douhashi/osoba/issues/123#issuecomment-1234567890", nil
						}
						return "", fmt.Errorf("unexpected command: %s %v", command, args)
					},
				}
			},
			wantErr: false,
		},
		{
			name:        "異常系: ownerが空",
			owner:       "",
			repo:        "osoba",
			issueNumber: 123,
			comment:     "test",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", nil
					},
				}
			},
			wantErr:       true,
			expectedError: "owner is required",
		},
		{
			name:        "異常系: repoが空",
			owner:       "douhashi",
			repo:        "",
			issueNumber: 123,
			comment:     "test",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", nil
					},
				}
			},
			wantErr:       true,
			expectedError: "repo is required",
		},
		{
			name:        "異常系: issueNumberが不正",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 0,
			comment:     "test",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", nil
					},
				}
			},
			wantErr:       true,
			expectedError: "issue number must be positive",
		},
		{
			name:        "異常系: commentが空",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			comment:     "",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", nil
					},
				}
			},
			wantErr:       true,
			expectedError: "comment is required",
		},
		{
			name:        "異常系: Issueが見つからない",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 999,
			comment:     "test",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", &ExecError{
							Command:  "gh",
							Args:     args,
							ExitCode: 1,
							Stderr:   "GraphQL: Could not resolve to an Issue",
						}
					},
				}
			},
			wantErr:       true,
			expectedError: "issue not found",
		},
		{
			name:        "異常系: コマンド実行エラー",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			comment:     "test",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", errors.New("network error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := tt.setupExecutor()
			client, err := NewClient(executor)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.CreateIssueComment(context.Background(), tt.owner, tt.repo, tt.issueNumber, tt.comment)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIssueComment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.expectedError != "" && err != nil && err.Error() != tt.expectedError {
				t.Errorf("CreateIssueComment() error = %v, expectedError %v", err.Error(), tt.expectedError)
			}
		})
	}
}
