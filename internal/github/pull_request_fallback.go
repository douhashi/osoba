package github

import (
	"context"
	"encoding/json"
	"fmt"
)

// SearchPullRequestByBranch はブランチ名でPRを検索する
func (c *GHClient) SearchPullRequestByBranch(ctx context.Context, branchName string) (*PullRequest, error) {
	// gh pr list --head <branch-name> --json number,title,state,mergeable,isDraft,headRefName,statusCheckRollup
	args := []string{
		"pr", "list",
		"--head", branchName,
		"--json", "number,title,state,mergeable,isDraft,headRefName,statusCheckRollup",
		"--state", "open",
	}

	if c.logger != nil {
		c.logger.Debug("Searching PR by branch name",
			"branch_name", branchName,
		)
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		if c.logger != nil {
			c.logger.Debug("Failed to search PR by branch",
				"branch_name", branchName,
				"error", err,
			)
		}
		return nil, fmt.Errorf("failed to search pull request by branch %s: %w", branchName, err)
	}

	if c.logger != nil {
		c.logger.Debug("Raw branch search output",
			"branch_name", branchName,
			"output", string(output),
		)
	}

	var prs []pullRequestWithStatus
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull request response (fallback): %w", err)
	}

	if len(prs) == 0 {
		if c.logger != nil {
			c.logger.Debug("No PR found for branch",
				"branch_name", branchName,
			)
		}
		return nil, nil // PRが存在しない
	}

	// 最初のPRを返す
	pr := &PullRequest{
		Number:       prs[0].Number,
		Title:        prs[0].Title,
		State:        prs[0].State,
		Mergeable:    prs[0].Mergeable,
		IsDraft:      prs[0].IsDraft,
		HeadRefName:  prs[0].HeadRefName,
		ChecksStatus: prs[0].getChecksStatus(),
	}

	if c.logger != nil {
		c.logger.Debug("Found PR by branch name",
			"branch_name", branchName,
			"pr_number", pr.Number,
			"state", pr.State,
			"mergeable", pr.Mergeable,
		)
	}

	return pr, nil
}
