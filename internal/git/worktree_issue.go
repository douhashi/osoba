package git

import (
	"context"
	"fmt"
	"path/filepath"
)

// GetWorktreePathForIssue は指定されたIssueのworktreeパスを返す（フェーズを含まない）
func (m *worktreeManager) GetWorktreePathForIssue(issueNumber int) string {
	// .git/osoba/worktrees/issue-{issue番号}
	return filepath.Join(m.basePath, ".git", "osoba", "worktrees", fmt.Sprintf("issue-%d", issueNumber))
}

// WorktreeExistsForIssue は指定されたIssueのworktreeが存在するかを確認する
func (m *worktreeManager) WorktreeExistsForIssue(ctx context.Context, issueNumber int) (bool, error) {
	worktreePath := m.GetWorktreePathForIssue(issueNumber)

	// worktree一覧を取得
	worktrees, err := m.worktree.List(ctx, m.basePath)
	if err != nil {
		return false, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// 指定されたパスのworktreeが存在するか確認
	for _, wt := range worktrees {
		if wt.Path == worktreePath {
			return true, nil
		}
	}

	return false, nil
}

// generateBranchNameForIssue はIssue番号からブランチ名を生成する（フェーズを含まない）
func (m *worktreeManager) generateBranchNameForIssue(issueNumber int) string {
	return fmt.Sprintf("osoba/#%d", issueNumber)
}

// CreateWorktreeForIssue は指定されたIssueのworktreeを作成する
func (m *worktreeManager) CreateWorktreeForIssue(ctx context.Context, issueNumber int) error {
	if issueNumber <= 0 {
		return fmt.Errorf("invalid issue number: %d", issueNumber)
	}

	worktreePath := m.GetWorktreePathForIssue(issueNumber)
	branchName := m.generateBranchNameForIssue(issueNumber)

	// 既存のworktreeが存在する場合は削除
	exists, err := m.WorktreeExistsForIssue(ctx, issueNumber)
	if err != nil {
		return fmt.Errorf("failed to check worktree existence: %w", err)
	}
	if exists {
		if err := m.RemoveWorktreeForIssue(ctx, issueNumber); err != nil {
			return fmt.Errorf("failed to remove existing worktree: %w", err)
		}
	}

	// ブランチが存在しない場合は作成
	branchExists := m.branch.Exists(ctx, m.basePath, branchName)

	if !branchExists {
		// mainブランチを最新化
		if err := m.UpdateMainBranch(ctx); err != nil {
			return fmt.Errorf("failed to update main branch: %w", err)
		}

		// mainブランチから新しいブランチを作成
		if err := m.branch.Create(ctx, m.basePath, branchName, "main"); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}
	}

	// worktreeを作成
	if err := m.worktree.Create(ctx, m.basePath, worktreePath, branchName); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// RemoveWorktreeForIssue は指定されたIssueのworktreeを削除する
func (m *worktreeManager) RemoveWorktreeForIssue(ctx context.Context, issueNumber int) error {
	if issueNumber <= 0 {
		return fmt.Errorf("invalid issue number: %d", issueNumber)
	}

	worktreePath := m.GetWorktreePathForIssue(issueNumber)

	// worktreeを削除
	if err := m.worktree.Remove(ctx, m.basePath, worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// ブランチも削除
	branchName := m.generateBranchNameForIssue(issueNumber)
	if err := m.branch.Delete(ctx, m.basePath, branchName, true); err != nil {
		// ブランチ削除のエラーは無視（既に削除されている可能性がある）
		// ログに記録する程度に留める
		_ = err
	}

	return nil
}
