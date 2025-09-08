package cleanup

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/tmux"
)

// Manager はクリーンアップ処理のインターフェース
type Manager interface {
	CleanupIssueResources(ctx context.Context, issueNumber int) error
}

// DefaultManager は標準のクリーンアップマネージャー
type DefaultManager struct {
	sessionName string
	logger      logger.Logger
	executor    tmux.CommandExecutor // テスト可能にするため
}

// NewManager は新しいクリーンアップマネージャーを作成する
// sessionNameが空の場合は後方互換性のため従来の動作をする
func NewManager(sessionName string, logger logger.Logger) Manager {
	return &DefaultManager{
		sessionName: sessionName,
		logger:      logger,
		executor:    &tmux.DefaultCommandExecutor{},
	}
}

// CleanupIssueResources はIssueに関連するリソースをクリーンアップする
func (m *DefaultManager) CleanupIssueResources(ctx context.Context, issueNumber int) error {
	// tmuxウィンドウをクローズ
	if err := m.closeTmuxWindowsForIssue(ctx, issueNumber); err != nil {
		if m.logger != nil {
			m.logger.Warn("Failed to close tmux windows",
				"issue_number", issueNumber,
				"error", err,
			)
		}
		// エラーは無視して続行
	}

	// worktreeを削除
	if err := m.removeWorktree(ctx, issueNumber); err != nil {
		if m.logger != nil {
			m.logger.Warn("Failed to remove worktree",
				"issue_number", issueNumber,
				"error", err,
			)
		}
		// エラーは無視して続行
	}

	return nil
}

// closeTmuxWindowsForIssue はIssueに関連するすべてのtmuxウィンドウを閉じる
func (m *DefaultManager) closeTmuxWindowsForIssue(ctx context.Context, issueNumber int) error {
	// セッション名が指定されていない場合は警告を出して従来の動作
	if m.sessionName == "" {
		if m.logger != nil {
			m.logger.Warn("Session name not specified, using legacy window cleanup",
				"issue_number", issueNumber,
			)
		}
		// 従来の動作（セッション名なしでウィンドウ名のみ指定）
		windowName := tmux.GetWindowNameForIssue(issueNumber)
		return m.closeTmuxWindowLegacy(ctx, windowName)
	}

	if m.logger != nil {
		m.logger.Debug("Cleaning up tmux windows for issue",
			"session", m.sessionName,
			"issue_number", issueNumber,
		)
	}

	// Issue番号に関連するすべてのウィンドウを取得
	windows, err := tmux.ListWindowsForIssueWithExecutor(m.sessionName, issueNumber, m.executor)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("Failed to list windows for issue",
				"session", m.sessionName,
				"issue_number", issueNumber,
				"error", err,
			)
		}
		// リスト取得に失敗してもエラーを返さない（クリーンアップを継続）
		return nil
	}

	if len(windows) == 0 {
		if m.logger != nil {
			m.logger.Info("No tmux windows found for issue",
				"session", m.sessionName,
				"issue_number", issueNumber,
			)
		}
		return nil
	}

	// 各ウィンドウを削除
	var windowNames []string
	for _, window := range windows {
		windowNames = append(windowNames, window.Name)
	}

	if m.logger != nil {
		m.logger.Info("Closing tmux windows for issue",
			"session", m.sessionName,
			"issue_number", issueNumber,
			"windows", windowNames,
		)
	}

	// ウィンドウを一括削除
	err = tmux.KillWindowsWithExecutor(m.sessionName, windowNames, m.executor)
	if err != nil {
		if m.logger != nil {
			m.logger.Warn("Some windows could not be closed",
				"session", m.sessionName,
				"issue_number", issueNumber,
				"windows", windowNames,
				"error", err,
			)
		}
		// 一部のウィンドウが削除できなくてもエラーを返さない
		return nil
	}

	if m.logger != nil {
		m.logger.Info("Successfully closed all tmux windows for issue",
			"session", m.sessionName,
			"issue_number", issueNumber,
			"window_count", len(windows),
		)
	}

	return nil
}

// closeTmuxWindowLegacy は従来の方法でtmuxウィンドウを閉じる（後方互換性のため）
func (m *DefaultManager) closeTmuxWindowLegacy(ctx context.Context, windowName string) error {
	// tmux kill-window -t <window-name>
	cmd := exec.CommandContext(ctx, "tmux", "kill-window", "-t", windowName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ウィンドウが存在しない場合もエラーになるが、それは問題ない
		return fmt.Errorf("failed to kill tmux window: %s", string(output))
	}

	if m.logger != nil {
		m.logger.Debug("Closed tmux window (legacy)",
			"window", windowName,
		)
	}

	return nil
}

// removeWorktree はgit worktreeを削除する
func (m *DefaultManager) removeWorktree(ctx context.Context, issueNumber int) error {
	// worktreeのパス（例: .git/osoba/worktrees/issue-123）
	worktreePath := fmt.Sprintf(".git/osoba/worktrees/issue-%d", issueNumber)

	// git worktree remove <path> --force
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// worktreeが存在しない場合もエラーになるが、それは問題ない
		return fmt.Errorf("failed to remove worktree: %s", string(output))
	}

	if m.logger != nil {
		m.logger.Debug("Removed worktree",
			"path", worktreePath,
		)
	}

	return nil
}
