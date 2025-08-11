package github

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

// SearchPullRequestForIssue はIssue番号に関連するPRを複数の方法で検索する
func (c *GHClient) SearchPullRequestForIssue(ctx context.Context, issueNumber int) (*PullRequest, error) {
	if c.logger != nil {
		c.logger.Debug("Starting comprehensive PR search for issue",
			"issue_number", issueNumber,
		)
	}

	// 方法1: linked検索（従来の方法）
	pr, err := c.searchByLinkedQuery(ctx, issueNumber)
	if err == nil && pr != nil {
		if c.logger != nil {
			c.logger.Info("Found PR via linked search",
				"issue_number", issueNumber,
				"pr_number", pr.Number,
			)
		}
		return pr, nil
	}

	// 方法2: PR本文でのIssue番号検索
	pr, err = c.searchByBodyMention(ctx, issueNumber)
	if err == nil && pr != nil {
		if c.logger != nil {
			c.logger.Info("Found PR via body mention search",
				"issue_number", issueNumber,
				"pr_number", pr.Number,
			)
		}
		return pr, nil
	}

	// 方法3: ブランチ名パターンによる検索
	pr, err = c.searchByBranchPattern(ctx, issueNumber)
	if err == nil && pr != nil {
		if c.logger != nil {
			c.logger.Info("Found PR via branch pattern",
				"issue_number", issueNumber,
				"pr_number", pr.Number,
			)
		}
		return pr, nil
	}

	if c.logger != nil {
		c.logger.Debug("No PR found for issue after all search methods",
			"issue_number", issueNumber,
		)
	}

	return nil, nil
}

// searchByLinkedQuery は従来のlinked検索を行う
func (c *GHClient) searchByLinkedQuery(ctx context.Context, issueNumber int) (*PullRequest, error) {
	args := []string{
		"pr", "list",
		"--search", fmt.Sprintf("linked:%d", issueNumber),
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
		"--state", "open",
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	var prs []pullRequestWithStatus
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, err
	}

	if len(prs) > 0 {
		return convertToPullRequest(&prs[0]), nil
	}

	return nil, nil
}

// searchByBodyMention はPR本文にIssue番号が記載されているPRを検索
func (c *GHClient) searchByBodyMention(ctx context.Context, issueNumber int) (*PullRequest, error) {
	// すべてのオープンPRを取得
	args := []string{
		"pr", "list",
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup,body",
		"--state", "open",
		"--limit", "100",
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	var prs []struct {
		pullRequestWithStatus
		Body string `json:"body"`
	}

	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, err
	}

	// Issue番号の記載パターン
	patterns := []*regexp.Regexp{
		regexp.MustCompile(fmt.Sprintf(`(?i)fixes\s+#%d\b`, issueNumber)),
		regexp.MustCompile(fmt.Sprintf(`(?i)closes\s+#%d\b`, issueNumber)),
		regexp.MustCompile(fmt.Sprintf(`(?i)resolves\s+#%d\b`, issueNumber)),
		regexp.MustCompile(fmt.Sprintf(`#%d\b`, issueNumber)),
	}

	for _, pr := range prs {
		for _, pattern := range patterns {
			if pattern.MatchString(pr.Body) {
				if c.logger != nil {
					c.logger.Debug("Found PR with issue mention in body",
						"issue_number", issueNumber,
						"pr_number", pr.Number,
						"pattern", pattern.String(),
					)
				}
				return convertToPullRequest(&pr.pullRequestWithStatus), nil
			}
		}
	}

	return nil, nil
}

// searchByBranchPattern はブランチ名パターンでPRを検索
func (c *GHClient) searchByBranchPattern(ctx context.Context, issueNumber int) (*PullRequest, error) {
	branchPatterns := []string{
		fmt.Sprintf("issue-%d", issueNumber),
		fmt.Sprintf("fix-%d", issueNumber),
		fmt.Sprintf("feature-%d", issueNumber),
		fmt.Sprintf("issue/%d", issueNumber),
		fmt.Sprintf("fix/%d", issueNumber),
		fmt.Sprintf("feature/%d", issueNumber),
		fmt.Sprintf("issue/%03d", issueNumber),
		fmt.Sprintf("fix/%03d", issueNumber),
	}

	for _, pattern := range branchPatterns {
		pr, err := c.SearchPullRequestByBranch(ctx, pattern)
		if err != nil {
			continue
		}
		if pr != nil {
			return pr, nil
		}
	}

	// 正規表現パターンで幅広く検索
	args := []string{
		"pr", "list",
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
		"--state", "open",
		"--limit", "100",
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	var prs []pullRequestWithStatus
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, err
	}

	// Issue番号を含むブランチ名のパターン
	branchRegex := regexp.MustCompile(fmt.Sprintf(`\b%d\b`, issueNumber))

	for _, pr := range prs {
		if branchRegex.MatchString(pr.HeadRefName) {
			if c.logger != nil {
				c.logger.Debug("Found PR with issue number in branch name",
					"issue_number", issueNumber,
					"pr_number", pr.Number,
					"branch", pr.HeadRefName,
				)
			}
			return convertToPullRequest(&pr), nil
		}
	}

	return nil, nil
}

// convertToPullRequest はpullRequestWithStatusをPullRequestに変換
func convertToPullRequest(pr *pullRequestWithStatus) *PullRequest {
	return &PullRequest{
		Number:       pr.Number,
		Title:        pr.Title,
		State:        pr.State,
		Mergeable:    pr.Mergeable,
		IsDraft:      pr.IsDraft,
		HeadRefName:  pr.HeadRefName,
		ChecksStatus: pr.getChecksStatus(),
	}
}

// GetPullRequestForIssueWithFallback は複数の検索方法を試すラッパー関数
func (c *GHClient) GetPullRequestForIssueWithFallback(ctx context.Context, issueNumber int) (*PullRequest, error) {
	// まず従来のGetPullRequestForIssueを試す
	pr, err := c.GetPullRequestForIssue(ctx, issueNumber)
	if err == nil && pr != nil {
		return pr, nil
	}

	// 失敗したら包括的な検索を実施
	if c.logger != nil {
		c.logger.Debug("Falling back to comprehensive search",
			"issue_number", issueNumber,
		)
	}

	return c.SearchPullRequestForIssue(ctx, issueNumber)
}
