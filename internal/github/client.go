package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

// Client はGitHub APIクライアントのラッパー
type Client struct {
	github *github.Client
}

// NewClient は新しいGitHub APIクライアントを作成する
func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, errors.New("GitHub token is required")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		github: github.NewClient(tc),
	}, nil
}

// GetRepository はリポジトリ情報を取得する
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	repository, _, err := c.github.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	return repository, nil
}

// ListIssuesByLabels は指定されたラベルを持つIssueを取得する
func (c *Client) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	opts := &github.IssueListByRepoOptions{
		Labels: labels,
		State:  "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := c.github.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// GetRateLimit はGitHub APIのレート制限情報を取得する
func (c *Client) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	rateLimit, _, err := c.github.RateLimits(ctx)
	if err != nil {
		return nil, err
	}

	return rateLimit, nil
}
