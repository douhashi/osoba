package github

import (
	"context"
	"errors"
	"net/http"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

// Client はGitHub APIクライアントのラッパー
type Client struct {
	github *github.Client
	logger logger.Logger
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

// NewClientWithLogger はログ機能付きの新しいGitHub APIクライアントを作成する
func NewClientWithLogger(token string, logger logger.Logger) (*Client, error) {
	if token == "" {
		return nil, errors.New("GitHub token is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	// oauth2のトランスポートを作成
	baseTransport := &oauth2.Transport{
		Source: ts,
		Base:   http.DefaultTransport,
	}

	// ログ機能付きのラウンドトリッパーでラップ
	// oauth2トランスポートの後にログを配置することで、認証ヘッダーもログに記録される
	httpClient := &http.Client{
		Transport: &loggingRoundTripper{
			base:   baseTransport,
			logger: logger,
		},
	}

	return &Client{
		github: github.NewClient(httpClient),
		logger: logger,
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

	// 操作開始のログ
	if c.logger != nil {
		c.logger.Debug("listing_issues_by_labels",
			"owner", owner,
			"repo", repo,
			"labels", labels,
		)
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
			if c.logger != nil {
				c.logger.Error("failed_to_list_issues",
					"owner", owner,
					"repo", repo,
					"labels", labels,
					"error", err.Error(),
				)
			}
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	// 取得完了のログ
	if c.logger != nil {
		c.logger.Info("issues_fetched",
			"owner", owner,
			"repo", repo,
			"count", len(allIssues),
		)
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
