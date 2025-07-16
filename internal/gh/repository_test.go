package gh

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestClient_GetRepository(t *testing.T) {
	tests := []struct {
		name          string
		owner         string
		repo          string
		setupExecutor func() CommandExecutor
		wantName      string
		wantOwner     string
		wantPrivate   bool
		wantErr       bool
		expectedError string
	}{
		{
			name:  "正常系: リポジトリ情報を取得できる",
			owner: "douhashi",
			repo:  "osoba",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						if command == "gh" && len(args) >= 3 && args[0] == "repo" && args[1] == "view" {
							repoData := ghRepository{
								Name: "osoba",
								Owner: ghOwner{
									Login: "douhashi",
								},
								Description: "自律的ソフトウェア開発支援ツール",
								DefaultBranchRef: ghBranchRef{
									Name: "main",
								},
								IsPrivate:  false,
								CreatedAt:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt:  time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
								URL:        "https://github.com/douhashi/osoba",
								SSHURL:     "git@github.com:douhashi/osoba.git",
								IsArchived: false,
								IsFork:     false,
							}
							jsonData, _ := json.Marshal(repoData)
							return string(jsonData), nil
						}
						return "", errors.New("unexpected command")
					},
				}
			},
			wantName:    "osoba",
			wantOwner:   "douhashi",
			wantPrivate: false,
			wantErr:     false,
		},
		{
			name:  "異常系: リポジトリが見つからない",
			owner: "invalid",
			repo:  "notfound",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "", &ExecError{
							Command:  "gh",
							Args:     args,
							ExitCode: 1,
							Stderr:   "Could not resolve to a Repository",
						}
					},
				}
			},
			wantErr:       true,
			expectedError: "repository not found",
		},
		{
			name:  "異常系: ownerが空",
			owner: "",
			repo:  "osoba",
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
			name:  "異常系: repoが空",
			owner: "douhashi",
			repo:  "",
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
			name:  "異常系: JSONパースエラー",
			owner: "douhashi",
			repo:  "osoba",
			setupExecutor: func() CommandExecutor {
				return &MockCommandExecutor{
					ExecuteFunc: func(ctx context.Context, command string, args ...string) (string, error) {
						return "invalid json", nil
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

			got, err := client.GetRepository(context.Background(), tt.owner, tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.expectedError != "" && err != nil && err.Error() != tt.expectedError {
					t.Errorf("GetRepository() error = %v, expectedError %v", err.Error(), tt.expectedError)
				}
				return
			}

			if got == nil {
				t.Error("GetRepository() returned nil repository")
				return
			}

			if got.Name == nil || *got.Name != tt.wantName {
				t.Errorf("GetRepository() Name = %v, want %v", got.Name, tt.wantName)
			}
			if got.Owner == nil || got.Owner.Login == nil || *got.Owner.Login != tt.wantOwner {
				t.Errorf("GetRepository() Owner.Login = %v, want %v", got.Owner, tt.wantOwner)
			}
			if got.Private == nil || *got.Private != tt.wantPrivate {
				t.Errorf("GetRepository() Private = %v, want %v", got.Private, tt.wantPrivate)
			}
		})
	}
}
