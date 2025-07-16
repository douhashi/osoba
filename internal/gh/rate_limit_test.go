package gh

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetRateLimit(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockCommandExecutor)
		expectedLimits *github.RateLimits
		expectedError  string
	}{
		{
			name: "正常系: レート制限情報の取得",
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					expectedCmd := []string{"api", "rate_limit"}
					if command == "gh" && equalStringSlices(args, expectedCmd) {
						return `{
							"resources": {
								"core": {
									"limit": 5000,
									"remaining": 4999,
									"reset": 1640995200,
									"used": 1,
									"resource": "core"
								},
								"search": {
									"limit": 30,
									"remaining": 29,
									"reset": 1640995260,
									"used": 1,
									"resource": "search"
								},
								"graphql": {
									"limit": 5000,
									"remaining": 4999,
									"reset": 1640995200,
									"used": 1,
									"resource": "graphql"
								}
							},
							"rate": {
								"limit": 5000,
								"remaining": 4999,
								"reset": 1640995200,
								"used": 1,
								"resource": "core"
							}
						}`, nil
					}
					return "", assert.AnError
				}
			},
			expectedLimits: &github.RateLimits{
				Core: &github.Rate{
					Limit:     5000,
					Remaining: 4999,
					Reset:     github.Timestamp{Time: time.Unix(1640995200, 0)},
				},
				Search: &github.Rate{
					Limit:     30,
					Remaining: 29,
					Reset:     github.Timestamp{Time: time.Unix(1640995260, 0)},
				},
				GraphQL: &github.Rate{
					Limit:     5000,
					Remaining: 4999,
					Reset:     github.Timestamp{Time: time.Unix(1640995200, 0)},
				},
			},
		},
		{
			name: "異常系: ghコマンドのエラー",
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "", assert.AnError
				}
			},
			expectedError: "failed to get rate limit",
		},
		{
			name: "異常系: 不正なJSON",
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "invalid json", nil
				}
			},
			expectedError: "failed to parse rate limit response",
		},
		{
			name: "異常系: 必須フィールドの欠落",
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return `{
						"resources": {
							"core": {
								"limit": 5000
							}
						}
					}`, nil
				}
			},
			expectedLimits: &github.RateLimits{
				Core: &github.Rate{
					Limit:     5000,
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Unix(0, 0)},
				},
				Search: &github.Rate{
					Limit:     0,
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Unix(0, 0)},
				},
				GraphQL: &github.Rate{
					Limit:     0,
					Remaining: 0,
					Reset:     github.Timestamp{Time: time.Unix(0, 0)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックエグゼキューターを設定
			mockExec := &MockCommandExecutor{}
			tt.setupMock(mockExec)

			// クライアントを作成
			client, err := NewClient(mockExec)
			require.NoError(t, err)

			// メソッドを実行
			limits, err := client.GetRateLimit(context.Background())

			// エラーの検証
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			// 成功の検証
			assert.NoError(t, err)
			assert.NotNil(t, limits)
			assert.NotNil(t, limits.Core)
			assert.NotNil(t, limits.Search)
			assert.NotNil(t, limits.GraphQL)

			// レート制限の値を検証
			assert.Equal(t, tt.expectedLimits.Core.Limit, limits.Core.Limit)
			assert.Equal(t, tt.expectedLimits.Core.Remaining, limits.Core.Remaining)
			assert.Equal(t, tt.expectedLimits.Core.Reset.Time.Unix(), limits.Core.Reset.Time.Unix())

			assert.Equal(t, tt.expectedLimits.Search.Limit, limits.Search.Limit)
			assert.Equal(t, tt.expectedLimits.Search.Remaining, limits.Search.Remaining)
			assert.Equal(t, tt.expectedLimits.Search.Reset.Time.Unix(), limits.Search.Reset.Time.Unix())

			assert.Equal(t, tt.expectedLimits.GraphQL.Limit, limits.GraphQL.Limit)
			assert.Equal(t, tt.expectedLimits.GraphQL.Remaining, limits.GraphQL.Remaining)
			assert.Equal(t, tt.expectedLimits.GraphQL.Reset.Time.Unix(), limits.GraphQL.Reset.Time.Unix())
		})
	}
}
