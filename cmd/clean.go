package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	allFlag   bool
	forceFlag bool
)

func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [issue-number]",
		Short: "tmuxウィンドウとgit worktreeをクリーンアップ",
		Long: `Issue番号に関連するtmuxウィンドウとgit worktreeを削除します。

使用例:
  osoba clean 83        # Issue #83に関連するウィンドウとworktreeを削除
  osoba clean --all     # すべてのIssue関連リソースを削除（確認あり）
  osoba clean --force   # 確認なしで削除
  osoba clean --all --force  # すべてのリソースを確認なしで削除`,
		Args: validateCleanArgs,
		RunE: runClean,
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "すべてのIssue関連リソースを削除")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "確認プロンプトを表示せずに削除")

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

	// Issue番号に関連するworktreeを取得
	worktrees, err := listWorktreesForIssueFunc(context.Background(), issueNumber)
	if err != nil {
		return fmt.Errorf("worktree一覧の取得に失敗しました: %w", err)
	}

	if len(windows) == 0 && len(worktrees) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Issue #%d に関連するリソースが見つかりませんでした。\n", issueNumber)
		return nil
	}

	// 未コミット変更のチェック
	hasUncommittedChanges := false
	var uncommittedWorktrees []git.WorktreeInfo
	for _, wt := range worktrees {
		hasChanges, err := hasUncommittedChangesFunc(context.Background(), wt.Path)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "警告: %s の未コミット変更チェックに失敗しました: %v\n", wt.Path, err)
			continue
		}
		if hasChanges {
			hasUncommittedChanges = true
			uncommittedWorktrees = append(uncommittedWorktrees, wt)
		}
	}

	// 未コミット変更がある場合は警告を表示
	if hasUncommittedChanges {
		fmt.Fprintf(cmd.OutOrStdout(), "警告: 以下のworktreeに未コミットの変更があります:\n")
		for _, wt := range uncommittedWorktrees {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", wt.Path)
		}

		if !forceFlag {
			confirmed, err := confirmPromptFunc("本当に削除しますか？ (yes/no): ")
			if err != nil {
				return fmt.Errorf("確認の読み取りに失敗しました: %w", err)
			}
			if !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "削除をキャンセルしました。")
				return nil
			}
		}
	}

	// ウィンドウを削除
	windowErrors := []error{}
	if len(windows) > 0 {
		if err := killWindowsForIssueFunc(sessionName, issueNumber); err != nil {
			windowErrors = append(windowErrors, fmt.Errorf("ウィンドウの削除に失敗しました: %w", err))
		}
	}

	// worktreeを削除
	worktreeErrors := []error{}
	for _, wt := range worktrees {
		if err := removeWorktreeFunc(context.Background(), wt.Path); err != nil {
			worktreeErrors = append(worktreeErrors, fmt.Errorf("worktree %s の削除に失敗しました: %w", wt.Path, err))
		}
	}

	// 結果を表示
	if len(windows) > 0 || len(worktrees) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Issue #%d のリソースを削除しました:\n", issueNumber)
		if len(windows) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  ウィンドウ:\n")
			for _, window := range windows {
				fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", window.Name)
			}
		}
		if len(worktrees) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  worktree:\n")
			for _, wt := range worktrees {
				fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", wt.Path)
			}
		}
	}

	// エラーがあれば報告
	if len(windowErrors) > 0 || len(worktreeErrors) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n以下のエラーが発生しました:\n")
		for _, err := range windowErrors {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %v\n", err)
		}
		for _, err := range worktreeErrors {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %v\n", err)
		}
	}

	return nil
}

func cleanAllWindows(cmd *cobra.Command, sessionName string) error {
	// Issue関連のウィンドウをすべて取得
	windows, err := listWindowsByPatternFunc(sessionName, `^\d+-\w+$|^issue-\d+$`)
	if err != nil {
		return fmt.Errorf("ウィンドウ一覧の取得に失敗しました: %w", err)
	}

	// 全てのworktreeを取得
	allWorktrees, err := listAllWorktreesFunc(context.Background())
	if err != nil {
		return fmt.Errorf("worktree一覧の取得に失敗しました: %w", err)
	}

	// osoba関連のworktreeをフィルタリング
	var worktrees []git.WorktreeInfo
	for _, wt := range allWorktrees {
		if strings.Contains(wt.Path, ".git/worktree/") || strings.Contains(wt.Path, ".git/osoba/") {
			worktrees = append(worktrees, wt)
		}
	}

	if len(windows) == 0 && len(worktrees) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "削除対象のリソースが見つかりませんでした。")
		return nil
	}

	// リソース一覧を表示
	fmt.Fprintln(cmd.OutOrStdout(), "以下のリソースを削除します:")
	if len(windows) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  ウィンドウ:")
		for _, window := range windows {
			fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", window.Name)
		}
	}
	if len(worktrees) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  worktree:")
		for _, wt := range worktrees {
			fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", wt.Path)
		}
	}

	// 未コミット変更のチェック
	hasUncommittedChanges := false
	var uncommittedWorktrees []git.WorktreeInfo
	for _, wt := range worktrees {
		hasChanges, err := hasUncommittedChangesFunc(context.Background(), wt.Path)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "警告: %s の未コミット変更チェックに失敗しました: %v\n", wt.Path, err)
			continue
		}
		if hasChanges {
			hasUncommittedChanges = true
			uncommittedWorktrees = append(uncommittedWorktrees, wt)
		}
	}

	// 未コミット変更がある場合は警告を表示
	if hasUncommittedChanges {
		fmt.Fprintf(cmd.OutOrStdout(), "\n警告: 以下のworktreeに未コミットの変更があります:\n")
		for _, wt := range uncommittedWorktrees {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", wt.Path)
		}
	}

	// 確認プロンプト
	if !forceFlag {
		confirmed, err := confirmPromptFunc("本当に削除しますか？ (yes/no): ")
		if err != nil {
			return fmt.Errorf("確認の読み取りに失敗しました: %w", err)
		}

		if !confirmed {
			fmt.Fprintln(cmd.OutOrStdout(), "削除をキャンセルしました。")
			return nil
		}
	}

	// ウィンドウを削除
	windowErrors := []error{}
	if len(windows) > 0 {
		windowNames := getWindowNames(windows)
		if err := killWindowsFunc(sessionName, windowNames); err != nil {
			windowErrors = append(windowErrors, fmt.Errorf("ウィンドウの削除に失敗しました: %w", err))
		}
	}

	// worktreeを削除
	worktreeErrors := []error{}
	for _, wt := range worktrees {
		if err := removeWorktreeFunc(context.Background(), wt.Path); err != nil {
			worktreeErrors = append(worktreeErrors, fmt.Errorf("worktree %s の削除に失敗しました: %w", wt.Path, err))
		}
	}

	// 結果を表示
	if len(windows) > 0 || len(worktrees) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "以下のリソースを削除しました:")
		if len(windows) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  ウィンドウ:")
			for _, window := range windows {
				fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", window.Name)
			}
		}
		if len(worktrees) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  worktree:")
			for _, wt := range worktrees {
				fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", wt.Path)
			}
		}
	}

	// エラーがあれば報告
	if len(windowErrors) > 0 || len(worktreeErrors) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n以下のエラーが発生しました:\n")
		for _, err := range windowErrors {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %v\n", err)
		}
		for _, err := range worktreeErrors {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %v\n", err)
		}
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

