package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/utils"
)

func TestInitCmd(t *testing.T) {
	// モック関数を保存しておく
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origGetEnv := getEnvFunc
	origWriteFile := writeFileFunc
	origMkdirAll := mkdirAllFunc
	origGitHubClient := createGitHubClientFunc
	origGetRemoteURL := getRemoteURLFunc
	origGetGitHubRepoInfo := getGitHubRepoInfoFunc
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		writeFileFunc = origWriteFile
		mkdirAllFunc = origMkdirAll
		createGitHubClientFunc = origGitHubClient
		getRemoteURLFunc = origGetRemoteURL
		getGitHubRepoInfoFunc = origGetGitHubRepoInfo
	}()

	tests := []struct {
		name               string
		args               []string
		setupMocks         func()
		wantErr            bool
		wantOutputContains []string
	}{
		{
			name:    "正常系: initコマンドヘルプ",
			args:    []string{"init", "--help"},
			wantErr: false,
			wantOutputContains: []string{
				"init",
				"osobaプロジェクトのための初期設定",
			},
		},
		{
			name: "正常系: initコマンド実行",
			args: []string{"init"},
			setupMocks: func() {
				isGitRepositoryFunc = func(path string) (bool, error) {
					return true, nil
				}
				checkCommandFunc = func(cmd string) error {
					return nil
				}
				getEnvFunc = func(key string) string {
					return ""
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					return nil
				}
				mkdirAllFunc = func(path string, perm os.FileMode) error {
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
				mockClient := &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						return nil
					},
				}
				createGitHubClientFunc = func(token string) githubInterface {
					return mockClient
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"初期化が完了しました",
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
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
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

func TestInitCmd_EnvironmentChecks(t *testing.T) {
	// モック関数を保存しておく
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origGetEnv := getEnvFunc
	origGetGitHubToken := getGitHubTokenFunc
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		getGitHubTokenFunc = origGetGitHubToken
	}()

	tests := []struct {
		name               string
		setupMocks         func()
		wantErr            bool
		wantOutputContains []string
		wantErrContains    string
	}{
		{
			name: "エラー: Gitリポジトリでない",
			setupMocks: func() {
				isGitRepositoryFunc = func(path string) (bool, error) {
					return false, nil
				}
			},
			wantErr:         true,
			wantErrContains: "Gitリポジトリのルートディレクトリで実行してください",
		},
		{
			name: "エラー: gitコマンドが存在しない",
			setupMocks: func() {
				isGitRepositoryFunc = func(path string) (bool, error) {
					return true, nil
				}
				checkCommandFunc = func(cmd string) error {
					if cmd == "git" {
						return fmt.Errorf("command not found: git")
					}
					return nil
				}
			},
			wantErr:         true,
			wantErrContains: "gitがインストールされていません",
		},
		{
			name: "エラー: tmuxコマンドが存在しない",
			setupMocks: func() {
				isGitRepositoryFunc = func(path string) (bool, error) {
					return true, nil
				}
				checkCommandFunc = func(cmd string) error {
					if cmd == "tmux" {
						return fmt.Errorf("command not found: tmux")
					}
					return nil
				}
			},
			wantErr:         true,
			wantErrContains: "tmuxがインストールされていません",
		},
		{
			name: "エラー: claudeコマンドが存在しない",
			setupMocks: func() {
				isGitRepositoryFunc = func(path string) (bool, error) {
					return true, nil
				}
				checkCommandFunc = func(cmd string) error {
					if cmd == "claude" {
						return fmt.Errorf("command not found: claude")
					}
					return nil
				}
			},
			wantErr:         true,
			wantErrContains: "claudeがインストールされていません",
		},
		{
			name: "警告: GitHub認証が設定されていない",
			setupMocks: func() {
				isGitRepositoryFunc = func(path string) (bool, error) {
					return true, nil
				}
				checkCommandFunc = func(cmd string) error {
					return nil
				}
				getEnvFunc = func(key string) string {
					return ""
				}
				getGitHubTokenFunc = func(cfg *config.Config) (string, string) {
					return "", ""
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[9/9] GitHubラベルの作成           ⚠️  (トークンなし)",
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

func TestInitCmd_SetupOperations(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	tempHome := filepath.Join(tempDir, "home")
	tempRepo := filepath.Join(tempDir, "repo")

	// ディレクトリを作成
	os.MkdirAll(tempHome, 0755)
	os.MkdirAll(filepath.Join(tempRepo, ".git"), 0755)

	// 元の環境変数を保存
	origHome := os.Getenv("HOME")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)
	}()

	// テスト用の環境変数を設定
	os.Setenv("HOME", tempHome)
	os.Unsetenv("XDG_CONFIG_HOME")

	// モック関数を保存しておく
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origGetEnv := getEnvFunc
	origWriteFile := writeFileFunc
	origMkdirAll := mkdirAllFunc
	origGitHubClient := createGitHubClientFunc
	origGetRemoteURL := getRemoteURLFunc
	origStat := statFunc
	origGetGitHubRepoInfo := getGitHubRepoInfoFunc
	origGetGitHubToken := getGitHubTokenFunc
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		writeFileFunc = origWriteFile
		mkdirAllFunc = origMkdirAll
		createGitHubClientFunc = origGitHubClient
		getRemoteURLFunc = origGetRemoteURL
		statFunc = origStat
		getGitHubRepoInfoFunc = origGetGitHubRepoInfo
		getGitHubTokenFunc = origGetGitHubToken
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
	getGitHubTokenFunc = func(cfg *config.Config) (string, string) {
		return "test-token", "test"
	}

	tests := []struct {
		name               string
		setupMocks         func()
		wantErr            bool
		wantOutputContains []string
		checkFiles         []string
	}{
		{
			name: "正常系: 設定ファイルとClaude commandsの作成",
			setupMocks: func() {
				fileCreated := make(map[string]bool)

				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}

				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					fileCreated[path] = true
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

				// GitHubクライアントのモック
				mockClient := &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						return nil
					},
				}
				createGitHubClientFunc = func(token string) githubInterface {
					return mockClient
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"🚀 osobaの初期化を開始します",
				"[1/8] Gitリポジトリの確認          ✅",
				"[2/9] 必要なツールの確認            ✅",
				"[6/9] 設定ファイルの作成           ✅",
				"[7/9] Claude commandsの配置        ✅",
				"[8/9] ドキュメントシステムの配置   ✅",
				"[9/9] GitHubラベルの作成           ✅",
				"✅ 初期化が完了しました！",
				"osoba start",
				"osoba open",
			},
		},
		{
			name: "正常系: 既存の設定ファイルがある場合はスキップ",
			setupMocks: func() {
				// 設定ファイルが既に存在することをシミュレート
				statFunc = func(name string) (os.FileInfo, error) {
					if strings.HasSuffix(name, "osoba.yml") {
						// ファイルが存在することを示す
						return nil, nil // FileInfoがnullでも、errがnilなら存在と判定
					}
					return nil, os.ErrNotExist
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

				mockClient := &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						return nil
					},
				}
				createGitHubClientFunc = func(token string) githubInterface {
					return mockClient
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[6/9] 設定ファイルの作成           ✅ (既存)",
			},
		},
		{
			name: "正常系: 作成される設定ファイルにClaude phases設定が含まれる",
			setupMocks: func() {
				// 設定ファイルが存在しないことをシミュレート
				statFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
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

				mockClient := &mockInitGitHubClient{
					ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
						return nil
					},
				}
				createGitHubClientFunc = func(token string) githubInterface {
					return mockClient
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[6/9] 設定ファイルの作成           ✅",
			},
		},
		{
			name: "エラー: GitHubラベル作成失敗（警告として処理）",
			setupMocks: func() {
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
				"[9/9] GitHubラベルの作成           ⚠️",
				"⚠️  GitHubラベルの作成に失敗しました",
				"手動でラベルを作成してください",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 作業ディレクトリを変更
			origWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			if err := os.Chdir(tempRepo); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(origWd); err != nil {
					t.Errorf("Failed to restore original directory: %v", err)
				}
			}()

			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			buf := new(bytes.Buffer)
			rootCmd = newRootCmd()
			rootCmd.AddCommand(newInitCmd())
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"init"})

			err = rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
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

// モック用のGitHubクライアント
type mockInitGitHubClient struct {
	ensureLabelsFunc func(ctx context.Context, owner, repo string) error
}

func (m *mockInitGitHubClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	if m.ensureLabelsFunc != nil {
		return m.ensureLabelsFunc(ctx, owner, repo)
	}
	return nil
}

func TestInitCmd_GitHubCLIChecks(t *testing.T) {
	// モック関数を保存しておく
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origGetEnv := getEnvFunc
	origExecCommand := execCommandFunc
	origMkdirAll := mkdirAllFunc
	origWriteFile := writeFileFunc
	origGetRemoteURL := getRemoteURLFunc
	origGitHubClient := createGitHubClientFunc
	origGetGitHubToken := getGitHubTokenFunc
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
		getGitHubTokenFunc = origGetGitHubToken
		getGitHubRepoInfoFunc = origGetGitHubRepoInfo
	}()

	// 基本的なモックを設定
	isGitRepositoryFunc = func(path string) (bool, error) {
		return true, nil
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
	mockClient := &mockInitGitHubClient{
		ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
			return nil
		},
	}
	createGitHubClientFunc = func(token string) githubInterface {
		return mockClient
	}
	getGitHubTokenFunc = func(cfg *config.Config) (string, string) {
		return "test-token", "test"
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
			name: "正常系: ghコマンドが利用可能で認証済み",
			setupMocks: func() {
				checkCommandFunc = func(cmd string) error {
					return nil
				}
				getEnvFunc = func(key string) string {
					return ""
				}
				execCommandFunc = func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 {
						switch args[0] {
						case "--version":
							return []byte("gh version 2.40.1"), nil
						case "auth":
							if len(args) > 1 && args[1] == "status" {
								return []byte("✓ Logged in to github.com as user (oauth_token)"), nil
							}
						case "repo":
							if len(args) > 1 && args[1] == "view" {
								return []byte("douhashi/osoba"), nil
							}
						}
					}
					return []byte{}, nil
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[3/9] GitHub CLI (gh)の確認        ✅",
				"[4/9] GitHub認証の確認             ✅",
				"[5/9] GitHubリポジトリへのアクセス確認  ✅",
			},
		},
		{
			name: "エラー: ghコマンドがインストールされていない",
			setupMocks: func() {
				checkCommandFunc = func(cmd string) error {
					if cmd == "gh" {
						return fmt.Errorf("command not found: gh")
					}
					return nil
				}
			},
			wantErr:         true,
			wantErrContains: "GitHub CLI (gh)がインストールされていません",
		},
		{
			name: "エラー: gh --versionが失敗",
			setupMocks: func() {
				checkCommandFunc = func(cmd string) error {
					return nil
				}
				execCommandFunc = func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 && args[0] == "--version" {
						return nil, fmt.Errorf("gh: command failed")
					}
					return []byte{}, nil
				}
			},
			wantErr:         true,
			wantErrContains: "GitHub CLI (gh)の動作確認に失敗しました",
		},
		{
			name: "警告: GitHub認証が未設定",
			setupMocks: func() {
				checkCommandFunc = func(cmd string) error {
					return nil
				}
				execCommandFunc = func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 {
						switch args[0] {
						case "--version":
							return []byte("gh version 2.40.1"), nil
						case "auth":
							if len(args) > 1 && args[1] == "status" {
								return nil, fmt.Errorf("not logged in")
							}
						case "repo":
							if len(args) > 1 && args[1] == "view" {
								return nil, fmt.Errorf("not found")
							}
						}
					}
					return []byte{}, nil
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[4/9] GitHub認証の確認             ⚠️",
				"⚠️  GitHub認証が設定されていません",
				"gh auth login",
			},
		},
		{
			name: "警告: リポジトリアクセス権限なし",
			setupMocks: func() {
				checkCommandFunc = func(cmd string) error {
					return nil
				}
				getEnvFunc = func(key string) string {
					return ""
				}
				execCommandFunc = func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 {
						switch args[0] {
						case "--version":
							return []byte("gh version 2.40.1"), nil
						case "auth":
							if len(args) > 1 && args[1] == "status" {
								return []byte("✓ Logged in to github.com as user"), nil
							}
						case "repo":
							if len(args) > 1 && args[1] == "view" {
								return nil, fmt.Errorf("not found")
							}
						}
					}
					return []byte{}, nil
				}
			},
			wantErr: false,
			wantOutputContains: []string{
				"[5/9] GitHubリポジトリへのアクセス確認  ⚠️",
				"⚠️  現在のリポジトリにアクセスできません",
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

func TestSetupClaudeCommands(t *testing.T) {
	// モック関数を保存しておく
	origMkdirAll := mkdirAllFunc
	origWriteFile := writeFileFunc
	origStat := statFunc
	defer func() {
		mkdirAllFunc = origMkdirAll
		writeFileFunc = origWriteFile
		statFunc = origStat
	}()

	tests := []struct {
		name         string
		setupMocks   func()
		wantErr      bool
		wantOutput   string
		filesCreated map[string]bool
		filesSkipped map[string]bool
	}{
		{
			name: "正常系: 全ファイルが新規作成される",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					// 全ファイルが存在しない
					return nil, os.ErrNotExist
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					return nil
				}
			},
			wantErr:    false,
			wantOutput: "✅",
			filesCreated: map[string]bool{
				".claude/commands/osoba/plan.md":        true,
				".claude/commands/osoba/implement.md":   true,
				".claude/commands/osoba/review.md":      true,
				".claude/commands/osoba/add-backlog.md": true,
			},
		},
		{
			name: "正常系: 全ファイルが既存でスキップされる",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					// 全ファイルが存在する
					return nil, nil
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					t.Errorf("writeFile should not be called for existing files: %s", path)
					return nil
				}
			},
			wantErr:    false,
			wantOutput: "✅ (既存)",
			filesSkipped: map[string]bool{
				".claude/commands/osoba/plan.md":        true,
				".claude/commands/osoba/implement.md":   true,
				".claude/commands/osoba/review.md":      true,
				".claude/commands/osoba/add-backlog.md": true,
			},
		},
		{
			name: "正常系: 一部ファイルが既存で残りは新規作成",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					// plan.mdのみ存在
					if strings.HasSuffix(name, "plan.md") {
						return nil, nil
					}
					return nil, os.ErrNotExist
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					if strings.HasSuffix(path, "plan.md") {
						t.Errorf("writeFile should not be called for existing file: %s", path)
					}
					return nil
				}
			},
			wantErr:    false,
			wantOutput: "✅ (一部既存)",
			filesCreated: map[string]bool{
				".claude/commands/osoba/implement.md":   true,
				".claude/commands/osoba/review.md":      true,
				".claude/commands/osoba/add-backlog.md": true,
			},
			filesSkipped: map[string]bool{
				".claude/commands/osoba/plan.md": true,
			},
		},
		{
			name: "エラー: ディレクトリ作成失敗",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return fmt.Errorf("permission denied")
				}
			},
			wantErr: true,
		},
		{
			name: "エラー: ファイル書き込み失敗",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					return fmt.Errorf("disk full")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			// ファイル作成・スキップの追跡
			filesCreated := make(map[string]bool)
			filesSkipped := make(map[string]bool)

			// モック関数を更新して追跡
			origWriteFile := writeFileFunc
			writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
				filesCreated[path] = true
				return origWriteFile(path, data, perm)
			}

			origStat := statFunc
			statFunc = func(name string) (os.FileInfo, error) {
				info, err := origStat(name)
				if err == nil && strings.Contains(name, ".claude/commands/osoba/") {
					filesSkipped[name] = true
				}
				return info, err
			}

			buf := new(bytes.Buffer)
			err := setupClaudeCommands(buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("setupClaudeCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				output := buf.String()
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("setupClaudeCommands() output = %v, want to contain %v", output, tt.wantOutput)
				}

				// ファイル作成の確認
				if tt.filesCreated != nil {
					for file, expected := range tt.filesCreated {
						if filesCreated[file] != expected {
							t.Errorf("file %s: created = %v, want %v", file, filesCreated[file], expected)
						}
					}
				}

				// ファイルスキップの確認
				if tt.filesSkipped != nil {
					for file, expected := range tt.filesSkipped {
						if filesSkipped[file] != expected {
							t.Errorf("file %s: skipped = %v, want %v", file, filesSkipped[file], expected)
						}
					}
				}
			}
		})
	}
}

