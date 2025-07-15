package tmux

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

// SessionInfo はtmuxセッションの情報を保持する構造体
type SessionInfo struct {
	Name     string
	Windows  int
	Created  string
	Attached bool
}

// ListSessions は存在するtmuxセッションの一覧を取得する
func ListSessions(prefix string) ([]*SessionInfo, error) {
	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxセッション一覧取得",
			"operation", "list_sessions",
			"prefix", prefix,
			"command", "tmux list-sessions")
	}

	// tmux list-sessions -F "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}"
	cmd := execCommand("tmux", "list-sessions", "-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}")
	output, err := cmd.Output()

	if err != nil {
		// セッションが存在しない場合もエラーになるが、それは正常な状態
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				if logger := GetLogger(); logger != nil {
					logger.Debug("tmuxセッションが存在しません")
				}
				return []*SessionInfo{}, nil
			}
		}
		if logger := GetLogger(); logger != nil {
			logger.Error("tmuxセッション一覧取得失敗", "error", err)
		}
		return nil, fmt.Errorf("tmuxセッション一覧の取得に失敗: %w", err)
	}

	sessions := []*SessionInfo{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			sessionName := parts[0]

			// prefixが指定されている場合は、そのprefixで始まるセッションのみ抽出
			if prefix != "" && !strings.HasPrefix(sessionName, prefix) {
				continue
			}

			windows := 0
			if n, err := fmt.Sscanf(parts[1], "%d", &windows); err == nil && n == 1 {
				// windows数を正常に取得
			}

			attached := parts[3] == "1"

			sessions = append(sessions, &SessionInfo{
				Name:     sessionName,
				Windows:  windows,
				Created:  parts[2],
				Attached: attached,
			})
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxセッション一覧取得完了",
			"count", len(sessions))
	}

	return sessions, nil
}
