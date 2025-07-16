package github

import (
	"context"
)

// GitHubClient はGitHub APIクライアントのインターフェース
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
	ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*Issue, error)
	GetRateLimit(ctx context.Context) (*RateLimits, error)
	TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error)
	TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error)
	EnsureLabelsExist(ctx context.Context, owner, repo string) error
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error
	RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
	AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
}
