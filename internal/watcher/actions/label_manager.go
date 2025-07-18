package actions

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/github"
)

// ActionsLabelManager はラベル管理のインターフェース
type ActionsLabelManager interface {
	TransitionLabel(ctx context.Context, issueNumber int, from, to string) error
	AddLabel(ctx context.Context, issueNumber int, label string) error
	RemoveLabel(ctx context.Context, issueNumber int, label string) error
}

// DefaultLabelManager はデフォルトのラベル管理実装
type DefaultLabelManager struct {
	Owner        string
	Repo         string
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

	return m.GitHubClient.AddLabel(ctx, m.Owner, m.Repo, issueNumber, label)
}

// RemoveLabel はラベルを削除する
func (m *DefaultLabelManager) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	if m.GitHubClient == nil {
		return fmt.Errorf("GitHub client is not initialized")
	}

	return m.GitHubClient.RemoveLabel(ctx, m.Owner, m.Repo, issueNumber, label)
}
