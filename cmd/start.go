package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/watcher"
	gh "github.com/google/go-github/v67/github"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var (
		watchFlag    bool
		intervalFlag string
		configFlag   string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "tmuxセッションを作成またはIssue監視を開始",
		Long: `現在のGitリポジトリ専用のtmuxセッションを作成します。
--watchフラグを指定すると、GitHub Issueの監視モードで起動します。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// フラグを構造体に渡して実行
			return runStartWithFlags(cmd, args, watchFlag, intervalFlag, configFlag)
		},
	}

	cmd.Flags().BoolVarP(&watchFlag, "watch", "w", false, "Issue監視モードで起動")
	cmd.Flags().StringVarP(&intervalFlag, "interval", "i", "5s", "ポーリング間隔")
	cmd.Flags().StringVarP(&configFlag, "config", "c", "", "設定ファイルのパス")

	return cmd
}

// テスト用にモック可能な関数変数
var runWatchWithFlagsFunc = runWatchWithFlags

func runStartWithFlags(cmd *cobra.Command, args []string, watchFlag bool, intervalFlag, configFlag string) error {
	// --watchフラグが指定されている場合はIssue監視モードを開始
	if watchFlag {
		return runWatchWithFlagsFunc(cmd, args, intervalFlag, configFlag)
	}

	// 通常のtmuxセッション作成モード
	return runTmuxSession(cmd, args)
}

func runTmuxSession(cmd *cobra.Command, args []string) error {
	// 1. Gitリポジトリ名を取得
	repoName, err := git.GetRepositoryName()
	if err != nil {
		if errors.Is(err, git.ErrNotGitRepository) {
			return fmt.Errorf("%w", err)
		}
		if errors.Is(err, git.ErrNoRemoteFound) {
			return fmt.Errorf("%w", err)
		}
		return fmt.Errorf("リポジトリ名の取得に失敗: %w", err)
	}

	// 2. tmuxがインストールされているか確認
	if err := checkTmuxInstalled(); err != nil {
		return fmt.Errorf("%w", err)
	}

	// 3. セッション名を生成
	sessionName := fmt.Sprintf("osoba-%s", repoName)

	// 4. 既存セッションの確認
	exists, err := sessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("セッションの確認に失敗: %w", err)
	}

	if exists {
		// 既存セッションがある場合
		fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッション '%s' は既に存在します。\n", sessionName)
		fmt.Fprintf(cmd.OutOrStdout(), "接続するには以下のコマンドを実行してください:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  tmux attach -t %s\n", sessionName)
		return nil
	}

	// 5. 新規セッションを作成
	if err := createSession(sessionName); err != nil {
		return fmt.Errorf("セッションの作成に失敗: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッション '%s' を作成しました。\n", sessionName)
	fmt.Fprintf(cmd.OutOrStdout(), "接続するには以下のコマンドを実行してください:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  tmux attach -t %s\n", sessionName)

	return nil
}

// runWatch はIssue監視モードを実行する（テスト用にモック可能）
var runWatch = func(cmd *cobra.Command, args []string) error {
	// この関数は後方互換性のために残す
	return runWatchWithFlags(cmd, args, "5s", "")
}

func runWatchWithFlags(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "Issue監視モードを開始します")

	// 設定を読み込む
	cfg := config.NewConfig()
	if configFlag != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "設定ファイル: %s\n", configFlag)
		if err := cfg.Load(configFlag); err != nil {
			return fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
		}
	} else {
		// 環境変数から読み込み
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			cfg.GitHub.Token = token
		} else if token := os.Getenv("OSOBA_GITHUB_TOKEN"); token != "" {
			cfg.GitHub.Token = token
		}
	}

	// ポーリング間隔を設定
	if intervalFlag != "" && intervalFlag != "5s" {
		interval, err := time.ParseDuration(intervalFlag)
		if err != nil {
			return fmt.Errorf("不正なポーリング間隔: %w", err)
		}
		cfg.GitHub.PollInterval = interval
		fmt.Fprintf(cmd.OutOrStdout(), "ポーリング間隔: %s\n", interval)
	}

	// 設定の検証
	if err := cfg.Validate(); err != nil {
		return err
	}

	// リポジトリ情報を取得
	repoName, err := git.GetRepositoryName()
	if err != nil {
		return fmt.Errorf("リポジトリ名の取得に失敗: %w", err)
	}

	// リポジトリのオーナーを取得（簡易的にリポジトリ名から推測）
	// TODO: より正確な方法で取得する
	owner := "douhashi"

	// GitHub APIクライアントを作成
	githubClient, err := github.NewClient(cfg.GitHub.Token)
	if err != nil {
		return fmt.Errorf("GitHubクライアントの作成に失敗: %w", err)
	}

	// セッション名を生成
	sessionName := fmt.Sprintf("osoba-%s", repoName)

	// 必要なラベルが存在することを確認
	fmt.Fprintln(cmd.OutOrStdout(), "必要なラベルを確認中...")
	if err := githubClient.EnsureLabelsExist(context.Background(), owner, repoName); err != nil {
		// エラーでも処理は続行（ラベル作成権限がない場合もあるため）
		fmt.Fprintf(cmd.OutOrStderr(), "警告: ラベルの確認/作成に失敗しました: %v\n", err)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "ラベルの確認が完了しました")
	}

	// Issue監視を作成
	issueWatcher, err := watcher.NewIssueWatcher(githubClient, owner, repoName, sessionName, cfg.GetLabels())
	if err != nil {
		return fmt.Errorf("Issue監視の作成に失敗: %w", err)
	}
	issueWatcher.SetPollInterval(cfg.GitHub.PollInterval)

	// シグナルハンドリング
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("シグナルを受信しました。終了します...")
		cancel()
	}()

	// Issue監視を開始
	issueWatcher.Start(ctx, func(issue *gh.Issue) {
		log.Printf("Issue detected: #%d - %s", *issue.Number, *issue.Title)
		// TODO: ここでtmuxウィンドウ作成やClaude実行などの処理を追加
	})

	return nil
}

// テスト時にモック可能な関数変数
var (
	checkTmuxInstalled = tmux.CheckTmuxInstalled
	sessionExists      = tmux.SessionExists
	createSession      = tmux.CreateSession
)
