package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/douhashi/osoba/internal/github"
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
		Core: &github.RateLimit{
			Limit:     response.Resources.Core.Limit,
			Remaining: response.Resources.Core.Remaining,
			Reset:     time.Unix(response.Resources.Core.Reset, 0),
		},
		Search: &github.RateLimit{
			Limit:     response.Resources.Search.Limit,
			Remaining: response.Resources.Search.Remaining,
			Reset:     time.Unix(response.Resources.Search.Reset, 0),
		},
		GraphQL: &github.RateLimit{
			Limit:     response.Resources.GraphQL.Limit,
			Remaining: response.Resources.GraphQL.Remaining,
			Reset:     time.Unix(response.Resources.GraphQL.Reset, 0),
		},
	}

	return limits, nil
}
