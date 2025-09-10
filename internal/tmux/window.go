package tmux

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// CommandExecutor はコマンド実行のインターフェース
type CommandExecutor interface {
	Execute(cmd string, args ...string) (string, error)
}

// DefaultCommandExecutor はデフォルトのコマンド実行実装
type DefaultCommandExecutor struct{}

// Execute はコマンドを実行する
func (e *DefaultCommandExecutor) Execute(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	output, err := command.Output()
	return string(output), err
}

// CreateWindow は新しいtmuxウィンドウを作成する
// この関数はglobal_manager.goのラッパー関数に置き換えられました

// CreateWindowWithExecutor はExecutorを使用して新しいtmuxウィンドウを作成する
func CreateWindowWithExecutor(sessionName, windowName string, executor CommandExecutor) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ作成開始",
			"operation", "create_window",
			"session_name", sessionName,
			"window_name", windowName,
			"command", "tmux new-window",
			"args", []string{"-t", sessionName, "-n", windowName})
	}

	_, err := executor.Execute("tmux", "new-window", "-t", sessionName, "-n", windowName)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウ作成失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"error", err)
		}
		return fmt.Errorf("failed to create window '%s' in session '%s': %w", windowName, sessionName, err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ作成完了",
			"session_name", sessionName,
			"window_name", windowName)
	}

	return nil
}

// SwitchToWindow は指定されたウィンドウに切り替える
// この関数はglobal_manager.goのラッパー関数に置き換えられました

// SwitchToWindowWithExecutor はExecutorを使用して指定されたウィンドウに切り替える
func SwitchToWindowWithExecutor(sessionName, windowName string, executor CommandExecutor) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}

	target := fmt.Sprintf("%s:%s", sessionName, windowName)

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ切り替え",
			"operation", "switch_window",
			"session_name", sessionName,
			"window_name", windowName,
			"target", target,
			"command", "tmux select-window",
			"args", []string{"-t", target})
	}

	_, err := executor.Execute("tmux", "select-window", "-t", target)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウ切り替え失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"target", target,
				"error", err)
		}
		return fmt.Errorf("failed to switch to window '%s' in session '%s': %w", windowName, sessionName, err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ切り替え完了",
			"session_name", sessionName,
			"window_name", windowName)
	}

	return nil
}

// WindowExists は指定されたウィンドウが存在するかを確認する
// この関数はglobal_manager.goのラッパー関数に置き換えられました

// WindowExistsWithExecutor はExecutorを使用して指定されたウィンドウが存在するかを確認する
func WindowExistsWithExecutor(sessionName, windowName string, executor CommandExecutor) (bool, error) {
	if sessionName == "" {
		return false, fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return false, fmt.Errorf("window name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxウィンドウ一覧取得",
			"operation", "list_windows",
			"session_name", sessionName,
			"target_window", windowName,
			"command", "tmux list-windows",
			"args", []string{"-t", sessionName, "-F", "#{window_name}"})
	}

	output, err := executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}")
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウ一覧取得失敗",
				"session_name", sessionName,
				"error", err)
		}
		return false, fmt.Errorf("failed to list windows in session '%s': %w", sessionName, err)
	}

	windows := strings.Split(strings.TrimSpace(output), "\n")
	exists := false
	for _, window := range windows {
		if window == windowName {
			exists = true
			break
		}
	}

	if logger := GetLogger(); logger != nil {
		if exists {
			logger.Debug("ウィンドウが存在します",
				"session_name", sessionName,
				"window_name", windowName)
		} else {
			logger.Debug("ウィンドウが存在しません",
				"session_name", sessionName,
				"window_name", windowName)
		}
	}

	return exists, nil
}

// GetWindowName はIssue番号からウィンドウ名を生成する
func GetWindowName(issueNumber int) string {
	return fmt.Sprintf("issue-%d", issueNumber)
}

// GetWindowNameWithPhase はIssue番号とフェーズからウィンドウ名を生成する
func GetWindowNameWithPhase(issueNumber int, phase string) (string, error) {
	// 有効なフェーズのチェック
	validPhases := map[string]bool{
		"plan":      true,
		"implement": true,
		"review":    true,
	}

	if phase == "" {
		return "", fmt.Errorf("invalid phase: phase cannot be empty")
	}

	if !validPhases[phase] {
		return "", fmt.Errorf("invalid phase: %s (valid phases are: plan, implement, review)", phase)
	}

	return fmt.Sprintf("%d-%s", issueNumber, phase), nil
}