// nullLogger は何もしないloggerの実装
type nullLogger struct{}

func (n *nullLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (n *nullLogger) Info(msg string, keysAndValues ...interface{})  {}
func (n *nullLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (n *nullLogger) Error(msg string, keysAndValues ...interface{}) {}
func (n *nullLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return n
}

// テスト時にモック可能な関数変数
var (
	checkTmuxInstalledFunc    = tmux.CheckTmuxInstalled
	getRepositoryNameFunc     = git.GetRepositoryName
	sessionExistsFunc         = tmux.SessionExists
	listWindowsForIssueFunc   = tmux.ListWindowsForIssue
	listWindowsByPatternFunc  = tmux.ListWindowsByPattern
	killWindowsForIssueFunc   = tmux.KillWindowsForIssue
	killWindowsFunc           = tmux.KillWindows
	confirmPromptFunc         = confirmPrompt
	listWorktreesForIssueFunc = createListWorktreesForIssueFunc()
	listAllWorktreesFunc      = createListAllWorktreesFunc()
	hasUncommittedChangesFunc = createHasUncommittedChangesFunc()
	removeWorktreeFunc        = createRemoveWorktreeFunc()
)

// WorktreeManagerのインスタンスを作成する関数
func createListWorktreesForIssueFunc() func(context.Context, int) ([]git.WorktreeInfo, error) {
	return func(ctx context.Context, issueNumber int) ([]git.WorktreeInfo, error) {
		// 実際の実装では、WorktreeManagerを使用
		nullLogger := &nullLogger{}
		repo := git.NewRepository(nullLogger)
		worktree := git.NewWorktree(nullLogger)
		branch := git.NewBranch(nullLogger)
		sync := git.NewSync(nullLogger)

		manager, err := git.NewWorktreeManager(repo, worktree, branch, sync)
		if err != nil {
			return nil, err
		}

		return manager.ListWorktreesForIssue(ctx, issueNumber)
	}
}

func createListAllWorktreesFunc() func(context.Context) ([]git.WorktreeInfo, error) {
	return func(ctx context.Context) ([]git.WorktreeInfo, error) {
		// 実際の実装では、WorktreeManagerを使用
		nullLogger := &nullLogger{}
		repo := git.NewRepository(nullLogger)
		worktree := git.NewWorktree(nullLogger)
		branch := git.NewBranch(nullLogger)
		sync := git.NewSync(nullLogger)

		manager, err := git.NewWorktreeManager(repo, worktree, branch, sync)
		if err != nil {
			return nil, err
		}

		return manager.ListAllWorktrees(ctx)
	}
}

func createHasUncommittedChangesFunc() func(context.Context, string) (bool, error) {
	return func(ctx context.Context, worktreePath string) (bool, error) {
		// 実際の実装では、WorktreeManagerを使用
		nullLogger := &nullLogger{}
		repo := git.NewRepository(nullLogger)
		worktree := git.NewWorktree(nullLogger)
		branch := git.NewBranch(nullLogger)
		sync := git.NewSync(nullLogger)

		manager, err := git.NewWorktreeManager(repo, worktree, branch, sync)
		if err != nil {
			return false, err
		}

		return manager.HasUncommittedChanges(ctx, worktreePath)
	}
}

func createRemoveWorktreeFunc() func(context.Context, string) error {
	return func(ctx context.Context, worktreePath string) error {
		// 実際の実装では、Worktreeを使用
		nullLogger := &nullLogger{}
		worktree := git.NewWorktree(nullLogger)

		// リポジトリのルートパスを取得
		repo := git.NewRepository(nullLogger)
		basePath, err := repo.GetRootPath(ctx)
		if err != nil {
			return err
		}

		return worktree.Remove(ctx, basePath, worktreePath)
	}
}