func TestSetupDocumentSystem(t *testing.T) {
	// モック関数を保存しておく
	origMkdirAll := mkdirAllFunc
	origWriteFile := writeFileFunc
	origStat := statFunc
	defer func() {
		mkdirAllFunc = origMkdirAll
		writeFileFunc = origWriteFile
		statFunc = origStat
	}()

	tests := []struct {
		name       string
		setupMocks func()
		wantErr    bool
		wantOutput string
	}{
		{
			name: "正常系: 新規ファイル作成",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					// ファイルが存在しない
					return nil, os.ErrNotExist
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					return nil
				}
			},
			wantErr:    false,
			wantOutput: "✅",
		},
		{
			name: "正常系: 既存ファイルをスキップ",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					// ファイルが存在する
					return nil, nil
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					t.Errorf("writeFile should not be called for existing file: %s", path)
					return nil
				}
			},
			wantErr:    false,
			wantOutput: "✅ (既存)",
		},
		{
			name: "エラー: ディレクトリ作成失敗",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return fmt.Errorf("permission denied")
				}
			},
			wantErr: true,
		},
		{
			name: "エラー: ファイル書き込み失敗",
			setupMocks: func() {
				mkdirAllFunc = func(path string, perm os.FileMode) error {
					return nil
				}
				statFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
				writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
					return fmt.Errorf("disk full")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			buf := &bytes.Buffer{}
			err := setupDocumentSystem(buf)

			if (err != nil) != tt.wantErr {
				t.Errorf("setupDocumentSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := strings.TrimSpace(buf.String())
				if output != tt.wantOutput {
					t.Errorf("output = %q, want %q", output, tt.wantOutput)
				}
			}
		})
	}
}
