package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/douhashi/osoba/internal/tmux"
	"github.com/spf13/cobra"
)

var allFlag bool

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [issue-number]",
		Short: "tmuxウィンドウをクリーンアップ",
		Long: `Issue番号に関連するtmuxウィンドウを削除します。

使用例:
  osoba clean 83        # Issue #83に関連するウィンドウを削除
  osoba clean --all     # すべてのIssue関連ウィンドウを削除（確認あり）`,
		Args: validateCleanArgs,
		RunE: runClean,
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "すべてのIssue関連ウィンドウを削除")

	return cmd
}

func validateCleanArgs(cmd *cobra.Command, args []string) error {
	if allFlag {
		if len(args) > 0 {
			return fmt.Errorf("--all オプションを使用する場合は引数を指定しないでください")
		}
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("Issue番号を指定するか、--all オプションを使用してください")
	}

	if len(args) > 1 {
		return fmt.Errorf("引数は1つだけ指定してください")
	}

	return nil
}

func runClean(cmd *cobra.Command, args []string) error {
	// 1. tmuxがインストールされているか確認
	if err := checkTmuxInstalledFunc(); err != nil {
		return err
	}

	// 2. Gitリポジトリ名を取得
	repoName, err := getRepositoryNameFunc()
	if err != nil {
		return err
	}

	// 3. セッション名を生成
	sessionName := fmt.Sprintf("osoba-%s", repoName)

	// 4. セッションが存在するか確認
	exists, err := sessionExistsFunc(sessionName)
	if err != nil {
		return fmt.Errorf("セッションの確認に失敗しました: %w", err)
	}

	if !exists {
		return fmt.Errorf("セッション '%s' が見つかりません", sessionName)
	}

	// 5. 処理分岐
	if allFlag {
		return cleanAllWindows(cmd, sessionName)
	}

	// Issue番号の解析
	issueNumber, err := parseIssueNumber(args[0])
	if err != nil {
		return err
	}

	return cleanIssueWindows(cmd, sessionName, issueNumber)
}

func parseIssueNumber(arg string) (int, error) {
	num, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("Issue番号は正の整数で指定してください")
	}
	if num <= 0 {
		return 0, fmt.Errorf("Issue番号は正の整数で指定してください")
	}
	return num, nil
}

func cleanIssueWindows(cmd *cobra.Command, sessionName string, issueNumber int) error {
	// Issue番号に関連するウィンドウを取得
	windows, err := listWindowsForIssueFunc(sessionName, issueNumber)
	if err != nil {
		return fmt.Errorf("ウィンドウ一覧の取得に失敗しました: %w", err)
	}

	if len(windows) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Issue #%d に関連するウィンドウが見つかりませんでした。\n", issueNumber)
		return nil
	}

	// ウィンドウを削除
	if err := killWindowsForIssueFunc(sessionName, issueNumber); err != nil {
		return fmt.Errorf("ウィンドウの削除に失敗しました: %w", err)
	}

	// 削除したウィンドウを表示
	fmt.Fprintf(cmd.OutOrStdout(), "Issue #%d のウィンドウを削除しました:\n", issueNumber)
	for _, window := range windows {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", window.Name)
	}

	return nil
}

func cleanAllWindows(cmd *cobra.Command, sessionName string) error {
	// Issue関連のウィンドウをすべて取得（パターン: 数字-フェーズ）
	windows, err := listWindowsByPatternFunc(sessionName, `^\d+-\w+$`)
	if err != nil {
		return fmt.Errorf("ウィンドウ一覧の取得に失敗しました: %w", err)
	}

	if len(windows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "削除対象のウィンドウが見つかりませんでした。")
		return nil
	}

	// ウィンドウ一覧を表示して確認
	fmt.Fprintln(cmd.OutOrStdout(), "以下のウィンドウを削除します:")
	for _, window := range windows {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", window.Name)
	}

	// 確認プロンプト
	confirmed, err := confirmPromptFunc("本当に削除しますか？ (yes/no): ")
	if err != nil {
		return fmt.Errorf("確認の読み取りに失敗しました: %w", err)
	}

	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "削除をキャンセルしました。")
		return nil
	}

	// ウィンドウ名のリストを作成
	windowNames := getWindowNames(windows)

	// ウィンドウを削除
	if err := killWindowsFunc(sessionName, windowNames); err != nil {
		return fmt.Errorf("ウィンドウの削除に失敗しました: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "以下のウィンドウを削除しました:")
	for _, window := range windows {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", window.Name)
	}

	return nil
}

func getWindowNames(windows []*tmux.WindowInfo) []string {
	if windows == nil {
		return []string{}
	}

	names := make([]string, len(windows))
	for i, window := range windows {
		names[i] = window.Name
	}
	return names
}

func confirmPrompt(prompt string) (bool, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "yes" || response == "y", nil
}

// テスト時にモック可能な関数変数
var (
	listWindowsForIssueFunc  = tmux.ListWindowsForIssue
	listWindowsByPatternFunc = tmux.ListWindowsByPattern
	killWindowsForIssueFunc  = tmux.KillWindowsForIssue
	killWindowsFunc          = tmux.KillWindows
	confirmPromptFunc        = confirmPrompt
)
