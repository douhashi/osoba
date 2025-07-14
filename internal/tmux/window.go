package tmux

import (
	"fmt"
	"os/exec"
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
func CreateWindow(sessionName, windowName string) error {
	return CreateWindowWithExecutor(sessionName, windowName, &DefaultCommandExecutor{})
}

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
func SwitchToWindow(sessionName, windowName string) error {
	return SwitchToWindowWithExecutor(sessionName, windowName, &DefaultCommandExecutor{})
}

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
func WindowExists(sessionName, windowName string) (bool, error) {
	return WindowExistsWithExecutor(sessionName, windowName, &DefaultCommandExecutor{})
}

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

// CreateWindowForIssue はIssue番号に基づいてウィンドウを作成する
func CreateWindowForIssue(sessionName string, issueNumber int) error {
	windowName := GetWindowName(issueNumber)

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue用ウィンドウ作成開始",
			"operation", "create_window_for_issue",
			"session_name", sessionName,
			"issue_number", issueNumber,
			"window_name", windowName)
	}

	// ウィンドウが既に存在する場合はスキップ
	exists, err := WindowExists(sessionName, windowName)
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

	return CreateWindow(sessionName, windowName)
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
