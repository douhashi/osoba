package actions

import (
	"context"
	"fmt"

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
	if a.transitioner == nil {
		return fmt.Errorf("label transitioner is not initialized")
	}
	return a.transitioner.TransitionLabel(ctx, issueNumber, from, to)
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
