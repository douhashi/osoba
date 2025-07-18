package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// CreateWindow 新しいtmuxウィンドウを作成
func (m *DefaultManager) CreateWindow(sessionName, windowName string) error {
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

	_, err := m.executor.Execute("tmux", "new-window", "-t", sessionName, "-n", windowName)
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

// SwitchToWindow 指定されたウィンドウに切り替え
func (m *DefaultManager) SwitchToWindow(sessionName, windowName string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}

	target := fmt.Sprintf("%s:%s", sessionName, windowName)

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ切り替え開始",
			"operation", "switch_window",
			"session_name", sessionName,
			"window_name", windowName,
			"target", target,
			"command", "tmux select-window",
			"args", []string{"-t", target})
	}

	_, err := m.executor.Execute("tmux", "select-window", "-t", target)
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

// WindowExists 指定されたウィンドウが存在するか確認
func (m *DefaultManager) WindowExists(sessionName, windowName string) (bool, error) {
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

	output, err := m.executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}")
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

// KillWindow 指定されたウィンドウを削除
func (m *DefaultManager) KillWindow(sessionName, windowName string) error {
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

	_, err := m.executor.Execute("tmux", "kill-window", "-t", target)
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

// CreateOrReplaceWindow ウィンドウが存在する場合は削除してから新規作成
func (m *DefaultManager) CreateOrReplaceWindow(sessionName, windowName string) error {
	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ作成/置換開始",
			"operation", "create_or_replace_window",
			"session_name", sessionName,
			"window_name", windowName)
	}

	// ウィンドウの存在確認
	exists, err := m.WindowExists(sessionName, windowName)
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

		if err := m.KillWindow(sessionName, windowName); err != nil {
			return fmt.Errorf("failed to kill existing window: %w", err)
		}
	}

	// 新しいウィンドウを作成
	if err := m.CreateWindow(sessionName, windowName); err != nil {
		return fmt.Errorf("failed to create new window: %w", err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxウィンドウ作成/置換完了",
			"session_name", sessionName,
			"window_name", windowName)
	}

	return nil
}

// ListWindows 指定されたセッションのウィンドウ一覧を取得
func (m *DefaultManager) ListWindows(sessionName string) ([]string, error) {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxウィンドウ一覧取得",
			"operation", "list_windows",
			"session_name", sessionName,
			"command", "tmux list-windows",
			"args", []string{"-t", sessionName, "-F", "#{window_name}"})
	}

	output, err := m.executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}")
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウ一覧取得失敗",
				"session_name", sessionName,
				"error", err)
		}
		return nil, fmt.Errorf("failed to list windows in session '%s': %w", sessionName, err)
	}

	windows := strings.Split(strings.TrimSpace(output), "\n")
	var result []string
	for _, window := range windows {
		if window != "" {
			result = append(result, window)
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("ウィンドウ一覧を取得しました",
			"session_name", sessionName,
			"count", len(result),
			"windows", result)
	}

	return result, nil
}

// SendKeys 指定されたウィンドウにキーを送信
func (m *DefaultManager) SendKeys(sessionName, windowName, keys string) error {
	if sessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return fmt.Errorf("window name cannot be empty")
	}
	if keys == "" {
		return fmt.Errorf("keys cannot be empty")
	}

	target := fmt.Sprintf("%s:%s", sessionName, windowName)

	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxウィンドウにキー送信",
			"operation", "send_keys",
			"session_name", sessionName,
			"window_name", windowName,
			"keys", keys,
			"command", "tmux send-keys",
			"args", []string{"-t", target, keys})
	}

	_, err := m.executor.Execute("tmux", "send-keys", "-t", target, keys)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxウィンドウへのキー送信失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"keys", keys,
				"error", err)
		}
		return fmt.Errorf("failed to send keys to window '%s' in session '%s': %w", windowName, sessionName, err)
	}

	return nil
}

// ClearWindow ウィンドウの内容をクリア
func (m *DefaultManager) ClearWindow(sessionName, windowName string) error {
	// Ctrl+L を送信してクリア
	return m.SendKeys(sessionName, windowName, "C-l")
}

// RunInWindow ウィンドウ内でコマンドを実行
func (m *DefaultManager) RunInWindow(sessionName, windowName, command string) error {
	// コマンドを送信してEnterキーを押す
	if err := m.SendKeys(sessionName, windowName, command); err != nil {
		return err
	}
	return m.SendKeys(sessionName, windowName, "Enter")
}

// GetIssueWindow Issue番号に対応するウィンドウ名を取得
func (m *DefaultManager) GetIssueWindow(issueNumber int) string {
	return fmt.Sprintf("issue-%d", issueNumber)
}

// MatchIssueWindow ウィンドウ名がIssueパターンにマッチするか確認
func (m *DefaultManager) MatchIssueWindow(windowName string) bool {
	return strings.HasPrefix(windowName, "issue-") || IsIssueWindow(windowName)
}

// FindIssueWindow ウィンドウ名からIssue番号を抽出
func (m *DefaultManager) FindIssueWindow(windowName string) (int, bool) {
	// "issue-123" 形式のチェック
	if strings.HasPrefix(windowName, "issue-") {
		numStr := strings.TrimPrefix(windowName, "issue-")
		if num, err := strconv.Atoi(numStr); err == nil {
			return num, true
		}
	}

	// "123-plan", "123-implement", "123-review" 形式のチェック
	parts := strings.Split(windowName, "-")
	if len(parts) == 2 {
		if num, err := strconv.Atoi(parts[0]); err == nil {
			phase := parts[1]
			if phase == "plan" || phase == "implement" || phase == "review" {
				return num, true
			}
		}
	}

	return 0, false
}

// IsIssueWindow はウィンドウ名がフェーズ付きのIssueウィンドウかチェック
func IsIssueWindow(windowName string) bool {
	parts := strings.Split(windowName, "-")
	if len(parts) != 2 {
		return false
	}

	// 最初の部分が数字かチェック
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return false
	}

	// 2番目の部分が有効なフェーズかチェック
	phase := parts[1]
	return phase == "plan" || phase == "implement" || phase == "review"
}

// CreateWindowForIssueWithNewWindowDetection Issue番号に基づいてウィンドウを作成し、新規作成かどうかを返す
func (m *DefaultManager) CreateWindowForIssueWithNewWindowDetection(sessionName string, issueNumber int) (string, bool, error) {
	return CreateWindowForIssueWithNewWindowDetection(sessionName, issueNumber, m.executor)
}
