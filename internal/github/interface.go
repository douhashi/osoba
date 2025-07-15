package github

import (
	"context"

	"github.com/google/go-github/v67/github"
)

// GitHubClient はGitHub APIクライアントのインターフェース
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error)
	ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error)
	GetRateLimit(ctx context.Context) (*github.RateLimits, error)
	TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error)
	TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error)
	EnsureLabelsExist(ctx context.Context, owner, repo string) error
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error
}
