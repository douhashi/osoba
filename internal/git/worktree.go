package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/douhashi/osoba/internal/logger"
)

// WorktreeInfo はworktreeの情報を表す構造体
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
}

// Worktree はgit worktree操作を管理する構造体
type Worktree struct {
	logger  logger.Logger
	command *Command
}

// NewWorktree は新しいWorktreeインスタンスを作成する
func NewWorktree(logger logger.Logger) *Worktree {
	return &Worktree{
		logger:  logger,
		command: NewCommand(logger),
	}
}

// Create は新しいworktreeを作成する
func (w *Worktree) Create(ctx context.Context, repoPath, worktreePath, branch string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"worktreePath", worktreePath,
		"branch", branch,
	}

	w.logger.Info("Creating git worktree", logFields...)

	// worktreeを作成（ブランチは既に存在するので-bフラグは使わない）
	args := []string{"worktree", "add", worktreePath, branch}
	output, err := w.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		w.logger.Error("Failed to create git worktree", errorFields...)
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	w.logger.Info("Git worktree created successfully", successFields...)

	return nil
}

// Remove は指定されたworktreeを削除する
func (w *Worktree) Remove(ctx context.Context, repoPath, worktreePath string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"worktreePath", worktreePath,
	}

	w.logger.Info("Removing git worktree", logFields...)

	// worktreeを削除
	args := []string{"worktree", "remove", worktreePath}
	output, err := w.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		// 強制削除を試みる
		w.logger.Warn("Normal removal failed, trying force removal", append(logFields, "error", err.Error())...)

		args = []string{"worktree", "remove", "--force", worktreePath}
		output, err = w.command.Run(ctx, "git", args, repoPath)
		if err != nil {
			errorFields := append(logFields, "error", err.Error())
			w.logger.Error("Failed to remove git worktree", errorFields...)
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	}

	// worktreeディレクトリが残っている場合は削除
	if _, err := os.Stat(worktreePath); err == nil {
		w.logger.Debug("Removing worktree directory", "path", worktreePath)
		if err := os.RemoveAll(worktreePath); err != nil {
			w.logger.Warn("Failed to remove worktree directory", "path", worktreePath, "error", err.Error())
		}
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	w.logger.Info("Git worktree removed successfully", successFields...)

	return nil
}

// List は全てのworktreeの情報を取得する
func (w *Worktree) List(ctx context.Context, repoPath string) ([]WorktreeInfo, error) {
	logFields := []interface{}{
		"repoPath", repoPath,
	}

	w.logger.Info("Listing git worktrees", logFields...)

	// worktree一覧を取得
	args := []string{"worktree", "list", "--porcelain"}
	output, err := w.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		w.logger.Error("Failed to list git worktrees", errorFields...)
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// 出力をパース
	worktrees := parseWorktreeList(output)

	// 成功ログ
	successFields := append(logFields, "count", len(worktrees))
	w.logger.Info("Git worktrees listed successfully", successFields...)

	// 各worktreeの詳細をデバッグログに出力
	for i, wt := range worktrees {
		w.logger.Debug("Worktree info",
			"index", i,
			"path", wt.Path,
			"branch", wt.Branch,
			"commit", wt.Commit,
		)
	}

	return worktrees, nil
}

// parseWorktreeList はgit worktree list --porcelainの出力をパースする
func parseWorktreeList(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	lines := strings.Split(output, "\n")

	var current WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "HEAD "):
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch ")
			// refs/heads/ プレフィックスを削除
			current.Branch = strings.TrimPrefix(current.Branch, "refs/heads/")
		}
	}

	// 最後のエントリを追加
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// GetWorktreeRoot は現在のディレクトリがworktree内にある場合、そのルートパスを返す
func (w *Worktree) GetWorktreeRoot(ctx context.Context, currentPath string) (string, error) {
	// git rev-parse --show-toplevelを実行
	output, err := w.command.Run(ctx, "git", []string{"rev-parse", "--show-toplevel"}, currentPath)
	if err != nil {
		return "", fmt.Errorf("failed to get worktree root: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// IsInsideWorktree は指定されたパスがgit worktree内にあるかを確認する
func (w *Worktree) IsInsideWorktree(ctx context.Context, path string) bool {
	// git rev-parse --is-inside-work-treeを実行
	output, err := w.command.Run(ctx, "git", []string{"rev-parse", "--is-inside-work-tree"}, path)
	if err != nil {
		return false
	}

	return strings.TrimSpace(output) == "true"
}

// GetMainWorktreePath はメインのworktreeのパスを取得する
func (w *Worktree) GetMainWorktreePath(ctx context.Context, repoPath string) (string, error) {
	// git rev-parse --git-common-dirを実行して.gitディレクトリを取得
	output, err := w.command.Run(ctx, "git", []string{"rev-parse", "--git-common-dir"}, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get git common dir: %w", err)
	}

	gitDir := strings.TrimSpace(output)
	// .gitディレクトリの親ディレクトリがメインworktree
	mainPath := filepath.Dir(gitDir)

	return mainPath, nil
}
