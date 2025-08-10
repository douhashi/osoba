package cleanup

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/douhashi/osoba/internal/logger"
)

// Manager はクリーンアップ処理のインターフェース
type Manager interface {
	CleanupIssueResources(ctx context.Context, issueNumber int) error
}

// DefaultManager は標準のクリーンアップマネージャー
type DefaultManager struct {
	logger logger.Logger
}

// NewManager は新しいクリーンアップマネージャーを作成する
func NewManager(logger logger.Logger) Manager {
	return &DefaultManager{
		logger: logger,
	}
}

// CleanupIssueResources はIssueに関連するリソースをクリーンアップする
func (m *DefaultManager) CleanupIssueResources(ctx context.Context, issueNumber int) error {
	// tmuxウィンドウ名
	windowName := strconv.Itoa(issueNumber)

	// tmuxウィンドウをクローズ
	if err := m.closeTmuxWindow(ctx, windowName); err != nil {
		if m.logger != nil {
			m.logger.Warn("Failed to close tmux window",
				"window", windowName,
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

// closeTmuxWindow はtmuxウィンドウを閉じる
func (m *DefaultManager) closeTmuxWindow(ctx context.Context, windowName string) error {
	// tmux kill-window -t <window-name>
	cmd := exec.CommandContext(ctx, "tmux", "kill-window", "-t", windowName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ウィンドウが存在しない場合もエラーになるが、それは問題ない
		return fmt.Errorf("failed to kill tmux window: %s", string(output))
	}

	if m.logger != nil {
		m.logger.Debug("Closed tmux window",
			"window", windowName,
		)
	}

	return nil
}

// removeWorktree はgit worktreeを削除する
func (m *DefaultManager) removeWorktree(ctx context.Context, issueNumber int) error {
	// worktreeのパス（例: .git/osoba/123）
	worktreePath := fmt.Sprintf(".git/osoba/%d", issueNumber)

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
