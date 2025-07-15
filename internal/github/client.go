package github

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

// Client はGitHub APIクライアントのラッパー
type Client struct {
	github       *github.Client
	logger       logger.Logger
	labelManager LabelManagerInterface
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

	ghClient := github.NewClient(tc)
	return &Client{
		github:       ghClient,
		labelManager: NewLabelManagerWithRetry(ghClient.Issues, 3, 1*time.Second),
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

	ghClient := github.NewClient(httpClient)
	return &Client{
		github:       ghClient,
		logger:       logger,
		labelManager: NewLabelManagerWithRetry(ghClient.Issues, 3, 1*time.Second),
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

// ListIssuesByLabels は指定されたラベルのいずれかを持つIssueを取得する（OR条件）
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

	// OR条件で検索するため、各ラベルごとに検索して結果をマージ
	issueMap := make(map[int]*github.Issue) // 重複を避けるためのマップ

	for _, label := range labels {
		opts := &github.IssueListByRepoOptions{
			Labels: []string{label}, // 単一ラベルで検索
			State:  "open",
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}

		for {
			issues, resp, err := c.github.Issues.ListByRepo(ctx, owner, repo, opts)
			if err != nil {
				if c.logger != nil {
					c.logger.Error("failed_to_list_issues",
						"owner", owner,
						"repo", repo,
						"label", label,
						"error", err.Error(),
					)
				}
				return nil, err
			}

			// 結果をマップに追加（重複を自動除去）
			for _, issue := range issues {
				issueMap[*issue.Number] = issue
			}

			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
	}

	// マップからスライスに変換
	var allIssues []*github.Issue
	for _, issue := range issueMap {
		allIssues = append(allIssues, issue)
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

// TransitionIssueLabel はIssueのラベルをトリガーラベルから実行中ラベルに遷移させる
func (c *Client) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	if owner == "" {
		return false, errors.New("owner is required")
	}
	if repo == "" {
		return false, errors.New("repo is required")
	}
	if issueNumber <= 0 {
		return false, errors.New("issue number must be positive")
	}

	// ログ出力
	if c.logger != nil {
		c.logger.Debug("transitioning_issue_label",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
		)
	}

	transitioned, err := c.labelManager.TransitionLabelWithRetry(ctx, owner, repo, issueNumber)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed_to_transition_label",
				"owner", owner,
				"repo", repo,
				"issue", issueNumber,
				"error", err.Error(),
			)
		}
		return false, err
	}

	if transitioned && c.logger != nil {
		c.logger.Info("label_transitioned",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
		)
	}

	return transitioned, nil
}

// TransitionIssueLabelWithInfo はIssueのラベルをトリガーラベルから実行中ラベルに遷移させ、遷移情報を返す
func (c *Client) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	if owner == "" {
		return false, nil, errors.New("owner is required")
	}
	if repo == "" {
		return false, nil, errors.New("repo is required")
	}
	if issueNumber <= 0 {
		return false, nil, errors.New("issue number must be positive")
	}

	// ログ出力
	if c.logger != nil {
		c.logger.Debug("transitioning_issue_label_with_info",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
		)
	}

	transitioned, info, err := c.labelManager.TransitionLabelWithInfoWithRetry(ctx, owner, repo, issueNumber)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed_to_transition_label",
				"owner", owner,
				"repo", repo,
				"issue", issueNumber,
				"error", err.Error(),
			)
		}
		return false, nil, err
	}

	if c.logger != nil && transitioned && info != nil {
		c.logger.Info("label_transitioned",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"from", info.From,
			"to", info.To,
		)
	}

	return transitioned, info, nil
}

// EnsureLabelsExist は必要なラベルがリポジトリに存在することを保証する
func (c *Client) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}

	// ログ出力
	if c.logger != nil {
		c.logger.Debug("ensuring_labels_exist",
			"owner", owner,
			"repo", repo,
		)
	}

	err := c.labelManager.EnsureLabelsExistWithRetry(ctx, owner, repo)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed_to_ensure_labels",
				"owner", owner,
				"repo", repo,
				"error", err.Error(),
			)
		}
		return err
	}

	if c.logger != nil {
		c.logger.Info("labels_ensured",
			"owner", owner,
			"repo", repo,
		)
	}

	return nil
}
