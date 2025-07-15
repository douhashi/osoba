package tmux

import (
	"errors"
	"fmt"
	"os/exec"
)

var (
	// ErrTmuxNotInstalled tmuxがインストールされていない場合のエラー
	ErrTmuxNotInstalled = errors.New("tmuxがインストールされていません。\n" +
		"インストール方法:\n" +
		"  Ubuntu/Debian: sudo apt-get install tmux\n" +
		"  macOS: brew install tmux\n" +
		"  その他: https://github.com/tmux/tmux/wiki/Installing")
)

// execCommand はテスト時にモック可能なコマンド実行関数
var execCommand = exec.Command

// CheckTmuxInstalled tmuxがインストールされているか確認
func CheckTmuxInstalled() error {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxインストール確認", "command", "which tmux")
	}

	cmd := execCommand("which", "tmux")
	if err := cmd.Run(); err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxがインストールされていません", "error", err)
		}
		return ErrTmuxNotInstalled
	}
	return nil
}

// SessionExists 指定された名前のtmuxセッションが存在するか確認
func SessionExists(sessionName string) (bool, error) {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxセッション確認",
			"operation", "session_exists",
			"session_name", sessionName,
			"command", "tmux has-session",
			"args", []string{"-t", sessionName})
	}

	cmd := execCommand("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()

	if err != nil {
		// tmuxのhas-sessionは、セッションが存在しない場合にエラーを返す
		if exitError, ok := err.(*exec.ExitError); ok {
			// 終了コード1はセッションが存在しないことを示す
			if exitError.ExitCode() == 1 {
				if logger := GetLogger(); logger != nil {
					logger.Debug("セッションが存在しません",
						"session_name", sessionName,
						"exit_code", 1)
				}
				return false, nil
			}
		}
		// その他のエラー
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxセッションの確認に失敗",
				"session_name", sessionName,
				"error", err)
		}
		return false, fmt.Errorf("tmuxセッションの確認に失敗: %w", err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("セッションが存在します",
			"session_name", sessionName)
	}
	return true, nil
}

// CreateSession 新しいtmuxセッションを作成
func CreateSession(sessionName string) error {
	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxセッション作成開始",
			"operation", "create_session",
			"session_name", sessionName,
			"command", "tmux new-session",
			"args", []string{"-d", "-s", sessionName})
	}

	// デタッチモードで新しいセッションを作成
	cmd := execCommand("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxセッション作成失敗",
				"session_name", sessionName,
				"error", err)
		}
		return fmt.Errorf("tmuxセッションの作成に失敗: %w", err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxセッション作成完了",
			"session_name", sessionName)
	}
	return nil
}

// EnsureSession tmuxセッションが存在することを保証（存在しない場合は作成）
func EnsureSession(sessionName string) error {
	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxセッション確保開始",
			"operation", "ensure_session",
			"session_name", sessionName)
	}

	// セッションの存在確認
	exists, err := SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("failed to check session existence: %w", err)
	}

	if exists {
		if logger := GetLogger(); logger != nil {
			logger.Info("セッションは既に存在します",
				"session_name", sessionName)
		}
		return nil
	}

	// セッションが存在しない場合は作成
	if logger := GetLogger(); logger != nil {
		logger.Info("セッションが存在しないため、新規作成します",
			"session_name", sessionName)
	}

	if err := CreateSession(sessionName); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxセッション確保完了",
			"session_name", sessionName)
	}

	return nil
}
