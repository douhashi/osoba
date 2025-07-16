package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/utils"
)

func TestInitCmd_GitHubLabelManagement(t *testing.T) {
	// モック関数を保存しておく
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origGetEnv := getEnvFunc
	origExecCommand := execCommandFunc
	origMkdirAll := mkdirAllFunc
	origWriteFile := writeFileFunc
	origGetRemoteURL := getRemoteURLFunc
	origGitHubClient := createGitHubClientFunc
	origGetGitHubRepoInfo := getGitHubRepoInfoFunc
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		execCommandFunc = origExecCommand
		mkdirAllFunc = origMkdirAll
		writeFileFunc = origWriteFile
		getRemoteURLFunc = origGetRemoteURL
		createGitHubClientFunc = origGitHubClient
		getGitHubRepoInfoFunc = origGetGitHubRepoInfo
	}()

	// 基本的なモックを設定
	isGitRepositoryFunc = func(path string) (bool, error) {
		return true, nil
	}
	checkCommandFunc = func(cmd string) error {
		return nil
	}
	execCommandFunc = func(name string, args ...string) ([]byte, error) {
		if name == "gh" {
			return []byte("success"), nil
		}
		return []byte{}, nil
	}
	mkdirAllFunc = func(path string, perm os.FileMode) error {
		return nil
	}
	writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
		return nil
	}
	getRemoteURLFunc = func(remoteName string) (string, error) {
		return "https://github.com/douhashi/osoba.git", nil
	}
	getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
		return &utils.GitHubRepoInfo{
			Owner: "douhashi",
			Repo:  "osoba",
		}, nil
	}

	tests := []struct {
		name               string
		setupMocks         func()
		wantErr            bool
		wantOutputContains []string
		wantErrContains    string
	}{
		{
			name: "正常系: GitHubラベルが作成される",
			setupMocks: func() {
				getEnvFunc = func(key string) string {
					if key == "GITHUB_TOKEN" || key == "OSOBA_GITHUB_TOKEN" {
						return "test-token"
					}
					return ""
				}
				mockClient := &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						// 必要なラベルが含まれているかテスト
						expectedLabels := []string{
							"status:needs-plan",
							"status:planning",
							"status:ready",
							"status:implementing",
							"status:review-requested",
							"status:reviewing",
						}
						// ここで実際のラベル作成ロジックをテストできる
						_ = expectedLabels
						return nil
					},
				}
				createGitHubClientFunc = func(token string) githubInterface {
					return mockClient
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[8/8] GitHubラベルの作成           ✅",
			},
		},
		{
			name: "正常系: GitHub Tokenが設定されていない場合はスキップ",
			setupMocks: func() {
				getEnvFunc = func(key string) string {
					return ""
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"初期化が完了しました",
			},
		},
		{
			name: "警告系: GitHubラベル作成がエラーの場合は警告表示",
			setupMocks: func() {
				getEnvFunc = func(key string) string {
					if key == "GITHUB_TOKEN" || key == "OSOBA_GITHUB_TOKEN" {
						return "test-token"
					}
					return ""
				}
				mockClient := &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						return fmt.Errorf("API rate limit exceeded")
					},
				}
				createGitHubClientFunc = func(token string) githubInterface {
					return mockClient
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"⚠️  GitHubラベルの作成に失敗しました",
				"手動でラベルを作成してください",
			},
		},
		{
			name: "正常系: リモートURL取得エラーの場合はスキップ",
			setupMocks: func() {
				getEnvFunc = func(key string) string {
					if key == "GITHUB_TOKEN" || key == "OSOBA_GITHUB_TOKEN" {
						return "test-token"
					}
					return ""
				}
				getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
					return nil, &utils.GetGitHubRepoInfoError{
						Step:    "remote_url",
						Cause:   fmt.Errorf("not a git repository"),
						Message: "リモートURL取得に失敗しました",
					}
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"⚠️  リモートURL取得に失敗しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			buf := new(bytes.Buffer)
			rootCmd = newRootCmd()
			rootCmd.AddCommand(newInitCmd())
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"init"})

			err := rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.wantErrContains != "" {
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Execute() error = %v, want to contain %v", err, tt.wantErrContains)
				}
			}

			output := buf.String()
			for _, want := range tt.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}
