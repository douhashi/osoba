package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/douhashi/osoba/internal/logger"
)

// mockGhExecutor はテスト用のghコマンド実行モック
type mockGhExecutor struct {
	output      string
	err         error
	lastCommand string
}

// Client はテスト用のクライアント構造体
type Client struct {
	owner    string
	repo     string
	executor *mockGhExecutor
	logger   logger.Logger
}

// GetPullRequestForIssue はテスト用の実装
func (c *Client) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*PullRequest, error) {
	args := []string{
		"pr", "list",
		"--search", fmt.Sprintf("linked:%d", issueNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	c.executor.lastCommand = "gh " + joinArgs(args)

	if c.executor.err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", c.executor.err)
	}

	var prs []pullRequestWithStatus
	if err := json.Unmarshal([]byte(c.executor.output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil
	}

	pr := &PullRequest{
		Number:       prs[0].Number,
		Title:        prs[0].Title,
		State:        prs[0].State,
		Mergeable:    prs[0].Mergeable,
		IsDraft:      prs[0].IsDraft,
		HeadRefName:  prs[0].HeadRefName,
		ChecksStatus: prs[0].StatusCheckRollup.State,
	}

	return pr, nil
}

// MergePullRequest はテスト用の実装
func (c *Client) MergePullRequest(ctx context.Context, prNumber int) error {
	args := []string{
		"pr", "merge",
		strconv.Itoa(prNumber),
		"--squash",
		"--auto",
	}

	c.executor.lastCommand = "gh " + joinArgs(args)

	if c.executor.err != nil {
		return fmt.Errorf("failed to merge pull request #%d: %w", prNumber, c.executor.err)
	}

	return nil
}

// GetPullRequestStatus はテスト用の実装
func (c *Client) GetPullRequestStatus(ctx context.Context, prNumber int) (*PullRequest, error) {
	args := []string{
		"pr", "view",
		strconv.Itoa(prNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	c.executor.lastCommand = "gh " + joinArgs(args)

	if c.executor.err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d: %w", prNumber, c.executor.err)
	}

	var prStatus pullRequestWithStatus
	if err := json.Unmarshal([]byte(c.executor.output), &prStatus); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	pr := &PullRequest{
		Number:       prStatus.Number,
		Title:        prStatus.Title,
		State:        prStatus.State,
		Mergeable:    prStatus.Mergeable,
		IsDraft:      prStatus.IsDraft,
		HeadRefName:  prStatus.HeadRefName,
		ChecksStatus: prStatus.StatusCheckRollup.State,
	}

	return pr, nil
}

// joinArgs は引数を結合する
func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}
