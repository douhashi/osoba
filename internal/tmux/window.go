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

	_, err := executor.Execute("tmux", "new-window", "-t", sessionName, "-n", windowName)
	if err != nil {
		return fmt.Errorf("failed to create window '%s' in session '%s': %w", windowName, sessionName, err)
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
	_, err := executor.Execute("tmux", "select-window", "-t", target)
	if err != nil {
		return fmt.Errorf("failed to switch to window '%s' in session '%s': %w", windowName, sessionName, err)
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

	output, err := executor.Execute("tmux", "list-windows", "-t", sessionName, "-F", "#{window_name}")
	if err != nil {
		return false, fmt.Errorf("failed to list windows in session '%s': %w", sessionName, err)
	}

	windows := strings.Split(strings.TrimSpace(output), "\n")
	for _, window := range windows {
		if window == windowName {
			return true, nil
		}
	}

	return false, nil
}

// GetWindowName はIssue番号からウィンドウ名を生成する
func GetWindowName(issueNumber int) string {
	return fmt.Sprintf("issue-%d", issueNumber)
}

// CreateWindowForIssue はIssue番号に基づいてウィンドウを作成する
func CreateWindowForIssue(sessionName string, issueNumber int) error {
	windowName := GetWindowName(issueNumber)

	// ウィンドウが既に存在する場合はスキップ
	exists, err := WindowExists(sessionName, windowName)
	if err != nil {
		return fmt.Errorf("failed to check window existence: %w", err)
	}
	if exists {
		return nil
	}

	return CreateWindow(sessionName, windowName)
}

// SwitchToIssueWindow はIssue番号に基づいてウィンドウに切り替える
func SwitchToIssueWindow(sessionName string, issueNumber int) error {
	windowName := GetWindowName(issueNumber)
	return SwitchToWindow(sessionName, windowName)
}
