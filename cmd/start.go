package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/gh"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/utils"
	"github.com/douhashi/osoba/internal/watcher"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var (
		intervalFlag   string
		configFlag     string
		foregroundFlag bool
		logFileFlag    string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Issue監視を開始",
		Long: `現在のGitリポジトリでGitHub Issueの監視を開始します。
tmuxセッションが存在しない場合は自動的に作成されます。
デフォルトではバックグラウンドで実行されます。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// ヘルプフラグが指定されている場合は何もしない（ヘルプ表示のみ）
			help, _ := cmd.Flags().GetBool("help")
			if help {
				return nil
			}

			// フォアグラウンドフラグが指定されている場合は従来の動作
			if foregroundFlag {
				return runWatchWithFlagsFunc(cmd, args, intervalFlag, configFlag)
			}

			// デーモンモードで起動されている場合
			if isDaemonModeFunc() {
				// PIDファイルのパスを取得
				repoIdentifier, err := getRepoIdentifierFunc()
				if err != nil {
					return err
				}
				pm := paths.NewPathManager("")
				pidFile := pm.PIDFile(repoIdentifier)

				// デーモンモードで実行
				return runInDaemonModeFunc(cmd, pidFile, intervalFlag, configFlag)
			}

			// バックグラウンドで起動
			return startInBackgroundFunc(cmd, args)
		},
	}

	cmd.Flags().StringVarP(&intervalFlag, "interval", "i", "5s", "ポーリング間隔")
	cmd.Flags().StringVarP(&configFlag, "config", "c", "", "設定ファイルのパス")
	cmd.Flags().BoolVar(&foregroundFlag, "foreground", false, "フォアグラウンドで実行（デフォルト: false）")
	cmd.Flags().StringVar(&logFileFlag, "log-file", "", "ログファイルパス（デフォルト: 自動生成）")

	return cmd
}

// テスト用にモック可能な関数変数
var (
	runWatchWithFlagsFunc    = runWatchWithFlags
	isDaemonModeFunc         = isDaemonMode
	getRepoIdentifierFunc    = getRepoIdentifier
	startInBackgroundFunc    = startInBackground
	runInDaemonModeFunc      = runInDaemonMode
	checkExistingProcessFunc = checkExistingProcess
	createPIDFileFunc        = createPIDFile
	osUserHomeDirFunc        = os.UserHomeDir
)

// checkConfigFileExists は設定ファイルの存在をチェックし、存在しない場合はエラーメッセージを出力します
func checkConfigFileExists(errOut io.Writer) error {
	_, found := findConfigFile()
	if !found {
		fmt.Fprintln(errOut, "エラー: 設定ファイルが見つかりません")
		fmt.Fprintln(errOut, "")
		fmt.Fprintln(errOut, "以下のいずれかの場所に設定ファイルを配置してください:")

		paths := getConfigFilePaths()
		for _, path := range paths {
			fmt.Fprintf(errOut, "  - %s\n", path)
		}

		fmt.Fprintln(errOut, "")
		fmt.Fprintln(errOut, "または、以下のコマンドで初期設定を行ってください:")
		fmt.Fprintln(errOut, "  osoba init")

		return fmt.Errorf("設定ファイルが見つかりません")
	}
	return nil
}

func runWatchWithFlags(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
	// 設定ファイルの存在チェック
	if err := checkConfigFileExists(cmd.OutOrStderr()); err != nil {
		return err
	}

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

	// gh認証状態を表示
	token, source := config.GetGitHubToken(cfg)
	if token != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  GitHub認証: 有効 (取得元: %s)\n", source)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  GitHub認証: 未設定")
	}

	// 設定の検証
	if err := cfg.Validate(); err != nil {
		return err
	}

	// リポジトリ情報を取得
	repoInfo, err := utils.GetGitHubRepoInfo(context.Background())
	if err != nil {
		return fmt.Errorf("GitHubリポジトリ情報の取得に失敗: %w", err)
	}

	// リポジトリ名とオーナーを取得
	repoName := repoInfo.Repo
	owner := repoInfo.Owner

	// GitHubクライアントを作成（ghコマンドのみ使用）
	executor := gh.NewRealCommandExecutor()
	ghClient, err := gh.NewClient(executor)
	if err != nil {
		return fmt.Errorf("ghクライアントの作成に失敗: %w", err)
	}
	// 前提条件を検証
	if err := ghClient.ValidatePrerequisites(context.Background()); err != nil {
		return fmt.Errorf("ghコマンドの前提条件を満たしていません: %w", err)
	}
	githubClient := ghClient
	fmt.Fprintln(cmd.OutOrStdout(), "  GitHub接続: ghコマンドを使用")

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
	claudeExecutor := claude.NewClaudeExecutorWithLogger(appLogger)

	// TmuxManagerを作成
	tmuxManager := tmux.NewManager(appLogger)

	// ActionFactoryを作成
	actionFactory := watcher.NewDefaultActionFactory(
		sessionName,
		githubClient,
		tmuxManager,
		worktreeManager,
		claudeExecutor,
		claudeConfig,
		cfg,
		owner,
		repoName,
		appLogger,
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

// isDaemonMode はデーモンモードで起動されているかを確認します
func isDaemonMode() bool {
	return os.Getenv("OSOBA_DAEMON_MODE") == "1"
}

// startInBackground はプロセスをバックグラウンドで起動します
func startInBackground(cmd *cobra.Command, args []string) error {
	// 設定ファイルの存在チェック
	if err := checkConfigFileExists(cmd.OutOrStderr()); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Issue監視を開始しています...")

	// 設定を読み込む
	cfg := config.NewConfig()
	flagConfig, _ := cmd.Flags().GetString("config")
	actualConfigPath := cfg.LoadOrDefault(flagConfig)

	// ポーリング間隔を設定に反映
	intervalFlag, _ := cmd.Flags().GetString("interval")
	if intervalFlag != "" && intervalFlag != "5s" {
		interval, err := time.ParseDuration(intervalFlag)
		if err != nil {
			return fmt.Errorf("不正なポーリング間隔: %w", err)
		}
		cfg.GitHub.PollInterval = interval
	}

	// リポジトリ識別子を取得
	repoIdentifier, err := getRepoIdentifierFunc()
	if err != nil {
		return err
	}

	// リポジトリ情報を表示用に取得
	repoInfo, err := utils.GetGitHubRepoInfo(context.Background())
	if err != nil {
		return fmt.Errorf("リポジトリ情報の取得に失敗: %w", err)
	}

	// 基本情報を表示
	fmt.Fprintf(cmd.OutOrStdout(), "リポジトリ: %s/%s\n", repoInfo.Owner, repoInfo.Repo)
	fmt.Fprintf(cmd.OutOrStdout(), "ポーリング間隔: %s\n", cfg.GitHub.PollInterval)

	if actualConfigPath != "" {
		if flagConfig != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "設定ファイル: %s (指定されたファイル)\n", actualConfigPath)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "設定ファイル: %s (デフォルト)\n", actualConfigPath)
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "設定ファイル: なし (デフォルト値を使用)")
	}

	// パスマネージャを作成
	pm := paths.NewPathManager("")
	if err := pm.EnsureDirectories(); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗: %w", err)
	}

	pidFile := pm.PIDFile(repoIdentifier)

	// 既存のプロセスをチェック
	isRunning, err := checkExistingProcessFunc(pidFile)
	if err != nil {
		return err
	}
	if isRunning {
		// PIDファイルから既存プロセス情報を読み取り
		processInfo, readErr := daemon.ReadPIDFile(pidFile)
		if readErr == nil {
			return fmt.Errorf("既に実行中です (PID: %d)", processInfo.PID)
		}
		return fmt.Errorf("既に実行中です")
	}

	// DaemonManagerを使用してバックグラウンドで起動
	dm := daemon.NewDaemonManager()

	// 現在のコマンドライン引数を取得
	cmdArgs := os.Args[1:]

	// デーモンモードで起動
	if err := dm.Start(context.Background(), cmdArgs); err != nil {
		return fmt.Errorf("バックグラウンド起動に失敗: %w", err)
	}

	// この行は親プロセスでは実行されない（os.Exitされるため）
	// テスト時のみここに到達する
	fmt.Fprintln(cmd.OutOrStdout(), "バックグラウンドで起動しました")
	fmt.Fprintf(cmd.OutOrStdout(), "PID: %d\n", os.Getpid())
	return nil
}

// runInDaemonMode はデーモンモードでの実行を処理します
func runInDaemonMode(cmd *cobra.Command, pidFile string, intervalFlag, configFlag string) error {
	// PIDファイルを作成
	if err := createPIDFileFunc(pidFile); err != nil {
		return fmt.Errorf("PIDファイルの作成に失敗: %w", err)
	}

	// クリーンアップを設定
	defer os.Remove(pidFile)

	// ログファイルの設定
	repoIdentifier, _ := getRepoIdentifierFunc()
	pm := paths.NewPathManager("")
	logDir := pm.LogDir(repoIdentifier)

	// ログディレクトリを作成
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ログディレクトリの作成に失敗: %w", err)
	}

	// ログファイルパスを生成（日付ベース）
	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")

	// ログファイルを開く
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("ログファイルのオープンに失敗: %w", err)
	}
	defer f.Close()

	// 標準出力とエラー出力をリダイレクト
	cmd.SetOut(f)
	cmd.SetErr(f)

	// 通常の監視処理を実行
	return runWatchWithFlagsFunc(cmd, []string{}, intervalFlag, configFlag)
}

// checkExistingProcess は既存のプロセスが実行中かチェックします
func checkExistingProcess(pidFile string) (bool, error) {
	dm := daemon.NewDaemonManager()
	return dm.IsRunning(pidFile), nil
}

// createPIDFile はPIDファイルを作成します
func createPIDFile(pidFile string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	info := &daemon.ProcessInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		RepoPath:  cwd,
	}

	return daemon.WritePIDFile(pidFile, info)
}
