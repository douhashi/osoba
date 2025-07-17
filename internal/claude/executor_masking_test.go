package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitiveData(t *testing.T) {
	executor := &DefaultClaudeExecutor{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHubトークンのマスキング - ghp_形式",
			input:    "Token: ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "Token: [GITHUB_TOKEN]",
		},
		{
			name:     "GitHubトークンのマスキング - github_pat_形式",
			input:    "PAT: github_pat_11AAAAAAA_abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNO",
			expected: "PAT: [GITHUB_TOKEN]",
		},
		{
			name:     "GitHubトークンのマスキング - ghs_形式",
			input:    "Secret: ghs_1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "Secret: [GITHUB_TOKEN]",
		},
		{
			name:     "APIキーのマスキング - sk-proj形式",
			input:    "API Key: sk-proj-1234567890abcdefghijklmnopqrstuvwxyz1234567890AB",
			expected: "API Key: [API_KEY]",
		},
		{
			name:     "一般的なAPIキーのマスキング - api_key=形式",
			input:    "Request with api_key=abcdefghijklmnopqrst1234567890",
			expected: "Request with api_key=[MASKED]",
		},
		{
			name:     "一般的なAPIキーのマスキング - apiKey:形式",
			input:    "Config: apiKey: \"my-super-secret-api-key-123456\"",
			expected: "Config: apiKey: [MASKED]",
		},
		{
			name:     "Bearerトークンのマスキング",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: Bearer [TOKEN]",
		},
		{
			name:     "複数のトークンを含む場合",
			input:    "Test with token: ghp_1234567890abcdefghijklmnopqrstuvwxyz and key: sk-proj-abcd1234567890abcdefghijklmnopqrstuvwxyz123456",
			expected: "Test with token: [GITHUB_TOKEN] and key: [API_KEY]",
		},
		{
			name:     "機密情報を含まない場合",
			input:    "This is a normal prompt without any sensitive data",
			expected: "This is a normal prompt without any sensitive data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := executor.maskSensitiveData(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
