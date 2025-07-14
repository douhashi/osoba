package github

import (
	"context"

	"github.com/google/go-github/v50/github"
)

// GitHubClient はGitHub APIクライアントのインターフェース
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error)
	ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error)
	GetRateLimit(ctx context.Context) (*github.RateLimits, error)
}
