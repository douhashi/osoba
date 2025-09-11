package github

import (
	"context"
)

// GitHubClient はGitHub APIクライアントのインターフェース
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
	ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*Issue, error)
	ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*Issue, error)
	ListClosedIssues(ctx context.Context, owner, repo string) ([]*Issue, error)
	ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*PullRequest, error)
	GetRateLimit(ctx context.Context) (*RateLimits, error)
	TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error)
	TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error)
	EnsureLabelsExist(ctx context.Context, owner, repo string) error
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error
	RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
	AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
	TransitionLabels(ctx context.Context, owner, repo string, issueNumber int, removeLabel, addLabel string) error
	GetPullRequestForIssue(ctx context.Context, issueNumber int) (*PullRequest, error)
	MergePullRequest(ctx context.Context, prNumber int) error
	GetPullRequestStatus(ctx context.Context, prNumber int) (*PullRequest, error)
	GetClosingIssueNumber(ctx context.Context, prNumber int) (int, error)
}
