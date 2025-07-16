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

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/utils"
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
	execCommandFunc        = execCommand
	createGitHubClientFunc = func(token string) githubInterface {
		client, _ := github.NewClient(token)
		return client
	}
	getGitHubRepoInfoFunc = utils.GetGitHubRepoInfo
	getGitHubTokenFunc    = config.GetGitHubToken
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

			// 初期化開始メッセージ
			fmt.Fprintln(out, "🚀 osobaの初期化を開始します...")
			fmt.Fprintln(out, "")

			// 1. Gitリポジトリの確認
			fmt.Fprint(out, "[1/8] Gitリポジトリの確認          ")
			if err := checkGitRepository(out); err != nil {
				fmt.Fprintln(out, "❌")
				return err
			}

			// 2. 必要ツールの確認
			fmt.Fprint(out, "[2/8] 必要なツールの確認            ")
			if err := checkRequiredTools(out); err != nil {
				fmt.Fprintln(out, "❌")
				return err
			}

			// 3. GitHub CLI (gh)の確認
			fmt.Fprint(out, "[3/8] GitHub CLI (gh)の確認        ")
			if err := checkGitHubCLI(out, errOut); err != nil {
				fmt.Fprintln(out, "❌")
				return err
			}

			// 4. GitHub認証の確認
			fmt.Fprint(out, "[4/8] GitHub認証の確認             ")
			checkGitHubAuth(out, errOut)

			// 5. GitHubリポジトリへのアクセス確認
			fmt.Fprint(out, "[5/8] GitHubリポジトリへのアクセス確認  ")
			checkRepositoryAccess(out, errOut)

			// 6. 設定ファイルの作成
			fmt.Fprint(out, "[6/8] 設定ファイルの作成           ")
			if err := setupConfigFile(out); err != nil {
				fmt.Fprintln(out, "❌")
				return fmt.Errorf("設定ファイルの作成に失敗しました: %w", err)
			}

			// 7. Claude commandsの配置
			fmt.Fprint(out, "[7/8] Claude commandsの配置        ")
			if err := setupClaudeCommands(out); err != nil {
				fmt.Fprintln(out, "❌")
				return fmt.Errorf("Claude commandsの配置に失敗しました: %w", err)
			}

			// 8. GitHubラベルの作成（エラーは警告）
			fmt.Fprint(out, "[8/8] GitHubラベルの作成           ")
			setupGitHubLabels(out, errOut)

			fmt.Fprintln(out, "")

			// 完了メッセージ
			showCompletionMessage(out)

			return nil
		},
	}
	return cmd
}

func checkCommand(command string) error {
	_, err := exec.LookPath(command)
	return err
}

// execCommand はコマンドを実行して出力を返す
func execCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
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

	fmt.Fprintln(out, "✅")
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

	fmt.Fprintln(out, "✅")
	return nil
}

// checkGitHubCLI はGitHub CLIの状態を確認する
func checkGitHubCLI(out, errOut io.Writer) error {
	// 1. ghコマンドがインストールされているかチェック
	if err := checkCommandFunc("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh)がインストールされていません。以下のURLからインストールしてください: https://cli.github.com/")
	}

	// 2. gh --versionで動作確認
	if _, err := execCommandFunc("gh", "--version"); err != nil {
		return fmt.Errorf("GitHub CLI (gh)の動作確認に失敗しました: %w", err)
	}
	fmt.Fprintln(out, "✅")

	return nil
}

// checkGitHubAuth はGitHub認証状態をチェックする
func checkGitHubAuth(out, errOut io.Writer) {
	if _, err := execCommandFunc("gh", "auth", "status"); err != nil {
		fmt.Fprintln(out, "⚠️")
		fmt.Fprintln(errOut, "⚠️  GitHub認証が設定されていません")
		fmt.Fprintln(errOut, "   以下のコマンドで認証してください:")
		fmt.Fprintln(errOut, "   gh auth login")
		return
	}
	fmt.Fprintln(out, "✅")
}

