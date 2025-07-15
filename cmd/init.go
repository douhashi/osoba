package cmd

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/douhashi/osoba/internal/github"
	"github.com/spf13/cobra"
)

//go:embed templates/*
var templateFS embed.FS

// githubInterface はテスト用のGitHubクライアントインターフェース
type githubInterface interface {
	EnsureLabelsExist(ctx context.Context, owner, repo string) error
}

// モック用の関数変数
var (
	isGitRepositoryFunc    = isGitRepository
	checkCommandFunc       = checkCommand
	getEnvFunc             = os.Getenv
	writeFileFunc          = os.WriteFile
	mkdirAllFunc           = os.MkdirAll
	statFunc               = os.Stat
	createGitHubClientFunc = func(token string) githubInterface {
		client, _ := github.NewClient(token)
		return client
	}
)

// isGitRepository は指定されたパスがgitリポジトリかを確認する
func isGitRepository(path string) (bool, error) {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// GetRemoteURL はgitリモートURLを取得する
var getRemoteURLFunc = func(remoteName string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL はgitリモートURLを取得する
func GetRemoteURL(remoteName string) (string, error) {
	return getRemoteURLFunc(remoteName)
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "プロジェクトを初期化",
		Long:  `osobaプロジェクトのための初期設定を行います。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()

			// 1. Gitリポジトリの確認
			if err := checkGitRepository(out); err != nil {
				return err
			}

			// 2. 必要ツールの確認
			if err := checkRequiredTools(out); err != nil {
				return err
			}

			// 3. GitHub Tokenの確認（警告のみ）
			checkGitHubToken(errOut)

			// 4. 設定ファイルの作成
			if err := setupConfigFile(out); err != nil {
				return fmt.Errorf("設定ファイルの作成に失敗しました: %w", err)
			}

			// 5. Claude commandsの配置
			if err := setupClaudeCommands(out); err != nil {
				return fmt.Errorf("Claude commandsの配置に失敗しました: %w", err)
			}

			// 6. GitHubラベルの作成（エラーは警告）
			setupGitHubLabels(out, errOut)

			// 7. 次の操作案内
			showNextSteps(out)

			return nil
		},
	}
	return cmd
}

func checkCommand(command string) error {
	_, err := exec.LookPath(command)
	return err
}

func checkGitRepository(out io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("現在のディレクトリを取得できません: %w", err)
	}

	isRepo, err := isGitRepositoryFunc(cwd)
	if err != nil {
		return fmt.Errorf("Gitリポジトリの確認に失敗しました: %w", err)
	}

	if !isRepo {
		return fmt.Errorf("Gitリポジトリのルートディレクトリで実行してください")
	}

	fmt.Fprintln(out, "✓ Gitリポジトリを確認しました")
	return nil
}

func checkRequiredTools(out io.Writer) error {
	tools := []struct {
		name string
		cmd  string
	}{
		{name: "git", cmd: "git"},
		{name: "tmux", cmd: "tmux"},
		{name: "claude", cmd: "claude"},
	}

	for _, tool := range tools {
		if err := checkCommandFunc(tool.cmd); err != nil {
			return fmt.Errorf("%sがインストールされていません", tool.name)
		}
	}

	fmt.Fprintln(out, "✓ 必要なツールを確認しました")
	return nil
}

func checkGitHubToken(out io.Writer) {
	token := getEnvFunc("GITHUB_TOKEN")
	if token == "" {
		token = getEnvFunc("OSOBA_GITHUB_TOKEN")
	}

	if token == "" {
		fmt.Fprintln(out, "⚠️  GitHub Personal Access Tokenが設定されていません")
		fmt.Fprintln(out, "   以下のコマンドで設定してください:")
		fmt.Fprintln(out, "   export GITHUB_TOKEN=your_token_here")
	}
}

func setupConfigFile(out io.Writer) error {
	home := getEnvFunc("HOME")
	xdgConfigHome := getEnvFunc("XDG_CONFIG_HOME")
	configDir := filepath.Join(home, ".config", "osoba")
	if xdgConfigHome != "" {
		configDir = filepath.Join(xdgConfigHome, "osoba")
	}

	configPath := filepath.Join(configDir, "osoba.yml")

	// 既存ファイルの確認
	if _, err := statFunc(configPath); err == nil {
		fmt.Fprintln(out, "✓ 設定ファイルは既に存在します")
		return nil
	}

	// ディレクトリの作成
	if err := mkdirAllFunc(configDir, 0755); err != nil {
		return fmt.Errorf("設定ディレクトリの作成に失敗しました: %w", err)
	}

	// デフォルト設定の作成
	defaultConfig := `# 最小限の設定ファイルサンプル

github:
  token: "${GITHUB_TOKEN}"
  poll_interval: 10s

tmux:
  session_prefix: "osoba-"

claude:
  phases:
    plan:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:plan {{issue-number}}"
    implement:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:implement {{issue-number}}"
    review:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:review {{issue-number}}"
`

	if err := writeFileFunc(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("設定ファイルの作成に失敗しました: %w", err)
	}

	fmt.Fprintf(out, "✓ 設定ファイルを作成しました: %s\n", configPath)
	return nil
}

func setupClaudeCommands(out io.Writer) error {
	// .claude/commands/osoba ディレクトリの作成
	dir := filepath.Join(".claude", "commands", "osoba")
	if err := mkdirAllFunc(dir, 0755); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗しました: %w", err)
	}

	// テンプレートファイルの配置
	files := []string{"plan.md", "implement.md", "review.md"}
	for _, file := range files {
		src := "templates/commands/" + file
		dst := filepath.Join(dir, file)

		data, err := templateFS.ReadFile(src)
		if err != nil {
			return fmt.Errorf("テンプレートファイルの読み込みに失敗しました: %w", err)
		}

		if err := writeFileFunc(dst, data, 0644); err != nil {
			return fmt.Errorf("ファイルの作成に失敗しました: %w", err)
		}
	}

	fmt.Fprintln(out, "✓ Claude commandsを配置しました")
	return nil
}

func setupGitHubLabels(out, errOut io.Writer) {
	token := getEnvFunc("GITHUB_TOKEN")
	if token == "" {
		token = getEnvFunc("OSOBA_GITHUB_TOKEN")
	}

	if token == "" {
		return // Tokenがない場合はスキップ
	}

	// リポジトリ情報の取得
	origin, err := GetRemoteURL("origin")
	if err != nil {
		fmt.Fprintf(errOut, "⚠️  GitリモートURLの取得に失敗しました: %v\n", err)
		return
	}

	owner, repo := parseGitHubURL(origin)
	if owner == "" || repo == "" {
		fmt.Fprintf(errOut, "⚠️  GitHubリポジトリ情報の解析に失敗しました\n")
		return
	}

	// GitHubクライアントの作成
	client := createGitHubClientFunc(token)

	// ラベルの作成のためにgithub.Clientを使用
	ctx := context.Background()
	if err := client.EnsureLabelsExist(ctx, owner, repo); err != nil {
		fmt.Fprintf(errOut, "⚠️  GitHubラベルの作成に失敗しました: %v\n", err)
		fmt.Fprintln(errOut, "   手動でラベルを作成してください")
		return
	}

	fmt.Fprintln(out, "✓ GitHubラベルを作成しました")
}

func parseGitHubURL(url string) (owner, repo string) {
	// 簡易的なGitHub URLパーサー
	// https://github.com/owner/repo.git -> owner, repo
	// git@github.com:owner/repo.git -> owner, repo

	url = strings.TrimSuffix(url, ".git")

	if strings.HasPrefix(url, "https://github.com/") {
		parts := strings.Split(strings.TrimPrefix(url, "https://github.com/"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	} else if strings.HasPrefix(url, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(url, "git@github.com:"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	}

	return "", ""
}

func showNextSteps(out io.Writer) {
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "🎉 初期化が完了しました！")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "次のステップ:")
	fmt.Fprintln(out, "1. osoba start - Watcherを起動してIssueの監視を開始")
	fmt.Fprintln(out, "2. osoba open  - 別ターミナルでtmuxセッションを開く")
	fmt.Fprintln(out, "3. GitHubでIssueを作成し、'status:needs-plan'ラベルを付ける")
}
