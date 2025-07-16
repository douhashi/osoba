package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/utils"
)

// TestGitHubRepoInfoConsistency は各コマンドでのGitHubリポジトリ情報取得の一貫性をテストする
func TestGitHubRepoInfoConsistency(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		expectOwner string
		expectRepo  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "正常系: HTTPS URL",
			remoteURL:   "https://github.com/douhashi/osoba.git",
			expectOwner: "douhashi",
			expectRepo:  "osoba",
			expectError: false,
		},
		{
			name:        "正常系: SSH URL",
			remoteURL:   "git@github.com:douhashi/osoba.git",
			expectOwner: "douhashi",
			expectRepo:  "osoba",
			expectError: false,
		},
		{
			name:        "正常系: HTTPS URL without .git",
			remoteURL:   "https://github.com/douhashi/osoba",
			expectOwner: "douhashi",
			expectRepo:  "osoba",
			expectError: false,
		},
		{
			name:        "エラー系: 不正なURL",
			remoteURL:   "invalid-url",
			expectError: true,
			errorMsg:    "GitHub URL解析に失敗",
		},
		{
			name:        "エラー系: GitHub以外のURL",
			remoteURL:   "https://gitlab.com/user/repo.git",
			expectError: true,
			errorMsg:    "invalid GitHub URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// init.goの実装をテスト
			t.Run("init.go parseGitHubURL", func(t *testing.T) {
				owner, repo := parseGitHubURL(tt.remoteURL)

				if tt.expectError {
					if owner != "" || repo != "" {
						t.Errorf("parseGitHubURL() should fail for %s, got owner=%s, repo=%s", tt.remoteURL, owner, repo)
					}
				} else {
					if owner != tt.expectOwner || repo != tt.expectRepo {
						t.Errorf("parseGitHubURL() owner=%s, repo=%s, want owner=%s, repo=%s", owner, repo, tt.expectOwner, tt.expectRepo)
					}
				}
			})

			// status.goの実装をテスト（utils.ParseGitHubURL）
			t.Run("status.go utils.ParseGitHubURL", func(t *testing.T) {
				// internal/utilsのParseGitHubURLをテスト
				// ここでは直接呼び出してテスト
				testParseGitHubURL(t, tt.remoteURL, tt.expectOwner, tt.expectRepo, tt.expectError, tt.errorMsg)
			})
		})
	}
}

// TestGitHubRepoInfoRetrievalErrors は各コマンドでのエラーハンドリングをテストする
func TestGitHubRepoInfoRetrievalErrors(t *testing.T) {
	tests := []struct {
		name             string
		setupMocks       func()
		testInitCmd      bool
		testStatusCmd    bool
		expectErrorMsg   string
		expectWarningMsg string
	}{
		{
			name: "エラー系: git remote get-url失敗",
			setupMocks: func() {
				getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
					return nil, &utils.GetGitHubRepoInfoError{
						Step:    "remote_url",
						Cause:   errors.New("fatal: No such remote 'origin'"),
						Message: "リモートURL取得に失敗しました",
					}
				}
			},
			testInitCmd:      true,
			expectWarningMsg: "リモートURL取得に失敗しました",
		},
		{
			name: "エラー系: 不正なGitHub URL",
			setupMocks: func() {
				getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
					return nil, &utils.GetGitHubRepoInfoError{
						Step:    "url_parsing",
						Cause:   errors.New("invalid URL format"),
						Message: "GitHubリポジトリ情報の解析に失敗しました",
					}
				}
			},
			testInitCmd:      true,
			expectWarningMsg: "GitHubリポジトリ情報の解析に失敗しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モック関数を保存
			origGetRemoteURL := getRemoteURLFunc
			origIsGitRepo := isGitRepositoryFunc
			origCheckCommand := checkCommandFunc
			origGetEnv := getEnvFunc
			origWriteFile := writeFileFunc
			origMkdirAll := mkdirAllFunc
			origGitHubClient := createGitHubClientFunc
			origGetGitHubRepoInfo := getGitHubRepoInfoFunc
			defer func() {
				getRemoteURLFunc = origGetRemoteURL
				isGitRepositoryFunc = origIsGitRepo
				checkCommandFunc = origCheckCommand
				getEnvFunc = origGetEnv
				writeFileFunc = origWriteFile
				mkdirAllFunc = origMkdirAll
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
			getEnvFunc = func(key string) string {
				if key == "GITHUB_TOKEN" || key == "OSOBA_GITHUB_TOKEN" {
					return "test-token"
				}
				return ""
			}
			writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
				return nil
			}
			mkdirAllFunc = func(path string, perm os.FileMode) error {
				return nil
			}
			createGitHubClientFunc = func(token string) githubInterface {
				return &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						return nil
					},
				}
			}
			// デフォルトのgetGitHubRepoInfoFuncモック
			getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
				return &utils.GitHubRepoInfo{
					Owner: "douhashi",
					Repo:  "osoba",
				}, nil
			}

			// テスト固有のモックを設定
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			if tt.testInitCmd {
				t.Run("init command", func(t *testing.T) {
					buf := new(bytes.Buffer)
					errBuf := new(bytes.Buffer)

					rootCmd := newRootCmd()
					rootCmd.AddCommand(newInitCmd())
					rootCmd.SetOut(buf)
					rootCmd.SetErr(errBuf)
					rootCmd.SetArgs([]string{"init"})

					err := rootCmd.Execute()

					if tt.expectErrorMsg != "" && err != nil {
						if !strings.Contains(err.Error(), tt.expectErrorMsg) {
							t.Errorf("Expected error to contain %q, got %v", tt.expectErrorMsg, err)
						}
					}

					if tt.expectWarningMsg != "" {
						output := buf.String() + errBuf.String()
						if !strings.Contains(output, tt.expectWarningMsg) {
							t.Errorf("Expected output to contain warning %q, got %s", tt.expectWarningMsg, output)
						}
					}
				})
			}

			if tt.testStatusCmd {
				t.Run("status command", func(t *testing.T) {
					buf := new(bytes.Buffer)

					rootCmd := newRootCmd()
					rootCmd.AddCommand(newStatusCmd())
					rootCmd.SetOut(buf)
					rootCmd.SetErr(buf)
					rootCmd.SetArgs([]string{"status"})

					err := rootCmd.Execute()

					if tt.expectErrorMsg != "" && err != nil {
						if !strings.Contains(err.Error(), tt.expectErrorMsg) {
							t.Errorf("Expected error to contain %q, got %v", tt.expectErrorMsg, err)
						}
					}

					if tt.expectWarningMsg != "" {
						output := buf.String()
						if !strings.Contains(output, tt.expectWarningMsg) {
							t.Errorf("Expected output to contain warning %q, got %s", tt.expectWarningMsg, output)
						}
					}
				})
			}
		})
	}
}

