package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PullRequest はプルリクエストの情報を表す
type PullRequest struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Mergeable    string `json:"mergeable"`
	IsDraft      bool   `json:"isDraft"`
	HeadRefName  string `json:"headRefName"`
	ChecksStatus string `json:"-"`
}

// pullRequestWithStatus はghコマンドのJSON出力用の構造体
type pullRequestWithStatus struct {
	Number            int    `json:"number"`
	Title             string `json:"title"`
	State             string `json:"state"`
	Mergeable         string `json:"mergeable"`
	IsDraft           bool   `json:"isDraft"`
	HeadRefName       string `json:"headRefName"`
	StatusCheckRollup struct {
		State string `json:"state"`
	} `json:"statusCheckRollup"`
}

// GetPullRequestForIssue はIssue番号に関連付けられたPRを取得する
func (c *GHClient) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*PullRequest, error) {
	if c.logger != nil {
		c.logger.Debug("Starting PR search for issue",
			"issue_number", issueNumber,
		)
	}

	// 方法1: GraphQL APIを使用（最も確実）
	if c.owner != "" && c.repo != "" {
		pr, err := c.GetPullRequestForIssueViaGraphQL(ctx, issueNumber)
		if err == nil && pr != nil {
			if c.logger != nil {
				c.logger.Info("Found PR via GraphQL",
					"issue_number", issueNumber,
					"pr_number", pr.Number,
				)
			}
			return pr, nil
		}
	}

	// 方法2: 単純な番号検索
	args := []string{
		"pr", "list",
		"--search", fmt.Sprintf("%d", issueNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup,body",
		"--state", "open",
	}

	if c.logger != nil {
		c.logger.Debug("Executing gh command for PR search",
			"issue_number", issueNumber,
			"search_query", fmt.Sprintf("linked:%d", issueNumber),
		)
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to execute gh command",
				"issue_number", issueNumber,
				"error", err,
			)
		}
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("Raw gh command output",
			"issue_number", issueNumber,
			"output", string(output),
		)
	}

	var prs []struct {
		pullRequestWithStatus
		Body string `json:"body"`
	}
	if err := json.Unmarshal(output, &prs); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to parse PR response",
				"issue_number", issueNumber,
				"raw_output", string(output),
				"error", err,
			)
		}
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	// Issue番号が記載されているPRを探す
	for _, pr := range prs {
		// PR本文にIssue番号への言及があるか確認
		if strings.Contains(pr.Body, fmt.Sprintf("#%d", issueNumber)) ||
			strings.Contains(pr.Body, fmt.Sprintf("fixes #%d", issueNumber)) ||
			strings.Contains(pr.Body, fmt.Sprintf("Fixes #%d", issueNumber)) ||
			strings.Contains(pr.Body, fmt.Sprintf("closes #%d", issueNumber)) ||
			strings.Contains(pr.Body, fmt.Sprintf("Closes #%d", issueNumber)) {

			if c.logger != nil {
				c.logger.Debug("Found PR with issue reference in body",
					"issue_number", issueNumber,
					"pr_number", pr.Number,
				)
			}

			// 最初のPRを返す
			result := &PullRequest{
				Number:       pr.Number,
				Title:        pr.Title,
				State:        pr.State,
				Mergeable:    pr.Mergeable,
				IsDraft:      pr.IsDraft,
				HeadRefName:  pr.HeadRefName,
				ChecksStatus: pr.StatusCheckRollup.State,
			}
			return result, nil
		}
	}

	if c.logger != nil {
		c.logger.Debug("No pull requests found with issue reference",
			"issue_number", issueNumber,
			"searched_prs", len(prs),
		)
	}

	return nil, nil // PRが存在しない

}

// MergePullRequest は指定されたPRをマージする
func (c *GHClient) MergePullRequest(ctx context.Context, prNumber int) error {
	// gh pr merge <pr-number> --squash --auto
	args := []string{
		"pr", "merge",
		strconv.Itoa(prNumber),
		"--squash",
		"--auto",
	}

	if c.logger != nil {
		c.logger.Info("Merging pull request",
			"pr_number", prNumber,
		)
	}

	_, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to merge pull request #%d: %w", prNumber, err)
	}

	if c.logger != nil {
		c.logger.Info("Successfully merged pull request",
			"pr_number", prNumber,
		)
	}

	return nil
}

// GetPullRequestStatus はPRの現在の状態を取得する
func (c *GHClient) GetPullRequestStatus(ctx context.Context, prNumber int) (*PullRequest, error) {
	// gh pr view <pr-number> --json number,title,state,mergeable,isDraft,headRefName,statusCheckRollup
	args := []string{
		"pr", "view",
		strconv.Itoa(prNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d: %w", prNumber, err)
	}

	var prStatus pullRequestWithStatus
	if err := json.Unmarshal(output, &prStatus); err != nil {
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

	if c.logger != nil {
		c.logger.Debug("Got pull request status",
			"pr_number", pr.Number,
			"state", pr.State,
			"mergeable", pr.Mergeable,
			"checks_status", pr.ChecksStatus,
		)
	}

	return pr, nil
}

// ListPullRequestsByLabels は指定されたラベルを持つPRをリストする
func (c *GHClient) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*PullRequest, error) {
	if c.logger != nil {
		c.logger.Debug("Listing pull requests by labels",
			"owner", owner,
			"repo", repo,
			"labels", labels,
		)
	}

	if len(labels) == 0 {
		return []*PullRequest{}, nil
	}

	// GraphQL APIを使用してラベル付きPRを検索
	return c.ListPullRequestsByLabelsViaGraphQL(ctx, owner, repo, labels)
}
