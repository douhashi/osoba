package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// ListSessionsAsSessionInfo は存在するtmuxセッションの詳細情報を取得する
// これは既存のコードとの互換性のために提供される関数です
func ListSessionsAsSessionInfo(prefix string) ([]*SessionInfo, error) {
	manager := globalManager
	if manager == nil {
		return nil, fmt.Errorf("tmux manager not initialized")
	}

	// CommandExecutorを使用した実装
	executor, ok := manager.(*DefaultManager)
	if !ok {
		// モックマネージャーの場合は空の結果を返す
		return []*SessionInfo{}, nil
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("tmuxセッション一覧取得",
			"operation", "list_sessions",
			"prefix", prefix,
			"command", "tmux list-sessions")
	}

	// tmux list-sessions -F "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}"
	output, err := executor.executor.Execute("tmux", "list-sessions", "-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}")

	if err != nil {
		// セッションが存在しない場合もエラーになるが、それは正常な状態
		if exitCode, isExit := IsExitError(err); isExit && exitCode == 1 {
			if logger := GetLogger(); logger != nil {
				logger.Debug("tmuxセッションが存在しません")
			}
			return []*SessionInfo{}, nil
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
			if n, err := strconv.Atoi(parts[1]); err == nil {
				windows = n
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