// testParseGitHubURL は utils.ParseGitHubURL をテストするヘルパー関数
func testParseGitHubURL(t *testing.T, url, expectOwner, expectRepo string, expectError bool, expectErrorMsg string) {
	// ここでinternal/utilsのParseGitHubURLを直接呼び出したいが、
	// テストファイルからimportする必要がある
	// 実際の修正時に適切にテストされるよう、この関数はプレースホルダーとしておく
	t.Logf("Testing URL parsing for: %s", url)

	// 実際のテストは utils パッケージのテストで行う
	// このテストは統合テストとして、各コマンドが同じ結果を返すことを確認するために使用
}

// TestErrorMessagesSpecificity は具体的なエラーメッセージの要求をテストする
func TestErrorMessagesSpecificity(t *testing.T) {
	tests := []struct {
		name         string
		scenario     string
		setupMocks   func()
		expectedMsgs []string
	}{
		{
			name:     "GitHubトークン未設定",
			scenario: "GitHub API token not configured",
			setupMocks: func() {
				getEnvFunc = func(key string) string {
					return "" // トークンなし
				}
			},
			expectedMsgs: []string{
				"GitHub Personal Access Tokenが設定されていません",
				"export GITHUB_TOKEN=",
			},
		},
		{
			name:     "リモートURL取得失敗",
			scenario: "git remote get-url fails",
			setupMocks: func() {
				getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
					return nil, &utils.GetGitHubRepoInfoError{
						Step:    "remote_url",
						Cause:   fmt.Errorf("fatal: No such remote 'origin'"),
						Message: "リモートURL取得に失敗しました",
					}
				}
			},
			expectedMsgs: []string{
				"リモートURL取得に失敗しました",
			},
		},
		{
			name:     "URL解析失敗",
			scenario: "GitHub URL parsing fails",
			setupMocks: func() {
				getGitHubRepoInfoFunc = func(ctx context.Context) (*utils.GitHubRepoInfo, error) {
					return nil, &utils.GetGitHubRepoInfoError{
						Step:    "url_parsing",
						Cause:   fmt.Errorf("invalid GitHub URL format"),
						Message: "GitHubリポジトリ情報の解析に失敗しました",
					}
				}
			},
			expectedMsgs: []string{
				"GitHubリポジトリ情報の解析に失敗しました",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト実装は修正作業時に追加
			t.Logf("Testing scenario: %s", tt.scenario)

			// この段階では、どのようなエラーメッセージが期待されるかを
			// ドキュメント化することが主目的
			for _, msg := range tt.expectedMsgs {
				t.Logf("Expected message: %s", msg)
			}
		})
	}
}
