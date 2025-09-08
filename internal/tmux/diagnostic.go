package tmux

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SessionDiagnostics はセッション診断情報を保持する構造体
type SessionDiagnostics struct {
	Name      string            // セッション名
	Windows   int               // ウィンドウ数
	Attached  bool              // アタッチ状態
	Created   string            // 作成日時
	Errors    []string          // エラー情報
	Metadata  map[string]string // 追加のメタデータ
	Timestamp time.Time         // 診断実行時刻
}

// WindowDiagnostics はウィンドウ診断情報を保持する構造体
type WindowDiagnostics struct {
	Name        string            // ウィンドウ名
	SessionName string            // 所属セッション名
	Index       int               // ウィンドウインデックス
	Exists      bool              // 存在状態
	Active      bool              // アクティブ状態
	Panes       int               // ペイン数
	IssueNumber int               // Issue番号（パース可能な場合）
	Phase       string            // フェーズ（パース可能な場合）
	Errors      []string          // エラー情報
	Metadata    map[string]string // 追加のメタデータ
	Timestamp   time.Time         // 診断実行時刻
}

// DiagnosticManager は診断機能のインターフェース
type DiagnosticManager interface {
	// DiagnoseSession 指定されたセッションの診断情報を取得
	DiagnoseSession(sessionName string) (*SessionDiagnostics, error)

	// DiagnoseWindow 指定されたウィンドウの診断情報を取得
	DiagnoseWindow(sessionName, windowName string) (*WindowDiagnostics, error)

	// ListSessionDiagnostics 指定されたプレフィックスで始まるセッションの診断情報一覧を取得
	ListSessionDiagnostics(prefix string) ([]*SessionDiagnostics, error)

	// ListWindowDiagnostics 指定されたセッションのウィンドウ診断情報一覧を取得
	ListWindowDiagnostics(sessionName string) ([]*WindowDiagnostics, error)
}

// DiagnoseSession 指定されたセッションの診断情報を取得
func (m *DefaultManager) DiagnoseSession(sessionName string) (*SessionDiagnostics, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("セッション診断開始",
			"operation", "diagnose_session",
			"session_name", sessionName)
	}

	diag := &SessionDiagnostics{
		Name:      sessionName,
		Errors:    make([]string, 0),
		Metadata:  make(map[string]string),
		Timestamp: time.Now(),
	}

	// セッション情報を取得
	output, err := m.executor.Execute("tmux", "list-sessions", "-t", sessionName,
		"-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}")

	if err != nil {
		// セッションが存在しない場合の処理
		if exitCode, isExit := IsExitError(err); isExit && exitCode == 1 {
			diag.Errors = append(diag.Errors, "session does not exist")
			diag.Metadata["exists"] = "false"
			
			if logger := GetLogger(); logger != nil {
				logger.Debug("セッションが存在しません",
					"session_name", sessionName)
			}
			return diag, nil
		}
		
		// その他のエラー
		errMsg := fmt.Sprintf("failed to get session info: %v", err)
		diag.Errors = append(diag.Errors, errMsg)
		
		if logger := GetLogger(); logger != nil {
			logger.Error("セッション情報取得エラー",
				"session_name", sessionName,
				"error", err)
		}
		return diag, nil
	}

	// 出力をパース
	parts := strings.Split(strings.TrimSpace(output), ":")
	if len(parts) >= 4 {
		// ウィンドウ数
		if windows, err := strconv.Atoi(parts[1]); err == nil {
			diag.Windows = windows
		} else {
			diag.Errors = append(diag.Errors, "failed to parse windows count")
		}

		// 作成日時
		diag.Created = parts[2]

		// アタッチ状態
		diag.Attached = parts[3] == "1"
		diag.Metadata["exists"] = "true"
	} else {
		diag.Errors = append(diag.Errors, "invalid session info format")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("セッション診断完了",
			"session_name", sessionName,
			"windows", diag.Windows,
			"attached", diag.Attached,
			"errors", len(diag.Errors))
	}

	return diag, nil
}

