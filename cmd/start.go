package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/gh"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/watcher"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var (
		intervalFlag string
		configFlag   string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Issue監視を開始",
		Long: `現在のGitリポジトリでGitHub Issueの監視を開始します。
tmuxセッションが存在しない場合は自動的に作成されます。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Issue監視を開始
			return runWatchWithFlagsFunc(cmd, args, intervalFlag, configFlag)
		},
	}

	cmd.Flags().StringVarP(&intervalFlag, "interval", "i", "5s", "ポーリング間隔")
	cmd.Flags().StringVarP(&configFlag, "config", "c", "", "設定ファイルのパス")

	return cmd
}

// テスト用にモック可能な関数変数
var runWatchWithFlagsFunc = runWatchWithFlags

func runWatchWithFlags(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "Issue監視モードを開始します")

	// 設定を読み込む
	cfg := config.NewConfig()
	// LoadOrDefaultを使用してデフォルト設定ファイルも読み込む
	actualConfigPath := cfg.LoadOrDefault(configFlag)

	// 設定ファイルの使用状況をログに出力
	if actualConfigPath != "" {
		if configFlag != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "設定ファイル: %s (指定されたファイル)\n", actualConfigPath)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "設定ファイル: %s (デフォルト)\n", actualConfigPath)
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "設定ファイル: なし (デフォルト値を使用)")
	}

	// ポーリング間隔を設定
	if intervalFlag != "" && intervalFlag != "5s" {
		interval, err := time.ParseDuration(intervalFlag)
		if err != nil {
			return fmt.Errorf("不正なポーリング間隔: %w", err)
		}
		cfg.GitHub.PollInterval = interval
	}

	// 設定値の詳細をログ出力
	fmt.Fprintln(cmd.OutOrStdout(), "\n設定値:")
	fmt.Fprintf(cmd.OutOrStdout(), "  ポーリング間隔: %s\n", cfg.GitHub.PollInterval)

	// トークンの取得元を表示
	token, source := config.GetGitHubToken(cfg)
	if token != "" {
		// トークンの最初の3文字と長さを表示（セキュリティのため全体は表示しない）
		maskedToken := fmt.Sprintf("%s... (長さ: %d文字)", token[:3], len(token))
		fmt.Fprintf(cmd.OutOrStdout(), "  GitHubトークン: %s (取得元: %s)\n", maskedToken, source)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  GitHubトークン: 未設定")
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

	// GitHubクライアントを作成（設定に応じてghコマンドまたはAPIを使用）
	var githubClient github.GitHubClient
	if cfg.GitHub.UseGhCommand {
		// ghコマンドベースのクライアント
		executor := gh.NewRealCommandExecutor()
		ghClient, err := gh.NewClient(executor)
		if err != nil {
			return fmt.Errorf("ghクライアントの作成に失敗: %w", err)
		}
		// 前提条件を検証
		if err := ghClient.ValidatePrerequisites(context.Background()); err != nil {
			return fmt.Errorf("ghコマンドの前提条件を満たしていません: %w", err)
		}
		githubClient = ghClient
		fmt.Fprintln(cmd.OutOrStdout(), "  GitHub接続: ghコマンドを使用")
	} else {
		// GitHub APIベースのクライアント
		apiClient, err := github.NewClient(cfg.GitHub.Token)
		if err != nil {
			return fmt.Errorf("GitHubクライアントの作成に失敗: %w", err)
		}
		githubClient = apiClient
		fmt.Fprintln(cmd.OutOrStdout(), "  GitHub接続: GitHub APIを使用")
	}

	// tmuxがインストールされているか確認
	if err := tmux.CheckTmuxInstalled(); err != nil {
		return fmt.Errorf("%w", err)
	}

	// セッション名を生成
	sessionName := fmt.Sprintf("%s%s", cfg.Tmux.SessionPrefix, repoName)

	// tmuxセッションを確保（存在しない場合は作成）
	fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッション '%s' を確認中...\n", sessionName)
	if err := tmux.EnsureSession(sessionName); err != nil {
		return fmt.Errorf("tmuxセッションの確保に失敗: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッション '%s' が利用可能です\n", sessionName)

	// 必要なラベルが存在することを確認
	fmt.Fprintln(cmd.OutOrStdout(), "必要なラベルを確認中...")
	if err := githubClient.EnsureLabelsExist(context.Background(), owner, repoName); err != nil {
		// エラーでも処理は続行（ラベル作成権限がない場合もあるため）
		fmt.Fprintf(cmd.OutOrStderr(), "警告: ラベルの確認/作成に失敗しました: %v\n", err)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "ラベルの確認が完了しました")
	}

	// ロガーを作成
	appLogger, err := logger.New(logger.WithLevel("info"))
	if err != nil {
		return fmt.Errorf("ロガーの作成に失敗: %w", err)
	}

	// Git関連のコンポーネントを作成
	gitRepository := git.NewRepository(appLogger)
	gitWorktree := git.NewWorktree(appLogger)
	gitBranch := git.NewBranch(appLogger)
	gitSync := git.NewSync(appLogger)

	// WorktreeManagerを作成
	worktreeManager, err := git.NewWorktreeManager(gitRepository, gitWorktree, gitBranch, gitSync)
	if err != nil {
		return fmt.Errorf("WorktreeManagerの作成に失敗: %w", err)
	}

	// Claude関連の設定とExecutorを作成
	claudeConfig := cfg.Claude
	if claudeConfig == nil {
		claudeConfig = claude.NewDefaultClaudeConfig()
	}
	claudeExecutor := claude.NewClaudeExecutor()

	// ActionFactoryを作成
	actionFactory := watcher.NewDefaultActionFactory(
		sessionName,
		githubClient,
		worktreeManager,
		claudeExecutor,
		claudeConfig,
		cfg,
		owner,
		repoName,
	)

	// Issue監視を作成
	issueWatcher, err := watcher.NewIssueWatcher(githubClient, owner, repoName, sessionName, cfg.GetLabels(), cfg.GitHub.PollInterval, appLogger)
	if err != nil {
		return fmt.Errorf("Issue監視の作成に失敗: %w", err)
	}

	// ActionManagerにActionFactoryを設定
	issueWatcher.GetActionManager().SetActionFactory(actionFactory)

	// シグナルハンドリング
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		appLogger.Info("シグナルを受信しました。終了します...")
		cancel()
	}()

	// Issue監視を開始（StartWithActionsを使用）
	issueWatcher.StartWithActions(ctx)
	return nil
}
