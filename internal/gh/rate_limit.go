package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v67/github"
)

// GetRateLimit はGitHub APIのレート制限情報を取得する
func (c *Client) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	// gh api rate_limit コマンドを実行
	output, err := c.executor.Execute(ctx, "gh", "api", "rate_limit")
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	// JSON出力をパース
	var response struct {
		Resources ghRateLimitResources `json:"resources"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return nil, fmt.Errorf("failed to parse rate limit response: %w", err)
	}

	// ghRateLimitResources から github.RateLimits に変換
	limits := &github.RateLimits{
		Core:    convertToGitHubRate(response.Resources.Core),
		Search:  convertToGitHubRate(response.Resources.Search),
		GraphQL: convertToGitHubRate(response.Resources.GraphQL),
	}

	return limits, nil
}

// convertToGitHubRate は ghRateLimitResource を github.Rate に変換する
func convertToGitHubRate(ghRate ghRateLimitResource) *github.Rate {
	return &github.Rate{
		Limit:     ghRate.Limit,
		Remaining: ghRate.Remaining,
		Reset:     github.Timestamp{Time: time.Unix(ghRate.Reset, 0)},
	}
}
