package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/douhashi/osoba/internal/gh"
	"github.com/douhashi/osoba/internal/logger"
)

// Client はGitHub APIクライアントのエイリアス（後方互換性のため）
type Client = GHClient

// GHClient はghコマンドを使用するGitHub APIクライアント
type GHClient struct {
	executor     gh.Executor
	logger       logger.Logger
	labelManager LabelManagerInterface
}

// NewClient は新しいGitHub APIクライアントを作成する（ghコマンドベース）
func NewClient(token string) (*GHClient, error) {
	// ghコマンドは環境変数やconfigでトークンを管理するため、ここでは不要
	executor := gh.NewExecutor()
	labelManager := NewGHLabelManager(executor, nil, 3, 1*time.Second)

	return &GHClient{
		executor:     executor,
		labelManager: labelManager,
	}, nil
}

// NewClientWithLogger はログ機能付きの新しいGitHub APIクライアントを作成する（ghコマンドベース）
func NewClientWithLogger(token string, logger logger.Logger) (*GHClient, error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	executor := gh.NewExecutor()
	labelManager := NewGHLabelManager(executor, logger, 3, 1*time.Second)

	return &GHClient{
		executor:     executor,
		logger:       logger,
		labelManager: labelManager,
	}, nil
}

// GetRepository はリポジトリ情報を取得する
func (c *GHClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	args := []string{"api", fmt.Sprintf("repos/%s/%s", owner, repo)}
	output, err := c.executor.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	var repository Repository
	if err := json.Unmarshal(output, &repository); err != nil {
		return nil, fmt.Errorf("failed to parse repository response: %w", err)
	}

	return &repository, nil
}

// ListIssuesByLabels は指定されたラベルのいずれかを持つIssueを取得する（OR条件）
func (c *GHClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*Issue, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	args := []string{
		"issue", "list",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--state", "open",
		"--json", "number,title,labels,state,body,user,assignees,createdAt,updatedAt,closedAt,milestone,comments,url",
	}

	// ラベルが指定されている場合、OR条件として追加
	if len(labels) > 0 {
		args = append(args, "--label", strings.Join(labels, ","))
	}

	output, err := c.executor.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	var issues []*Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues response: %w", err)
	}

	// ghコマンドの出力からHTMLURLを設定
	for _, issue := range issues {
		if issue.HTMLURL == nil && issue.Number != nil {
			url := fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, *issue.Number)
			issue.HTMLURL = String(url)
		}
	}

	return issues, nil
}

// GetRateLimit はAPI利用制限情報を取得する
func (c *GHClient) GetRateLimit(ctx context.Context) (*RateLimits, error) {
	args := []string{"api", "rate_limit"}
	output, err := c.executor.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	var rateLimitResp gh.RateLimitResponse
	if err := json.Unmarshal(output, &rateLimitResp); err != nil {
		return nil, fmt.Errorf("failed to parse rate limit response: %w", err)
	}

	// gh.RateLimitResponseからRateLimitsに変換
	rateLimits := &RateLimits{
		Core: &RateLimit{
			Limit:     rateLimitResp.Resources.Core.Limit,
			Remaining: rateLimitResp.Resources.Core.Remaining,
			Reset:     time.Unix(rateLimitResp.Resources.Core.Reset, 0),
		},
		Search: &RateLimit{
			Limit:     rateLimitResp.Resources.Search.Limit,
			Remaining: rateLimitResp.Resources.Search.Remaining,
			Reset:     time.Unix(rateLimitResp.Resources.Search.Reset, 0),
		},
	}

	if c.logger != nil {
		c.logger.Debug("Rate limit info",
			"core_remaining", rateLimits.Core.Remaining,
			"search_remaining", rateLimits.Search.Remaining,
		)
	}

	return rateLimits, nil
}

// TransitionIssueLabel はIssueのラベルを遷移させる
func (c *GHClient) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	return c.labelManager.TransitionLabelWithRetry(ctx, owner, repo, issueNumber)
}

// TransitionIssueLabelWithInfo はIssueのラベルを遷移させ、詳細情報を返す
func (c *GHClient) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	return c.labelManager.TransitionLabelWithInfoWithRetry(ctx, owner, repo, issueNumber)
}

// EnsureLabelsExist は必要なラベルが存在することを確認する
func (c *GHClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	return c.labelManager.EnsureLabelsExistWithRetry(ctx, owner, repo)
}

// CreateIssueComment はIssueにコメントを作成する
func (c *GHClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if comment == "" {
		return errors.New("comment is required")
	}

	args := []string{
		"issue", "comment", strconv.Itoa(issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--body", comment,
	}

	if _, err := c.executor.Execute(ctx, args); err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("Created issue comment",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
		)
	}

	return nil
}

// RemoveLabel はIssueからラベルを削除する
func (c *GHClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if label == "" {
		return errors.New("label is required")
	}

	args := []string{
		"issue", "edit", fmt.Sprintf("%d", issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--remove-label", label,
	}

	if _, err := c.executor.Execute(ctx, args); err != nil {
		return fmt.Errorf("failed to remove label %s from issue #%d: %w", label, issueNumber, err)
	}

	if c.logger != nil {
		c.logger.Debug("Removed label from issue",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"label", label,
		)
	}

	return nil
}

// AddLabel はIssueにラベルを追加する
func (c *GHClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if label == "" {
		return errors.New("label is required")
	}

	args := []string{
		"issue", "edit", fmt.Sprintf("%d", issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--add-label", label,
	}

	if _, err := c.executor.Execute(ctx, args); err != nil {
		return fmt.Errorf("failed to add label %s to issue #%d: %w", label, issueNumber, err)
	}

	if c.logger != nil {
		c.logger.Debug("Added label to issue",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"label", label,
		)
	}

	return nil
}

// Ensure GHClient implements GitHubClient interface
var _ GitHubClient = (*GHClient)(nil)
