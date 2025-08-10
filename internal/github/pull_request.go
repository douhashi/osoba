package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	// gh pr list --search "linked:<issue-number>" --json number,title,state,mergeable,isDraft,headRefName,statusCheckRollup
	args := []string{
		"pr", "list",
		"--search", fmt.Sprintf("linked:%d", issueNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	var prs []pullRequestWithStatus
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil // PRが存在しない
	}

	// 最初のPRを返す（通常、1つのIssueに対して1つのPR）
	pr := &PullRequest{
		Number:       prs[0].Number,
		Title:        prs[0].Title,
		State:        prs[0].State,
		Mergeable:    prs[0].Mergeable,
		IsDraft:      prs[0].IsDraft,
		HeadRefName:  prs[0].HeadRefName,
		ChecksStatus: prs[0].StatusCheckRollup.State,
	}

	if c.logger != nil {
		c.logger.Debug("Found pull request for issue",
			"issue_number", issueNumber,
			"pr_number", pr.Number,
			"state", pr.State,
			"mergeable", pr.Mergeable,
		)
	}

	return pr, nil
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