// CreateWindowForIssue はIssue番号に基づいてウィンドウを作成する
func CreateWindowForIssue(sessionName string, issueNumber int) error {
	return CreateIssueWindowWithExecutor(sessionName, issueNumber, &DefaultCommandExecutor{})
}

// CreateIssueWindowWithExecutor はExecutorを使用してIssue番号に基づいてウィンドウを作成する
func CreateIssueWindowWithExecutor(sessionName string, issueNumber int, executor CommandExecutor) error {
	windowName := GetWindowName(issueNumber)

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue用ウィンドウ作成開始",
			"operation", "create_window_for_issue",
			"session_name", sessionName,
			"issue_number", issueNumber,
			"window_name", windowName)
	}

	// ウィンドウが既に存在する場合はスキップ
	exists, err := WindowExistsWithExecutor(sessionName, windowName, executor)
	if err != nil {
		return fmt.Errorf("failed to check window existence: %w", err)
	}
	if exists {
		if logger := GetLogger(); logger != nil {
			logger.Info("ウィンドウは既に存在します",
				"session_name", sessionName,
				"window_name", windowName)
		}
		return nil
	}

	return CreateWindowWithExecutor(sessionName, windowName, executor)
}

// SwitchToIssueWindow はIssue番号に基づいてウィンドウに切り替える
func SwitchToIssueWindow(sessionName string, issueNumber int) error {
	windowName := GetWindowName(issueNumber)

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue用ウィンドウへ切り替え",
			"operation", "switch_to_issue_window",
			"session_name", sessionName,
			"issue_number", issueNumber,
			"window_name", windowName)
	}

	return SwitchToWindow(sessionName, windowName)
}

// KillWindow は指定されたウィンドウを削除する
// この関数はglobal_manager.goのラッパー関数に置き換えられました

// KillWindowWithExecutor はExecutorを使用して指定されたウィンドウを削除する
func KillWindowWithExecutor(sessionName, windowName string, executor CommandExecutor) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}

	target := fmt.Sprintf("%s:%s", sessionName, windowName)

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ削除開始",
			"operation", "kill_window",
			"session_name", sessionName,
			"window_name", windowName,
			"target", target,
			"command", "tmux kill-window",
			"args", []string{"-t", target})
	}

	_, err := executor.Execute("tmux", "kill-window", "-t", target)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウ削除失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"target", target,
				"error", err)
		}
		return fmt.Errorf("failed to kill window '%s' in session '%s': %w", windowName, sessionName, err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ削除完了",
			"session_name", sessionName,
			"window_name", windowName)
	}

	return nil
}

// CreateOrReplaceWindow は既存のウィンドウを削除してから新しいウィンドウを作成する
// この関数はglobal_manager.goのラッパー関数に置き換えられました

// CreateOrReplaceWindowWithExecutor はExecutorを使用して既存のウィンドウを削除してから新しいウィンドウを作成する
func CreateOrReplaceWindowWithExecutor(sessionName, windowName string, executor CommandExecutor) error {
	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ作成/置換開始",
			"operation", "create_or_replace_window",
			"session_name", sessionName,
			"window_name", windowName)
	}

	// ウィンドウの存在確認
	exists, err := WindowExistsWithExecutor(sessionName, windowName, executor)
	if err != nil {
		return fmt.Errorf("failed to check window existence: %w", err)
	}

	// 既存のウィンドウが存在する場合は削除
	if exists {
		if logger := GetLogger(); logger != nil {
			logger.Info("既存のウィンドウを削除します",
				"session_name", sessionName,
				"window_name", windowName)
		}

		if err := KillWindowWithExecutor(sessionName, windowName, executor); err != nil {
			return fmt.Errorf("failed to kill existing window: %w", err)
		}
	}

	// 新しいウィンドウを作成
	if err := CreateWindowWithExecutor(sessionName, windowName, executor); err != nil {
		return fmt.Errorf("failed to create new window: %w", err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ作成/置換完了",
			"session_name", sessionName,
			"window_name", windowName)
	}

	return nil
}

// WindowInfo はtmuxウィンドウの情報を保持する構造体
type WindowInfo struct {
	Index  int    // ウィンドウインデックス
	Name   string // ウィンドウ名
	Active bool   // アクティブウィンドウかどうか
	Panes  int    // ペイン数
}

// ListWindows は指定されたセッション内のウィンドウ一覧を取得する
func ListWindows(sessionName string) ([]*WindowInfo, error) {
	return ListWindowsWithExecutor(sessionName, &DefaultCommandExecutor{})
}

