package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/daemon"
	githubClient "github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/utils"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "現在の状態を表示",
		Long:  `実行中の開発セッションとその状態を表示します。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatusCmd(cmd)
		},
	}

	// --debugフラグを追加
	cmd.Flags().Bool("debug", false, "詳細な診断情報を表示")

	return cmd
}

func runStatusCmd(cmd *cobra.Command) error {
	ctx := context.Background()

	fmt.Fprintln(cmd.OutOrStdout(), "=== osobaステータス ===")
	fmt.Fprintln(cmd.OutOrStdout())

	// 設定を読み込み
	cfg := config.NewConfig()

	// rootコマンドで読み込まれた設定ファイルのパスを取得
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		// -cフラグが指定されている場合はそれを使用
		configPath = viper.GetString("config")
	}

	// 設定ファイルのパスが取得できた場合、またはデフォルトパスから読み込み
	if configPath != "" {
		_ = cfg.LoadOrDefault(configPath)
	} else {
		// configPathが空の場合もデフォルト設定ファイルをチェック
		_ = cfg.LoadOrDefault("")
	}

	// tmuxがインストールされているかチェック
	if err := tmux.CheckTmuxInstalled(); err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "⚠️  tmuxがインストールされていません")
		fmt.Fprintln(cmd.OutOrStdout(), "   ", err.Error())
		return nil
	}

	// debugフラグの確認
	debugMode, _ := cmd.Flags().GetBool("debug")

	// tmuxセッション一覧を取得
	sessions, err := tmux.ListSessionsAsSessionInfo(cfg.Tmux.SessionPrefix)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "⚠️  tmuxセッション取得エラー: %v\n", err)
	} else {
		if debugMode {
			displayTmuxSessionsWithDiagnostics(cmd, sessions, cfg.Tmux.SessionPrefix)
		} else {
			displayTmuxSessions(cmd, sessions)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())

	// バックグラウンドプロセスの状態を表示
	displayBackgroundProcess(cmd)

	fmt.Fprintln(cmd.OutOrStdout())

	// GitHubリポジトリ情報を取得（共通関数を使用）
	repoInfo, err := utils.GetGitHubRepoInfo(ctx)
	if err != nil {
		// 詳細なエラーメッセージを表示
		if repoErr, ok := err.(*utils.GetGitHubRepoInfoError); ok {
			switch repoErr.Step {
			case "working_directory":
				fmt.Fprintf(cmd.OutOrStdout(), "⚠️  作業ディレクトリの取得に失敗しました: %v\n", repoErr.Cause)
			case "git_directory":
				fmt.Fprintf(cmd.OutOrStdout(), "⚠️  Gitリポジトリが見つかりません。Gitリポジトリのルートディレクトリで実行してください\n")
			case "remote_url":
				fmt.Fprintf(cmd.OutOrStdout(), "⚠️  リモートURL取得に失敗しました: %v\n", repoErr.Cause)
				fmt.Fprintf(cmd.OutOrStdout(), "   'git remote add origin <URL>' でリモートを設定してください\n")
			case "url_parsing":
				fmt.Fprintf(cmd.OutOrStdout(), "⚠️  GitHub URL解析に失敗しました: %v\n", repoErr.Cause)
				fmt.Fprintf(cmd.OutOrStdout(), "   GitHubのリポジトリURLが正しく設定されているか確認してください\n")
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "⚠️  GitHubリポジトリ情報取得エラー: %v\n", err)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "⚠️  GitHubリポジトリ情報取得エラー: %v\n", err)
		}
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "📂 リポジトリ: %s/%s\n", repoInfo.Owner, repoInfo.Repo)
	fmt.Fprintln(cmd.OutOrStdout())

	// 設定値を表示
	if err := displayConfiguration(cmd, cfg); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "⚠️  設定表示エラー: %v\n", err)
	}

	// GitHub認証が利用可能かチェック
	token, _ := config.GetGitHubToken(cfg)
	if token == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "⚠️  GitHub認証が設定されていません")
		fmt.Fprintln(cmd.OutOrStdout(), "   詳細なステータス情報を表示するには、以下のコマンドで認証してください:")
		fmt.Fprintln(cmd.OutOrStdout(), "   gh auth login")
		return nil
	}

	// GitHub クライアントを作成（ghコマンドのみ使用）
	client, err := githubClient.NewClient("")
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "⚠️  GitHub クライアント作成エラー: %v\n", err)
		return nil
	}

	// 各ステータスラベルのIssueを取得して表示
	if err := displayGitHubIssues(cmd, ctx, client, repoInfo, cfg); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "⚠️  GitHub Issue取得エラー: %v\n", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())

	// 自動マージメトリクスを表示
	displayAutoMergeMetrics(cmd, cfg)

	return nil
}

func displayTmuxSessions(cmd *cobra.Command, sessions []*tmux.SessionInfo) {
	fmt.Fprintln(cmd.OutOrStdout(), "🖥️  tmuxセッション:")
	if len(sessions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "   実行中のセッションはありません")
		return
	}

	for _, session := range sessions {
		status := "detached"
		if session.Attached {
			status = "attached"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "   📺 %s (%d windows, %s)\n",
			session.Name, session.Windows, status)

		// セッション内のウィンドウ詳細を表示
		displaySessionWindows(cmd, session.Name)
	}
}

func displayTmuxSessionsWithDiagnostics(cmd *cobra.Command, sessions []*tmux.SessionInfo, prefix string) {
	fmt.Fprintln(cmd.OutOrStdout(), "🖥️  tmuxセッション（診断モード）:")

	// tmuxマネージャーを作成
	manager := tmux.NewDefaultManager()

	// セッション診断情報を取得
	diagnostics, err := manager.ListSessionDiagnostics(prefix)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "   ⚠️  診断情報取得エラー: %v\n", err)
		// 通常モードにフォールバック
		displayTmuxSessions(cmd, sessions)
		return
	}

	if len(diagnostics) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "   実行中のセッションはありません")
		return
	}

	for _, diag := range diagnostics {
		status := "detached"
		if diag.Attached {
			status = "attached"
		}

		// エラーがある場合は警告マークを表示
		errorIndicator := ""
		if len(diag.Errors) > 0 {
			errorIndicator = " ⚠️"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "   📺 %s (%d windows, %s)%s\n",
			diag.Name, diag.Windows, status, errorIndicator)

		// デバッグ情報を表示
		fmt.Fprintf(cmd.OutOrStdout(), "      Created: %s\n", diag.Created)
		fmt.Fprintf(cmd.OutOrStdout(), "      Timestamp: %s\n", diag.Timestamp.Format("2006-01-02 15:04:05"))

		if len(diag.Errors) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "      Errors:")
			for _, errMsg := range diag.Errors {
				fmt.Fprintf(cmd.OutOrStdout(), "        - %s\n", errMsg)
			}
		}

		// メタデータを表示
		if len(diag.Metadata) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "      Metadata:")
			for key, value := range diag.Metadata {
				fmt.Fprintf(cmd.OutOrStdout(), "        %s: %s\n", key, value)
			}
		}

		// セッション内のウィンドウ詳細を診断モードで表示
		displaySessionWindowsWithDiagnostics(cmd, diag.Name, manager)
		fmt.Fprintln(cmd.OutOrStdout())
	}
}

func displaySessionWindowsWithDiagnostics(cmd *cobra.Command, sessionName string, manager *tmux.DefaultManager) {
	// ウィンドウ診断情報を取得
	windowDiags, err := manager.ListWindowDiagnostics(sessionName)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "      ⚠️  ウィンドウ診断エラー: %v\n", err)
		return
	}

	if len(windowDiags) == 0 {
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n     Windows (%d) - 診断情報:\n", len(windowDiags))
	for _, diag := range windowDiags {
		activeMarker := ""
		if diag.Active {
			activeMarker = " [active]"
		}

		errorIndicator := ""
		if len(diag.Errors) > 0 {
			errorIndicator = " ⚠️"
		}

		// Issue番号とフェーズが取得できた場合は詳細表示
		if diag.IssueNumber > 0 && diag.Phase != "" {
			phaseDisplay := getPhaseDisplay(diag.Phase)
			fmt.Fprintf(cmd.OutOrStdout(), "       %s  Issue #%d (%s)%s%s\n",
				diag.Name, diag.IssueNumber, phaseDisplay, activeMarker, errorIndicator)
		} else {
			// パースできない場合はウィンドウ名のみ表示
			fmt.Fprintf(cmd.OutOrStdout(), "       %s%s%s\n", diag.Name, activeMarker, errorIndicator)
		}

		// 診断詳細情報
		fmt.Fprintf(cmd.OutOrStdout(), "         Index: %d, Panes: %d, Exists: %v\n",
			diag.Index, diag.Panes, diag.Exists)
		fmt.Fprintf(cmd.OutOrStdout(), "         Timestamp: %s\n",
			diag.Timestamp.Format("2006-01-02 15:04:05"))

		if len(diag.Errors) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "         Errors:")
			for _, errMsg := range diag.Errors {
				fmt.Fprintf(cmd.OutOrStdout(), "           - %s\n", errMsg)
			}
		}

		if len(diag.Metadata) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "         Metadata:")
			for key, value := range diag.Metadata {
				fmt.Fprintf(cmd.OutOrStdout(), "           %s: %s\n", key, value)
			}
		}
	}
}

func displaySessionWindows(cmd *cobra.Command, sessionName string) {
	// ウィンドウ詳細情報を取得
	details, err := tmux.GetSortedWindowDetails(sessionName)
	if err != nil {
		// エラーが発生した場合はログに記録するが、表示は継続
		if logger := getLogger(); logger != nil {
			logger.Debug("ウィンドウ詳細取得エラー",
				"session_name", sessionName,
				"error", err)
		}
		return
	}

	if len(details) == 0 {
		return
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n     Windows (%d):\n", len(details))
	for _, detail := range details {
		activeMarker := ""
		if detail.Active {
			activeMarker = " [active]"
		}

		// Issue番号とフェーズが取得できた場合は詳細表示
		if detail.IssueNumber > 0 && detail.Phase != "" {
			phaseDisplay := getPhaseDisplay(detail.Phase)
			fmt.Fprintf(cmd.OutOrStdout(), "       %s  Issue #%d (%s)%s\n",
				detail.Name, detail.IssueNumber, phaseDisplay, activeMarker)
		} else {
			// パースできない場合はウィンドウ名のみ表示
			fmt.Fprintf(cmd.OutOrStdout(), "       %s%s\n", detail.Name, activeMarker)
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())
}

func getPhaseDisplay(phase string) string {
	switch phase {
	case "plan":
		return "Planning"
	case "implement":
		return "Implementing"
	case "review":
		return "Reviewing"
	default:
		return phase
	}
}

func getLogger() logger.Logger {
	// ロガーを取得
	log, _ := logger.New()
	return log
}

func displayGitHubIssues(cmd *cobra.Command, ctx context.Context, client githubClient.GitHubClient, repoInfo *utils.GitHubRepoInfo, cfg *config.Config) error {
	statusLabels := []string{
		"status:planning",
		"status:implementing",
		"status:reviewing",
		"status:needs-plan",
		"status:ready",
		"status:review-requested",
	}

	fmt.Fprintln(cmd.OutOrStdout(), "📋 Issues:")

	hasIssues := false
	for _, label := range statusLabels {
		issues, err := client.ListIssuesByLabels(ctx, repoInfo.Owner, repoInfo.Repo, []string{label})
		if err != nil {
			return fmt.Errorf("ラベル '%s' のIssue取得に失敗: %w", label, err)
		}

		if len(issues) > 0 {
			hasIssues = true
			displayIssuesForLabel(cmd, label, issues)
		}
	}

	if !hasIssues {
		fmt.Fprintln(cmd.OutOrStdout(), "   処理中のIssueはありません")
	}

	return nil
}

func displayIssuesForLabel(cmd *cobra.Command, label string, issues []*githubClient.Issue) {
	emoji := getEmojiForLabel(label)
	fmt.Fprintf(cmd.OutOrStdout(), "   %s %s:\n", emoji, label)

	for _, issue := range issues {
		title := *issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		fmt.Fprintf(cmd.OutOrStdout(), "     #%d %s\n", *issue.Number, title)
	}
}

func getEmojiForLabel(label string) string {
	switch label {
	case "status:needs-plan":
		return "📝"
	case "status:planning":
		return "🔄"
	case "status:ready":
		return "✅"
	case "status:implementing":
		return "🔨"
	case "status:review-requested":
		return "👀"
	case "status:reviewing":
		return "🔍"
	default:
		return "📌"
	}
}

// maskSensitiveValue は機密情報をマスクして表示用に変換する
func maskSensitiveValue(value string) string {
	if value == "" {
		return ""
	}

	if len(value) <= 3 {
		return strings.Repeat("*", len(value))
	}

	if len(value) <= 6 {
		return value[:3] + strings.Repeat("*", len(value)-3)
	}

	return value[:3] + strings.Repeat("*", len(value)-3)
}

// displayConfiguration は設定値を表示する
func displayConfiguration(cmd *cobra.Command, cfg *config.Config) error {
	configPath := viper.GetString("config")

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "📋 Configuration:")

	// 設定ファイルが指定されているかチェック
	if configPath != "" {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Fprintf(cmd.OutOrStdout(), "⚠️  Configuration file not found: %s\n", configPath)
			fmt.Fprintln(cmd.OutOrStdout(), "   Using default values")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "📄 Config file: %s\n", configPath)
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "📄 Config file: (using defaults)")
	}

	fmt.Fprintln(cmd.OutOrStdout())

	// GitHub設定
	fmt.Fprintln(cmd.OutOrStdout(), "  GitHub:")

	// トークンと取得元を表示
	token, source := config.GetGitHubToken(cfg)
	if token == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "    Token: (not set)")
	} else {
		maskedToken := maskSensitiveValue(token)
		fmt.Fprintf(cmd.OutOrStdout(), "    Token: %s (from %s)\n", maskedToken, source)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "    Poll Interval: %v\n", cfg.GitHub.PollInterval)

	// ラベル設定
	fmt.Fprintln(cmd.OutOrStdout(), "    Labels:")
	fmt.Fprintf(cmd.OutOrStdout(), "      Plan: %s\n", cfg.GitHub.Labels.Plan)
	fmt.Fprintf(cmd.OutOrStdout(), "      Ready: %s\n", cfg.GitHub.Labels.Ready)
	fmt.Fprintf(cmd.OutOrStdout(), "      Review: %s\n", cfg.GitHub.Labels.Review)

	// メッセージ設定
	fmt.Fprintln(cmd.OutOrStdout(), "    Messages:")
	fmt.Fprintf(cmd.OutOrStdout(), "      Plan: %s\n", cfg.GitHub.Messages.Plan)
	fmt.Fprintf(cmd.OutOrStdout(), "      Implement: %s\n", cfg.GitHub.Messages.Implement)
	fmt.Fprintf(cmd.OutOrStdout(), "      Review: %s\n", cfg.GitHub.Messages.Review)

	fmt.Fprintln(cmd.OutOrStdout())

	// TMux設定
	fmt.Fprintln(cmd.OutOrStdout(), "  TMux:")
	fmt.Fprintf(cmd.OutOrStdout(), "    Session Prefix: %s\n", cfg.Tmux.SessionPrefix)

	fmt.Fprintln(cmd.OutOrStdout())

	// Claude設定
	if cfg.Claude != nil && cfg.Claude.Phases != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "  Claude Phases:")

		// フェーズ順序を定義
		phases := []string{"plan", "implement", "review"}

		for _, phaseName := range phases {
			if phaseConfig, exists := cfg.Claude.Phases[phaseName]; exists && phaseConfig != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "    %s:\n", strings.Title(phaseName))
				fmt.Fprintf(cmd.OutOrStdout(), "      Args: %v\n", phaseConfig.Args)
				fmt.Fprintf(cmd.OutOrStdout(), "      Prompt: %s\n", phaseConfig.Prompt)
			}
		}
	}

	return nil
}

// displayBackgroundProcess はバックグラウンドプロセスの状態を表示します
func displayBackgroundProcess(cmd *cobra.Command) {
	fmt.Fprintln(cmd.OutOrStdout(), "🔄 バックグラウンドプロセス:")

	// リポジトリ識別子を取得
	repoIdentifier, err := getRepoIdentifier()
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "   ⚠️  リポジトリ情報の取得に失敗しました")
		return
	}

	// パスマネージャを作成
	pm := paths.NewPathManager("")
	pidFile := pm.PIDFile(repoIdentifier)

	// プロセスの状態を確認
	dm := daemon.NewDaemonManager()
	status, err := dm.Status(pidFile)
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "   実行中のプロセスはありません")
		return
	}

	if !status.Running {
		fmt.Fprintln(cmd.OutOrStdout(), "   実行中のプロセスはありません")
		return
	}

	// 実行時間を計算
	uptime := time.Since(status.StartTime)
	uptimeStr := formatDuration(uptime)

	fmt.Fprintf(cmd.OutOrStdout(), "   PID: %d (実行時間: %s)\n", status.PID, uptimeStr)
	fmt.Fprintf(cmd.OutOrStdout(), "   リポジトリ: %s\n", status.RepoPath)

	// ログファイルのパスを表示
	logDir := pm.LogDir(repoIdentifier)
	logFile := fmt.Sprintf("%s/%s.log", logDir, time.Now().Format("2006-01-02"))
	fmt.Fprintf(cmd.OutOrStdout(), "   ログファイル: %s\n", logFile)
}

// formatDuration は期間を人間が読みやすい形式にフォーマットします
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分", int(d.Minutes()))
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes == 0 {
		return fmt.Sprintf("%d時間", hours)
	}
	return fmt.Sprintf("%d時間%d分", hours, minutes)
}

// displayAutoMergeMetrics は自動マージメトリクスを表示する
func displayAutoMergeMetrics(cmd *cobra.Command, cfg *config.Config) {
	fmt.Fprintln(cmd.OutOrStdout(), "🔀 自動マージメトリクス:")

	// 自動マージ機能が無効な場合
	if !cfg.GitHub.AutoMergeLGTM {
		fmt.Fprintln(cmd.OutOrStdout(), "   自動マージ機能が無効です")
		return
	}

	// リポジトリ識別子を取得
	repoIdentifier, err := getRepoIdentifier()
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "   ⚠️  リポジトリ情報の取得に失敗しました")
		return
	}

	// バックグラウンドプロセスが実行中かチェック
	pm := paths.NewPathManager("")
	pidFile := pm.PIDFile(repoIdentifier)

	dm := daemon.NewDaemonManager()
	status, err := dm.Status(pidFile)
	if err != nil || !status.Running {
		fmt.Fprintln(cmd.OutOrStdout(), "   バックグラウンドプロセスが実行されていないため、メトリクスが利用できません")
		return
	}

	// メトリクス情報をファイルから取得（実装は将来的に追加予定）
	// 現在は実行中であることのみ表示
	fmt.Fprintln(cmd.OutOrStdout(), "   自動マージ機能が有効で、バックグラウンドプロセスが実行中です")
	fmt.Fprintln(cmd.OutOrStdout(), "   詳細なメトリクス表示は今後のバージョンで追加予定です")
}
