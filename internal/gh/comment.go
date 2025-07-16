package gh

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CreateIssueComment はIssueにコメントを投稿する
func (c *Client) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if issueNumber <= 0 {
		return errors.New("issue number must be positive")
	}
	if comment == "" {
		return errors.New("comment is required")
	}

	// gh issue comment コマンドを実行
	_, err := c.executor.Execute(ctx, "gh", "issue", "comment",
		strconv.Itoa(issueNumber),
		"--body", comment,
		"--repo", fmt.Sprintf("%s/%s", owner, repo))

	if err != nil {
		var execErr *ExecError
		if errors.As(err, &execErr) {
			if strings.Contains(execErr.Stderr, "Could not resolve to an Issue") {
				return errors.New("issue not found")
			}
		}
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}