// ListWindowsWithExecutor はExecutorを使用して指定されたセッション内のウィンドウ一覧を取得する
func ListWindowsWithExecutor(sessionName string, executor CommandExecutor) ([]*WindowInfo, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxウィンドウ一覧取得",
			"operation", "list_windows",
			"session_name", sessionName,
			"command", "tmux list-windows")
	}

	// tmux list-windows -t session -F "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"
	output, err := executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}")
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウ一覧取得失敗",
				"session_name", sessionName,
				"error", err)
		}
		return nil, fmt.Errorf("failed to list windows in session '%s': %w", sessionName, err)
	}

	windows := []*WindowInfo{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			windowInfo := &WindowInfo{}

			// ウィンドウインデックス
			if index, err := fmt.Sscanf(parts[0], "%d", &windowInfo.Index); err == nil && index == 1 {
				// indexを正常に取得
			}

			// ウィンドウ名
			windowInfo.Name = parts[1]

			// アクティブウィンドウかどうか
			windowInfo.Active = parts[2] == "1"

			// ペイン数
			if panes, err := fmt.Sscanf(parts[3], "%d", &windowInfo.Panes); err == nil && panes == 1 {
				// panes数を正常に取得
			}

			windows = append(windows, windowInfo)
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxウィンドウ一覧取得完了",
			"session_name", sessionName,
			"count", len(windows))
	}

	return windows, nil
}

// WindowDetail はウィンドウの詳細情報を保持する構造体
type WindowDetail struct {
	*WindowInfo
	IssueNumber int    // Issue番号（パースできた場合）
	Phase       string // フェーズ（パースできた場合）
}

// ParseWindowName はウィンドウ名をパースしてIssue番号とフェーズを抽出する
func ParseWindowName(windowName string) (issueNumber int, phase string, ok bool) {
	// "37-plan", "40-implement", "42-review" の形式をパース
	parts := strings.Split(windowName, "-")
	if len(parts) != 2 {
		return 0, "", false
	}

	// Issue番号をパース
	num, err := fmt.Sscanf(parts[0], "%d", &issueNumber)
	if err != nil || num != 1 {
		return 0, "", false
	}

	// フェーズを検証
	phase = parts[1]
	validPhases := map[string]bool{
		"plan":      true,
		"implement": true,
		"review":    true,
	}

	if !validPhases[phase] {
		return 0, "", false
	}

	return issueNumber, phase, true
}

// GetWindowDetails はウィンドウ一覧を取得して詳細情報に変換する
func GetWindowDetails(sessionName string) ([]*WindowDetail, error) {
	return GetWindowDetailsWithExecutor(sessionName, &DefaultCommandExecutor{})
}

// GetWindowDetailsWithExecutor はExecutorを使用してウィンドウ一覧を取得して詳細情報に変換する
func GetWindowDetailsWithExecutor(sessionName string, executor CommandExecutor) ([]*WindowDetail, error) {
	windows, err := ListWindowsWithExecutor(sessionName, executor)
	if err != nil {
		return nil, err
	}

	details := make([]*WindowDetail, 0, len(windows))
	for _, window := range windows {
		detail := &WindowDetail{
			WindowInfo: window,
		}

		// ウィンドウ名をパース
		if issueNumber, phase, ok := ParseWindowName(window.Name); ok {
			detail.IssueNumber = issueNumber
			detail.Phase = phase
		}

		details = append(details, detail)
	}

	return details, nil
}

// SortWindowDetails はウィンドウ詳細情報を名前の昇順でソートする
func SortWindowDetails(details []*WindowDetail) {
	sort.Slice(details, func(i, j int) bool {
		return details[i].Name < details[j].Name
	})
}

// GetSortedWindowDetails はソート済みのウィンドウ詳細情報を取得する
func GetSortedWindowDetails(sessionName string) ([]*WindowDetail, error) {
	return GetSortedWindowDetailsWithExecutor(sessionName, &DefaultCommandExecutor{})
}

// GetSortedWindowDetailsWithExecutor はExecutorを使用してソート済みのウィンドウ詳細情報を取得する
func GetSortedWindowDetailsWithExecutor(sessionName string, executor CommandExecutor) ([]*WindowDetail, error) {
	details, err := GetWindowDetailsWithExecutor(sessionName, executor)
	if err != nil {
		return nil, err
	}

	SortWindowDetails(details)
	return details, nil
}

