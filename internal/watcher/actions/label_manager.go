package actions

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/github"
)

// DefaultLabelManager はデフォルトのラベル管理実装
type DefaultLabelManager struct {
	GitHubClient github.GitHubClient
}

// TransitionLabel はラベルを遷移させる
func (m *DefaultLabelManager) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	if m.GitHubClient == nil {
		return fmt.Errorf("GitHub client is not initialized")
	}

	// 古いラベルを削除
	if err := m.RemoveLabel(ctx, issueNumber, from); err != nil {
		return fmt.Errorf("failed to remove label %s: %w", from, err)
	}

	// 新しいラベルを追加
	if err := m.AddLabel(ctx, issueNumber, to); err != nil {
		return fmt.Errorf("failed to add label %s: %w", to, err)
	}

	return nil
}

// AddLabel はラベルを追加する
func (m *DefaultLabelManager) AddLabel(ctx context.Context, issueNumber int, label string) error {
	if m.GitHubClient == nil {
		return fmt.Errorf("GitHub client is not initialized")
	}

	// TODO: 実際のGitHub API呼び出しを実装
	// m.GitHubClient.AddLabelToIssue(ctx, issueNumber, label)
	return nil
}

// RemoveLabel はラベルを削除する
func (m *DefaultLabelManager) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	if m.GitHubClient == nil {
		return fmt.Errorf("GitHub client is not initialized")
	}

	// TODO: 実際のGitHub API呼び出しを実装
	// m.GitHubClient.RemoveLabelFromIssue(ctx, issueNumber, label)
	return nil
}
