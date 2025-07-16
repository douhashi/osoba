package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetRateLimit はGitHub APIのレート制限情報を取得する
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimitResponse, error) {
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

	// ghRateLimitResources から RateLimitResponse に変換
	limits := &RateLimitResponse{
		Resources: RateLimitResources{
			Core:    convertToRateLimit(response.Resources.Core),
			Search:  convertToRateLimit(response.Resources.Search),
			GraphQL: convertToRateLimit(response.Resources.GraphQL),
		},
	}

	return limits, nil
}

// convertToRateLimit は ghRateLimitResource を RateLimit に変換する
func convertToRateLimit(ghRate ghRateLimitResource) RateLimit {
	return RateLimit{
		Limit:     ghRate.Limit,
		Remaining: ghRate.Remaining,
		Reset:     ghRate.Reset,
	}
}
