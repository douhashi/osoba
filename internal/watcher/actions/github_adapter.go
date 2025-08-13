package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
)

// GitHubAdapter はgithub.Clientをactions.GitHubClientInterfaceに適合させるアダプター
type GitHubAdapter struct {
	client       github.GitHubClient
	owner        string
	repo         string
	transitioner github.LabelTransitioner
}

// NewGitHubAdapter は新しいGitHubAdapterを作成する
func NewGitHubAdapter(client github.GitHubClient, owner, repo string, transitioner github.LabelTransitioner) *GitHubAdapter {
	return &GitHubAdapter{
		client:       client,
		owner:        owner,
		repo:         repo,
		transitioner: transitioner,
	}
}

// CreateIssueComment はIssueにコメントを投稿する
func (a *GitHubAdapter) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	// 引数のowner/repoは無視して、アダプター作成時に設定されたものを使用
	return a.client.CreateIssueComment(ctx, a.owner, a.repo, issueNumber, comment)
}

// TransitionLabel はラベルを遷移させる
func (a *GitHubAdapter) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	if a.transitioner != nil {
		// LabelTransitionerが利用可能な場合（APIクライアント）
		log.Printf("DEBUG: Using LabelTransitioner for issue #%d: %s -> %s", issueNumber, from, to)
		return a.transitioner.TransitionLabel(ctx, issueNumber, from, to)
	}

	// transitionerがnilの場合（ghクライアント）、手動でラベルを削除/追加する
	log.Printf("DEBUG: Using manual label transition for issue #%d: %s -> %s", issueNumber, from, to)

	// 古いラベルを削除
	if err := a.client.RemoveLabel(ctx, a.owner, a.repo, issueNumber, from); err != nil {
		log.Printf("DEBUG: Failed to remove label %s: %v", from, err)
		return fmt.Errorf("failed to remove label %s: %w", from, err)
	}

	// 新しいラベルを追加
	if err := a.client.AddLabel(ctx, a.owner, a.repo, issueNumber, to); err != nil {
		log.Printf("DEBUG: Failed to add label %s: %v", to, err)
		return fmt.Errorf("failed to add label %s: %w", to, err)
	}

	log.Printf("DEBUG: Label transition successful for issue #%d: %s -> %s", issueNumber, from, to)
	return nil
}

// AddLabel はラベルを追加する
func (a *GitHubAdapter) AddLabel(ctx context.Context, issueNumber int, label string) error {
	return a.client.AddLabel(ctx, a.owner, a.repo, issueNumber, label)
}

// RemoveLabel はラベルを削除する
func (a *GitHubAdapter) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	return a.client.RemoveLabel(ctx, a.owner, a.repo, issueNumber, label)
}

// GetPullRequestForIssue はIssueに関連するPRを取得する
func (a *GitHubAdapter) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
	return a.client.GetPullRequestForIssue(ctx, issueNumber)
}

// ConfigAdapter はconfig.Configをactions.ConfigProviderに適合させるアダプター
type ConfigAdapter struct {
	config *config.Config
}

// NewConfigAdapter は新しいConfigAdapterを作成する
func NewConfigAdapter(cfg *config.Config) *ConfigAdapter {
	return &ConfigAdapter{
		config: cfg,
	}
}

// GetPhaseMessage は指定されたフェーズのメッセージを返す
func (a *ConfigAdapter) GetPhaseMessage(phase string) (string, bool) {
	if a.config == nil {
		return "", false
	}
	return a.config.GetPhaseMessage(phase)
}
