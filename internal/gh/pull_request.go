package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	internalGitHub "github.com/douhashi/osoba/internal/github"
)

// GetPullRequestForIssue はIssue番号に関連付けられたPRを取得する
func (c *Client) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*internalGitHub.PullRequest, error) {
	// gh pr list --search "linked:<issue-number>" --json number,title,state,mergeable,isDraft,headRefName,statusCheckRollup
	args := []string{
		"pr", "list",
		"--search", fmt.Sprintf("linked:%d", issueNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	output, err := c.executor.Execute(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	// JSONをパース
	var prs []struct {
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

	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	if len(prs) == 0 {
		return nil, nil // PRが存在しない
	}

	// 最初のPRを返す（通常、1つのIssueに対して1つのPR）
	pr := &internalGitHub.PullRequest{
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

// MergePullRequest は指定されたPRをマージする
func (c *Client) MergePullRequest(ctx context.Context, prNumber int) error {
	// gh pr merge <pr-number> --squash --auto
	args := []string{
		"pr", "merge",
		strconv.Itoa(prNumber),
		"--squash",
		"--auto",
	}

	_, err := c.executor.Execute(ctx, "gh", args...)
	if err != nil {
		return fmt.Errorf("failed to merge pull request #%d: %w", prNumber, err)
	}

	return nil
}

// GetPullRequestStatus はPRの現在の状態を取得する
func (c *Client) GetPullRequestStatus(ctx context.Context, prNumber int) (*internalGitHub.PullRequest, error) {
	// gh pr view <pr-number> --json number,title,state,mergeable,isDraft,headRefName,statusCheckRollup
	args := []string{
		"pr", "view",
		strconv.Itoa(prNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	output, err := c.executor.Execute(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d: %w", prNumber, err)
	}

	// JSONをパース
	var prStatus struct {
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

	if err := json.Unmarshal([]byte(output), &prStatus); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	pr := &internalGitHub.PullRequest{
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

// ListPullRequestsByLabels は指定されたラベルを持つPRをリストする
func (c *Client) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*internalGitHub.PullRequest, error) {
	if len(labels) == 0 {
		return []*internalGitHub.PullRequest{}, nil
	}

	// gh pr list --state open --label <label> --json number,title,state,mergeable,isDraft,headRefName,statusCheckRollup
	args := []string{
		"pr", "list",
		"--state", "open",
		"--label", labels[0], // 最初のラベルで検索
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
	}

	output, err := c.executor.Execute(ctx, "gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests by labels: %w", err)
	}

	// JSONをパース
	var prs []struct {
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

	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response: %w", err)
	}

	// 結果をGitHubPullRequest構造体に変換
	var result []*internalGitHub.PullRequest
	for _, pr := range prs {
		result = append(result, &internalGitHub.PullRequest{
			Number:       pr.Number,
			Title:        pr.Title,
			State:        pr.State,
			Mergeable:    pr.Mergeable,
			IsDraft:      pr.IsDraft,
			HeadRefName:  pr.HeadRefName,
			ChecksStatus: pr.StatusCheckRollup.State,
		})
	}

	return result, nil
}