// checkRepositoryAccess は現在のリポジトリへのアクセスをチェックする
func checkRepositoryAccess(out, errOut io.Writer) {
	if _, err := execCommandFunc("gh", "repo", "view"); err != nil {
		fmt.Fprintln(out, "⚠️")
		fmt.Fprintln(errOut, "⚠️  現在のリポジトリにアクセスできません")
		fmt.Fprintln(errOut, "   リポジトリのアクセス権限を確認してください")
		return
	}
	fmt.Fprintln(out, "✅")
}

func checkGitHubToken(out io.Writer) {
	// config.GetGitHubTokenを使用してトークンと取得元を取得
	cfg := config.NewConfig()
	token, source := config.GetGitHubToken(cfg)

	if token == "" {
		fmt.Fprintln(out, "⚠️  GitHub Personal Access Tokenが設定されていません")
		fmt.Fprintln(out, "   以下のいずれかの方法で設定してください:")
		fmt.Fprintln(out, "   1. export GITHUB_TOKEN=your_token_here")
		fmt.Fprintln(out, "   2. gh auth login (GitHub CLIでログイン)")
	} else {
		fmt.Fprintf(out, "✅ GitHub Token設定済み (取得元: %s)\n", source)
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
		fmt.Fprintln(out, "✅ (既存)")
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

	fmt.Fprintln(out, "✅")
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

	fmt.Fprintln(out, "✅")
	return nil
}

func setupGitHubLabels(out, errOut io.Writer) {
	// config.GetGitHubTokenを使用してトークンを取得
	cfg := config.NewConfig()
	token, _ := getGitHubTokenFunc(cfg)

	if token == "" {
		fmt.Fprintln(out, "⚠️  (トークンなし)")
		return // Tokenがない場合はスキップ
	}

	// リポジトリ情報の取得（共通関数を使用）
	ctx := context.Background()
	repoInfo, err := getGitHubRepoInfoFunc(ctx)
	if err != nil {
		fmt.Fprintln(out, "⚠️")
		// 詳細なエラーメッセージを表示
		if repoErr, ok := err.(*utils.GetGitHubRepoInfoError); ok {
			switch repoErr.Step {
			case "working_directory":
				fmt.Fprintf(errOut, "⚠️  作業ディレクトリの取得に失敗しました: %v\n", repoErr.Cause)
			case "git_directory":
				fmt.Fprintf(errOut, "⚠️  Gitリポジトリが見つかりません。Gitリポジトリのルートディレクトリで実行してください\n")
			case "remote_url":
				fmt.Fprintf(errOut, "⚠️  リモートURL取得に失敗しました: %v\n", repoErr.Cause)
				fmt.Fprintf(errOut, "   'git remote add origin <URL>' でリモートを設定してください\n")
			case "url_parsing":
				fmt.Fprintf(errOut, "⚠️  GitHubリポジトリ情報の解析に失敗しました: %v\n", repoErr.Cause)
				fmt.Fprintf(errOut, "   GitHubのリポジトリURLが正しく設定されているか確認してください\n")
			default:
				fmt.Fprintf(errOut, "⚠️  GitHubリポジトリ情報の取得に失敗しました: %v\n", err)
			}
		} else {
			fmt.Fprintf(errOut, "⚠️  GitHubリポジトリ情報の取得に失敗しました: %v\n", err)
		}
		return
	}

	// GitHubクライアントの作成
	client := createGitHubClientFunc(token)

	// ラベルの作成のためにgithub.Clientを使用
	if err := client.EnsureLabelsExist(ctx, repoInfo.Owner, repoInfo.Repo); err != nil {
		fmt.Fprintln(out, "⚠️")
		fmt.Fprintf(errOut, "⚠️  GitHubラベルの作成に失敗しました: %v\n", err)
		fmt.Fprintln(errOut, "   手動でラベルを作成してください")
		return
	}

	fmt.Fprintln(out, "✅")
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

// showCompletionMessage は初期化完了メッセージを表示する
func showCompletionMessage(out io.Writer) {
	fmt.Fprintln(out, "✅ 初期化が完了しました！")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "次のステップ:")
	fmt.Fprintln(out, "1. osoba start - Watcherを起動してIssueの監視を開始")
	fmt.Fprintln(out, "2. osoba open  - 別ターミナルでtmuxセッションを開く")
	fmt.Fprintln(out, "3. GitHubでIssueを作成し、'status:needs-plan'ラベルを付ける")
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