// CreateWindowForIssueWithExecutor はIssue番号とフェーズに基づいてウィンドウを作成する
func CreateWindowForIssueWithExecutor(sessionName string, issueNumber int, phase string, executor CommandExecutor) error {
	// フェーズの検証とウィンドウ名の生成
	windowName, err := GetWindowNameWithPhase(issueNumber, phase)
	if err != nil {
		return err
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue用フェーズウィンドウ作成開始",
			"operation", "create_window_for_issue_with_phase",
			"session_name", sessionName,
			"issue_number", issueNumber,
			"phase", phase,
			"window_name", windowName)
	}

	// ウィンドウが既に存在する場合はスキップ
	exists, err := WindowExistsWithExecutor(sessionName, windowName, executor)
	if err != nil {
		return fmt.Errorf("failed to check window existence: %w", err)
	}
	if exists {
		if logger := GetLogger(); logger != nil {
			logger.Info("フェーズウィンドウは既に存在します",
				"session_name", sessionName,
				"window_name", windowName)
		}
		return nil
	}

	return CreateWindowWithExecutor(sessionName, windowName, executor)
}

// SwitchToIssueWindowWithExecutor はIssue番号とフェーズに基づいてウィンドウに切り替える
func SwitchToIssueWindowWithExecutor(sessionName string, issueNumber int, phase string, executor CommandExecutor) error {
	// フェーズの検証とウィンドウ名の生成
	windowName, err := GetWindowNameWithPhase(issueNumber, phase)
	if err != nil {
		return err
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue用フェーズウィンドウへ切り替え",
			"operation", "switch_to_issue_window_with_phase",
			"session_name", sessionName,
			"issue_number", issueNumber,
			"phase", phase,
			"window_name", windowName)
	}

	return SwitchToWindowWithExecutor(sessionName, windowName, executor)
}

// ListWindowsByPattern は正規表現パターンに一致するウィンドウのリストを取得する
func ListWindowsByPattern(sessionName, pattern string) ([]*WindowInfo, error) {
	return ListWindowsByPatternWithExecutor(sessionName, pattern, &DefaultCommandExecutor{})
}

// ListWindowsByPatternWithExecutor はExecutorを使用して正規表現パターンに一致するウィンドウのリストを取得する
func ListWindowsByPatternWithExecutor(sessionName, pattern string, executor CommandExecutor) ([]*WindowInfo, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	// 正規表現のコンパイル
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// 全ウィンドウを取得
	windows, err := ListWindowsWithExecutor(sessionName, executor)
	if err != nil {
		return nil, err
	}

	// パターンに一致するウィンドウをフィルタリング
	var matched []*WindowInfo
	for _, window := range windows {
		if re.MatchString(window.Name) {
			matched = append(matched, window)
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("パターンマッチング完了",
			"session_name", sessionName,
			"pattern", pattern,
			"total_windows", len(windows),
			"matched_windows", len(matched))
	}

	return matched, nil
}

// ListWindowsForIssue は特定のIssue番号に関連するウィンドウのリストを取得する
func ListWindowsForIssue(sessionName string, issueNumber int) ([]*WindowInfo, error) {
	return ListWindowsForIssueWithExecutor(sessionName, issueNumber, &DefaultCommandExecutor{})
}

// ListWindowsForIssueWithExecutor はExecutorを使用して特定のIssue番号に関連するウィンドウのリストを取得する
func ListWindowsForIssueWithExecutor(sessionName string, issueNumber int, executor CommandExecutor) ([]*WindowInfo, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}
	if issueNumber <= 0 {
		return nil, fmt.Errorf("issue number must be positive")
	}

	// Issue番号に関連するウィンドウのパターン
	// 以下のパターンに一致するウィンドウを検索:
	// - "issue-144" (GetWindowNameで生成されるパターン)
	// - "144-plan", "144-implement", "144-review" (GetWindowNameWithPhaseで生成されるパターン)
	pattern := fmt.Sprintf("^(issue-%d|%d-.+)$", issueNumber, issueNumber)
	return ListWindowsByPatternWithExecutor(sessionName, pattern, executor)
}

// KillWindows は複数のウィンドウを一括削除する
func KillWindows(sessionName string, windowNames []string) error {
	return KillWindowsWithExecutor(sessionName, windowNames, &DefaultCommandExecutor{})
}