// DiagnoseWindow 指定されたウィンドウの診断情報を取得
func (m *DefaultManager) DiagnoseWindow(sessionName, windowName string) (*WindowDiagnostics, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}
	if windowName == "" {
		return nil, fmt.Errorf("window name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("ウィンドウ診断開始",
			"operation", "diagnose_window",
			"session_name", sessionName,
			"window_name", windowName)
	}

	diag := &WindowDiagnostics{
		Name:        windowName,
		SessionName: sessionName,
		Errors:      make([]string, 0),
		Metadata:    make(map[string]string),
		Timestamp:   time.Now(),
	}

	// ウィンドウ名をパースしてIssue情報を抽出
	if issueNumber, phase, ok := ParseWindowName(windowName); ok {
		diag.IssueNumber = issueNumber
		diag.Phase = phase
		diag.Metadata["issue_window"] = "true"
	}

	// ウィンドウ情報を取得
	output, err := m.executor.Execute("tmux", "list-windows", "-t", sessionName,
		"-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}")

	if err != nil {
		// セッションが存在しない場合
		if exitCode, isExit := IsExitError(err); isExit && exitCode == 1 {
			diag.Errors = append(diag.Errors, "session does not exist")
			diag.Metadata["session_exists"] = "false"
		} else {
			errMsg := fmt.Sprintf("failed to get window info: %v", err)
			diag.Errors = append(diag.Errors, errMsg)
		}
		
		if logger := GetLogger(); logger != nil {
			logger.Error("ウィンドウ情報取得エラー",
				"session_name", sessionName,
				"window_name", windowName,
				"error", err)
		}
		return diag, nil
	}

	// 指定されたウィンドウを検索
	lines := strings.Split(strings.TrimSpace(output), "\n")
	found := false
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ":")
		if len(parts) >= 4 && parts[1] == windowName {
			found = true
			diag.Exists = true
			diag.Metadata["exists"] = "true"

			// インデックス
			if index, err := strconv.Atoi(parts[0]); err == nil {
				diag.Index = index
			}

			// アクティブ状態
			diag.Active = parts[2] == "1"

			// ペイン数
			if panes, err := strconv.Atoi(parts[3]); err == nil {
				diag.Panes = panes
			}
			break
		}
	}

	if !found {
		diag.Exists = false
		diag.Metadata["exists"] = "false"
		diag.Errors = append(diag.Errors, "window does not exist in session")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("ウィンドウ診断完了",
			"session_name", sessionName,
			"window_name", windowName,
			"exists", diag.Exists,
			"active", diag.Active,
			"panes", diag.Panes,
			"errors", len(diag.Errors))
	}

	return diag, nil
}

// ListSessionDiagnostics 指定されたプレフィックスで始まるセッションの診断情報一覧を取得
func (m *DefaultManager) ListSessionDiagnostics(prefix string) ([]*SessionDiagnostics, error) {
	if logger := GetLogger(); logger != nil {
		logger.Debug("セッション診断一覧取得開始",
			"operation", "list_session_diagnostics",
			"prefix", prefix)
	}

	// セッション一覧を取得
	output, err := m.executor.Execute("tmux", "list-sessions",
		"-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}")

	if err != nil {
		// セッションが存在しない場合
		if exitCode, isExit := IsExitError(err); isExit && exitCode == 1 {
			if logger := GetLogger(); logger != nil {
				logger.Debug("tmuxセッションが存在しません")
			}
			return []*SessionDiagnostics{}, nil
		}
		
		if logger := GetLogger(); logger != nil {
			logger.Error("セッション一覧取得エラー", "error", err)
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	diagnostics := []*SessionDiagnostics{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			sessionName := parts[0]

			// プレフィックスでフィルタリング
			if prefix != "" && !strings.HasPrefix(sessionName, prefix) {
				continue
			}

			diag := &SessionDiagnostics{
				Name:      sessionName,
				Created:   parts[2],
				Attached:  parts[3] == "1",
				Errors:    make([]string, 0),
				Metadata:  make(map[string]string),
				Timestamp: time.Now(),
			}

			// ウィンドウ数
			if windows, err := strconv.Atoi(parts[1]); err == nil {
				diag.Windows = windows
			} else {
				diag.Errors = append(diag.Errors, "failed to parse windows count")
			}

			diag.Metadata["exists"] = "true"
			diagnostics = append(diagnostics, diag)
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("セッション診断一覧取得完了",
			"prefix", prefix,
			"count", len(diagnostics))
	}

	return diagnostics, nil
}

// ListWindowDiagnostics 指定されたセッションのウィンドウ診断情報一覧を取得
func (m *DefaultManager) ListWindowDiagnostics(sessionName string) ([]*WindowDiagnostics, error) {
	if sessionName == "" {
		return nil, fmt.Errorf("session name cannot be empty")
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("ウィンドウ診断一覧取得開始",
			"operation", "list_window_diagnostics",
			"session_name", sessionName)
	}

	// ウィンドウ一覧を取得
	output, err := m.executor.Execute("tmux", "list-windows", "-t", sessionName,
		"-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}")

	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("ウィンドウ一覧取得エラー",
				"session_name", sessionName,
				"error", err)
		}
		return nil, fmt.Errorf("failed to list windows in session '%s': %w", sessionName, err)
	}

	diagnostics := []*WindowDiagnostics{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			windowName := parts[1]
			
			diag := &WindowDiagnostics{
				Name:        windowName,
				SessionName: sessionName,
				Exists:      true,
				Active:      parts[2] == "1",
				Errors:      make([]string, 0),
				Metadata:    make(map[string]string),
				Timestamp:   time.Now(),
			}

			// インデックス
			if index, err := strconv.Atoi(parts[0]); err == nil {
				diag.Index = index
			}

			// ペイン数
			if panes, err := strconv.Atoi(parts[3]); err == nil {
				diag.Panes = panes
			}

			// ウィンドウ名をパースしてIssue情報を抽出
			if issueNumber, phase, ok := ParseWindowName(windowName); ok {
				diag.IssueNumber = issueNumber
				diag.Phase = phase
				diag.Metadata["issue_window"] = "true"
			}

			diag.Metadata["exists"] = "true"
			diagnostics = append(diagnostics, diag)
		}
	}

	if logger := GetLogger(); logger != nil {
		logger.Debug("ウィンドウ診断一覧取得完了",
			"session_name", sessionName,
			"count", len(diagnostics))
	}

	return diagnostics, nil
}