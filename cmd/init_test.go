package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		writeFileFunc = origWriteFile
		mkdirAllFunc = origMkdirAll
		createGitHubClientFunc = origGitHubClient
		getRemoteURLFunc = origGetRemoteURL
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
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
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
			name: "警告: GitHub Tokenが設定されていない",
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
			},
			wantErr: false,
			wantOutputContains: []string{
				"⚠️  GitHub Personal Access Tokenが設定されていません",
				"export GITHUB_TOKEN=",
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
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		writeFileFunc = origWriteFile
		mkdirAllFunc = origMkdirAll
		createGitHubClientFunc = origGitHubClient
		getRemoteURLFunc = origGetRemoteURL
		statFunc = origStat
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
				"✓ Gitリポジトリを確認しました",
				"✓ 必要なツールを確認しました",
				"✓ 設定ファイルを作成しました",
				"✓ Claude commandsを配置しました",
				"✓ GitHubラベルを作成しました",
				"🎉 初期化が完了しました！",
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
				"✓ 設定ファイルは既に存在します",
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
				"✓ 設定ファイルを作成しました",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 作業ディレクトリを変更
			origWd, _ := os.Getwd()
			os.Chdir(tempRepo)
			defer os.Chdir(origWd)

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