// KillWindowsWithExecutor はExecutorを使用して複数のウィンドウを一括削除する
func KillWindowsWithExecutor(sessionName string, windowNames []string, executor CommandExecutor) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("複数ウィンドウ削除開始",
			"operation", "kill_windows",
			"session_name", sessionName,
			"window_count", len(windowNames))
	}

	var errors []error
	for _, windowName := range windowNames {
		if windowName == "" {
			err := fmt.Errorf("window name cannot be empty")
			errors = append(errors, err)
			if logger := GetLogger(); logger != nil {
				logger.Error("空のウィンドウ名をスキップ", "error", err)
			}
			continue
		}

		if err := KillWindowWithExecutor(sessionName, windowName, executor); err != nil {
			errors = append(errors, fmt.Errorf("failed to kill window '%s': %w", windowName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to kill some windows: %v", errors)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("複数ウィンドウ削除完了",
			"session_name", sessionName,
			"window_count", len(windowNames))
	}

	return nil
}

// KillWindowsForIssue は特定のIssue番号に関連するウィンドウを一括削除する
func KillWindowsForIssue(sessionName string, issueNumber int) error {
	return KillWindowsForIssueWithExecutor(sessionName, issueNumber, &DefaultCommandExecutor{})
}

// KillWindowsForIssueWithExecutor はExecutorを使用して特定のIssue番号に関連するウィンドウを一括削除する
func KillWindowsForIssueWithExecutor(sessionName string, issueNumber int, executor CommandExecutor) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if issueNumber <= 0 {
		return fmt.Errorf("issue number must be positive")
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue関連ウィンドウ削除開始",
			"operation", "kill_windows_for_issue",
			"session_name", sessionName,
			"issue_number", issueNumber)
	}

	// Issue番号に関連するウィンドウを取得
	windows, err := ListWindowsForIssueWithExecutor(sessionName, issueNumber, executor)
	if err != nil {
		return fmt.Errorf("failed to list windows for issue %d: %w", issueNumber, err)
	}

	if len(windows) == 0 {
		if logger := GetLogger(); logger != nil {
			logger.Info("削除対象のウィンドウが見つかりません",
				"session_name", sessionName,
				"issue_number", issueNumber)
		}
		return nil
	}

	// ウィンドウ名のリストを作成
	windowNames := make([]string, len(windows))
	for i, window := range windows {
		windowNames[i] = window.Name
	}

	// ウィンドウを削除
	return KillWindowsWithExecutor(sessionName, windowNames, executor)
}

// Issue #147: pane管理メソッド

// SelectPaneByTitle はタイトルを指定してpaneを選択する
func SelectPaneByTitle(sessionName, windowName, paneTitle string) error {
	return SelectPaneByTitleWithExecutor(sessionName, windowName, paneTitle, &DefaultCommandExecutor{})
}

// SelectPaneByTitleWithExecutor はExecutorを使用してタイトルを指定してpaneを選択する
func SelectPaneByTitleWithExecutor(sessionName, windowName, paneTitle string, executor CommandExecutor) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}
	if paneTitle == "" {
		return fmt.Errorf("pane title cannot be empty")
	}

	target := fmt.Sprintf("%s:%s", sessionName, windowName)

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxペイン選択開始",
			"operation", "select_pane",
			"session_name", sessionName,
			"window_name", windowName,
			"pane_title", paneTitle,
			"target", target)
	}

	// pane一覧を取得
	output, err := executor.Execute("tmux", "list-panes", "-t", target, "-F", "#{pane_index}:#{pane_title}")
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxペイン一覧取得失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"error", err)
		}
		return fmt.Errorf("failed to list panes in window '%s': %w", windowName, err)
	}

	// 指定されたタイトルのpaneを検索
	panes := strings.Split(strings.TrimSpace(output), "\n")
	for _, pane := range panes {
		if pane == "" {
			continue
		}
		parts := strings.Split(pane, ":")
		if len(parts) >= 2 {
			paneIndex := parts[0]
			title := strings.Join(parts[1:], ":")
			if title == paneTitle {
				// paneを選択
				paneTarget := fmt.Sprintf("%s.%s", target, paneIndex)
				_, err := executor.Execute("tmux", "select-pane", "-t", paneTarget)
				if err != nil {
					if logger := GetLogger(); logger != nil {
						logger.Error("tmuxペイン選択失敗",
							"session_name", sessionName,
							"window_name", windowName,
							"pane_title", paneTitle,
							"pane_index", paneIndex,
							"error", err)
					}
					return fmt.Errorf("failed to select pane '%s' in window '%s': %w", paneTitle, windowName, err)
				}

				if logger := GetLogger(); logger != nil {
					logger.Info("tmuxペイン選択完了",
						"session_name", sessionName,
						"window_name", windowName,
						"pane_title", paneTitle,
						"pane_index", paneIndex)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("pane with title '%s' not found in window '%s'", paneTitle, windowName)
}
