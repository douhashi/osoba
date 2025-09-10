package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/tmux"
)

func newResizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resize [issue-number]",
		Short: "tmuxペインのリサイズを実行",
		Long: `指定されたIssueウィンドウまたは現在のウィンドウのペインを均等にリサイズします。

Issue番号を指定する場合:
  osoba resize 123

現在のウィンドウをリサイズする場合:
  osoba resize

ドライランで実行内容を確認:
  osoba resize --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runResizeCmd(cmd, args)
		},
	}

	// フラグの追加
	cmd.Flags().Bool("dry-run", false, "実際にリサイズせず、実行内容のみ表示")
	cmd.Flags().String("session", "", "使用するtmuxセッション名を指定（省略時は設定から取得）")

	return cmd
}

func runResizeCmd(cmd *cobra.Command, args []string) error {
	// フラグの取得
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	sessionName, _ := cmd.Flags().GetString("session")

	// 設定を読み込み
	cfg := config.NewConfig()
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		configPath = viper.GetString("config")
	}
	if configPath != "" {
		_ = cfg.LoadOrDefault(configPath)
	} else {
		_ = cfg.LoadOrDefault("")
	}

	// セッション名が指定されていない場合はデフォルトを使用
	if sessionName == "" {
		sessionName = cfg.Tmux.SessionPrefix + "main"
	}

	// Issue番号の解析
	var issueNumber int
	var windowName string
	var err error

	if len(args) > 0 {
		// Issue番号が指定された場合
		issueNumber, err = strconv.Atoi(args[0])
		if err != nil || issueNumber <= 0 {
			return fmt.Errorf("無効なIssue番号: %s", args[0])
		}
		windowName = fmt.Sprintf("issue-%d", issueNumber)
	} else {
		// Issue番号が指定されていない場合、現在のtmuxウィンドウを検出
		windowName, err = detectCurrentWindow()
		if err != nil {
			return fmt.Errorf("現在のウィンドウを検出できませんでした: %w\n\nヒント: Issue番号を指定してください (例: osoba resize 123)", err)
		}
	}

	// tmuxがインストールされているかチェック
	if err := tmux.CheckTmuxInstalled(); err != nil {
		return fmt.Errorf("tmuxがインストールされていません: %w", err)
	}

	// tmuxマネージャーを作成
	manager := tmux.NewDefaultManager()

	// セッション存在確認
	exists, err := manager.SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("セッション確認エラー: %w", err)
	}
	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionName)
	}

	// ウィンドウ存在確認
	windows, err := tmux.ListWindows(sessionName)
	if err != nil {
		return fmt.Errorf("ウィンドウ一覧取得エラー: %w", err)
	}

	windowExists := false
	for _, window := range windows {
		if window.Name == windowName {
			windowExists = true
			break
		}
	}
	if !windowExists {
		return fmt.Errorf("ウィンドウ '%s' がセッション '%s' に見つかりません", windowName, sessionName)
	}

	// ペイン数を確認
	panes, err := manager.ListPanes(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("ペイン一覧取得エラー: %w", err)
	}

	if len(panes) <= 1 {
		fmt.Fprintf(cmd.OutOrStdout(), "✅ ウィンドウ '%s' のペイン数は %d 個です。リサイズは不要です。\n", windowName, len(panes))
		return nil
	}

	// 実行内容の表示
	fmt.Fprintf(cmd.OutOrStdout(), "🔧 ペインリサイズ実行\n")
	fmt.Fprintf(cmd.OutOrStdout(), "   セッション: %s\n", sessionName)
	fmt.Fprintf(cmd.OutOrStdout(), "   ウィンドウ: %s\n", windowName)
	fmt.Fprintf(cmd.OutOrStdout(), "   ペイン数: %d\n", len(panes))

	if dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "\n✨ ドライラン: 実際のリサイズは実行されませんでした\n")
		return nil
	}

	// リサイズ実行
	err = manager.ResizePanesEvenly(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("リサイズ実行エラー: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✅ リサイズが完了しました\n")

	// verboseモードの場合、詳細情報を表示
	if verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "\n📊 リサイズ後の状態:\n")
		panesAfter, err := manager.ListPanes(sessionName, windowName)
		if err == nil {
			for i, pane := range panesAfter {
				fmt.Fprintf(cmd.OutOrStdout(), "   ペイン%d: %dx%d\n", i, pane.Width, pane.Height)
			}
		}
	}

	return nil
}

// detectCurrentWindow は現在のtmuxウィンドウ名を検出する
func detectCurrentWindow() (string, error) {
	// TMUX環境変数が設定されているかチェック
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return "", fmt.Errorf("tmux環境内で実行されていません")
	}

	// 現在のウィンドウ名を取得
	cmd := exec.Command("tmux", "display-message", "-p", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ウィンドウ名の取得に失敗しました: %w", err)
	}

	windowName := strings.TrimSpace(string(output))
	if windowName == "" {
		return "", fmt.Errorf("ウィンドウ名が空です")
	}

	return windowName, nil
}
