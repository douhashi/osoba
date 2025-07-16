package tmux

import (
	"fmt"
	"strings"
)

// CheckTmuxInstalled tmuxがインストールされているか確認
func (m *DefaultManager) CheckTmuxInstalled() error {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxインストール確認", "command", "which tmux")
	}

	_, err := m.executor.Execute("which", "tmux")
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxがインストールされていません", "error", err)
		}
		return ErrTmuxNotInstalled
	}
	return nil
}

// SessionExists 指定された名前のtmuxセッションが存在するか確認
func (m *DefaultManager) SessionExists(sessionName string) (bool, error) {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxセッション確認",
			"operation", "session_exists",
			"session_name", sessionName,
			"command", "tmux has-session",
			"args", []string{"-t", sessionName})
	}

	_, err := m.executor.Execute("tmux", "has-session", "-t", sessionName)

	if err != nil {
		// tmuxのhas-sessionは、セッションが存在しない場合にエラーを返す
		if exitCode, isExit := IsExitError(err); isExit {
			// 終了コード1はセッションが存在しないことを示す
			if exitCode == 1 {
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
func (m *DefaultManager) CreateSession(sessionName string) error {
	if logger := GetLogger(); logger != nil {
		logger.Info("tmuxセッション作成開始",
			"operation", "create_session",
			"session_name", sessionName,
			"command", "tmux new-session",
			"args", []string{"-d", "-s", sessionName})
	}

	_, err := m.executor.Execute("tmux", "new-session", "-d", "-s", sessionName)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxセッションの作成に失敗",
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

// EnsureSession tmuxセッションが存在しない場合は作成
func (m *DefaultManager) EnsureSession(sessionName string) error {
	exists, err := m.SessionExists(sessionName)
	if err != nil {
		return err
	}

	if !exists {
		if logger := GetLogger(); logger != nil {
			logger.Info("セッションが存在しないため作成します",
				"session_name", sessionName)
		}
		return m.CreateSession(sessionName)
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("セッションは既に存在します",
			"session_name", sessionName)
	}
	return nil
}

// ListSessions 指定されたプレフィックスで始まるセッション一覧を取得
func (m *DefaultManager) ListSessions(prefix string) ([]string, error) {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxセッション一覧取得",
			"operation", "list_sessions",
			"prefix", prefix,
			"command", "tmux list-sessions",
			"args", []string{"-F", "#{session_name}"})
	}

	output, err := m.executor.Execute("tmux", "list-sessions", "-F", "#{session_name}")
	if err != nil {
		// tmuxが起動していない場合や、セッションが1つもない場合はエラーになる
		if exitCode, isExit := IsExitError(err); isExit && exitCode == 1 {
			if logger := GetLogger(); logger != nil {
				logger.Debug("tmuxセッションが存在しません")
			}
			return []string{}, nil
		}
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxセッション一覧の取得に失敗",
				"error", err)
		}
		return nil, fmt.Errorf("tmuxセッション一覧の取得に失敗: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var sessions []string
	for _, line := range lines {
		if line != "" && strings.HasPrefix(line, prefix) {
			sessions = append(sessions, line)
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("セッション一覧を取得しました",
			"prefix", prefix,
			"count", len(sessions),
			"sessions", sessions)
	}

	return sessions, nil
}
