package git

import (
	"context"
	"fmt"
	"path/filepath"
)

// Phase はworktreeのフェーズを表す型
type Phase string

const (
	// PhasePlan は計画フェーズ
	PhasePlan Phase = "plan"
	// PhaseImplementation は実装フェーズ
	PhaseImplementation Phase = "implementation"
	// PhaseReview はレビューフェーズ
	PhaseReview Phase = "review"
)

// WorktreeManager はIssueベースのworktree管理を行うインターフェース
type WorktreeManager interface {
	// UpdateMainBranch はmainブランチを最新化する
	UpdateMainBranch(ctx context.Context) error

	// CreateWorktree は指定されたIssueとフェーズのworktreeを作成する
	CreateWorktree(ctx context.Context, issueNumber int, phase Phase) error

	// RemoveWorktree は指定されたIssueとフェーズのworktreeを削除する
	RemoveWorktree(ctx context.Context, issueNumber int, phase Phase) error

	// GetWorktreePath は指定されたIssueとフェーズのworktreeパスを返す
	GetWorktreePath(issueNumber int, phase Phase) string

	// WorktreeExists は指定されたworktreeが存在するかを確認する
	WorktreeExists(ctx context.Context, issueNumber int, phase Phase) (bool, error)

	// GetWorktreePathForIssue は指定されたIssueのworktreeパスを返す（フェーズを含まない）
	GetWorktreePathForIssue(issueNumber int) string

	// WorktreeExistsForIssue は指定されたIssueのworktreeが存在するかを確認する
	WorktreeExistsForIssue(ctx context.Context, issueNumber int) (bool, error)

	// CreateWorktreeForIssue は指定されたIssueのworktreeを作成する
	CreateWorktreeForIssue(ctx context.Context, issueNumber int) error

	// RemoveWorktreeForIssue は指定されたIssueのworktreeを削除する
	RemoveWorktreeForIssue(ctx context.Context, issueNumber int) error
}

// worktreeManager はWorktreeManagerの実装
type worktreeManager struct {
	repository Repository
	worktree   *Worktree
	branch     *Branch
	sync       *Sync
	basePath   string
}

// NewWorktreeManager は新しいWorktreeManagerインスタンスを作成する
func NewWorktreeManager(repository Repository, worktree *Worktree, branch *Branch, sync *Sync) (WorktreeManager, error) {
	// リポジトリのルートパスを取得
	basePath, err := repository.GetRootPath(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get repository root path: %w", err)
	}

	return &worktreeManager{
		repository: repository,
		worktree:   worktree,
		branch:     branch,
		sync:       sync,
		basePath:   basePath,
	}, nil
}

// UpdateMainBranch はmainブランチを最新化する
func (m *worktreeManager) UpdateMainBranch(ctx context.Context) error {
	// 現在のブランチを保存
	currentBranch, err := m.branch.GetCurrent(ctx, m.basePath)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// mainブランチに切り替え
	if err := m.branch.Checkout(ctx, m.basePath, "main", false); err != nil {
		return fmt.Errorf("failed to checkout main branch: %w", err)
	}

	// mainブランチを最新化
	if err := m.sync.Pull(ctx, m.basePath, "origin", "main", false); err != nil {
		// 元のブランチに戻す
		_ = m.branch.Checkout(ctx, m.basePath, currentBranch, false)
		return fmt.Errorf("failed to pull main branch: %w", err)
	}

	// 元のブランチに戻す（mainでない場合）
	if currentBranch != "main" {
		if err := m.branch.Checkout(ctx, m.basePath, currentBranch, false); err != nil {
			return fmt.Errorf("failed to checkout back to %s: %w", currentBranch, err)
		}
	}

	return nil
}

// CreateWorktree は指定されたIssueとフェーズのworktreeを作成する
func (m *worktreeManager) CreateWorktree(ctx context.Context, issueNumber int, phase Phase) error {
	// worktreeパスとブランチ名を生成
	worktreePath := m.GetWorktreePath(issueNumber, phase)
	branchName := m.generateBranchName(issueNumber, phase)

	// 既存のworktreeが存在する場合は削除
	exists, err := m.WorktreeExists(ctx, issueNumber, phase)
	if err != nil {
		return fmt.Errorf("failed to check worktree existence: %w", err)
	}
	if exists {
		if err := m.RemoveWorktree(ctx, issueNumber, phase); err != nil {
			return fmt.Errorf("failed to remove existing worktree: %w", err)
		}
	}

	// 既存のブランチが存在する場合は削除
	branches, err := m.branch.List(ctx, m.basePath, false)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	for _, b := range branches {
		if b.Name == branchName {
			// ローカルブランチを削除
			if err := m.branch.Delete(ctx, m.basePath, branchName, true); err != nil {
				return fmt.Errorf("failed to delete existing branch: %w", err)
			}
			break
		}
	}

	// ブランチを作成（既存の場合は削除したので再作成）
	if err := m.branch.Create(ctx, m.basePath, branchName, "main"); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// 新しいworktreeを作成
	if err := m.worktree.Create(ctx, m.basePath, worktreePath, branchName); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	return nil
}

// RemoveWorktree は指定されたIssueとフェーズのworktreeを削除する
func (m *worktreeManager) RemoveWorktree(ctx context.Context, issueNumber int, phase Phase) error {
	worktreePath := m.GetWorktreePath(issueNumber, phase)

	// worktreeを削除
	if err := m.worktree.Remove(ctx, m.basePath, worktreePath); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// ブランチも削除
	branchName := m.generateBranchName(issueNumber, phase)
	if err := m.branch.Delete(ctx, m.basePath, branchName, true); err != nil {
		// ブランチ削除のエラーは無視（既に削除されている可能性がある）
		return nil
	}

	return nil
}

// GetWorktreePath は指定されたIssueとフェーズのworktreeパスを返す
func (m *worktreeManager) GetWorktreePath(issueNumber int, phase Phase) string {
	// .git/osoba/worktrees/{issue番号}-{フェーズ}
	return filepath.Join(m.basePath, ".git", "osoba", "worktrees", fmt.Sprintf("%d-%s", issueNumber, phase))
}

// WorktreeExists は指定されたworktreeが存在するかを確認する
func (m *worktreeManager) WorktreeExists(ctx context.Context, issueNumber int, phase Phase) (bool, error) {
	worktreePath := m.GetWorktreePath(issueNumber, phase)

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

// generateBranchName はIssue番号とフェーズからブランチ名を生成する
func (m *worktreeManager) generateBranchName(issueNumber int, phase Phase) string {
	// osoba/#{issue番号}-{フェーズ}
	return fmt.Sprintf("osoba/#%d-%s", issueNumber, phase)
}
