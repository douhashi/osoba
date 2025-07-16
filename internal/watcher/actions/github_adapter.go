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

	// transitionerがnilの場合（ghクライアント）、GitHubClientのTransitionIssueLabelメソッドを使用
	// このメソッドはIssueのラベルを自動的に遷移させる
	log.Printf("DEBUG: Using GitHubClient.TransitionIssueLabel for issue #%d (repo: %s/%s)", issueNumber, a.owner, a.repo)
	transitioned, err := a.client.TransitionIssueLabel(ctx, a.owner, a.repo, issueNumber)
	if err != nil {
		log.Printf("DEBUG: TransitionIssueLabel failed: %v", err)
		return fmt.Errorf("failed to transition issue label: %w", err)
	}

	if !transitioned {
		log.Printf("DEBUG: No label transition occurred for issue #%d", issueNumber)
		return fmt.Errorf("no label transition occurred for issue #%d", issueNumber)
	}

	log.Printf("DEBUG: Label transition successful for issue #%d", issueNumber)
	return nil
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
