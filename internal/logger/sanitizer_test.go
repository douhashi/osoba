package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "GitHub token",
			input:    "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "ghp_***MASKED***",
		},
		{
			name:     "GitHub app token",
			input:    "ghs_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "ghs_***MASKED***",
		},
		{
			name:     "GitHub user access token",
			input:    "ghu_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "ghu_***MASKED***",
		},
		{
			name:     "GitHub installation token",
			input:    "ghi_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "ghi_***MASKED***",
		},
		{
			name:     "Claude API key",
			input:    "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnopqrstuvwxyz1234-abcdefghij",
			expected: "sk-ant-api03-***MASKED***",
		},
		{
			name:     "API key in Authorization header",
			input:    "Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "Bearer ***MASKED***",
		},
		{
			name:     "API key in token header",
			input:    "token ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "token ***MASKED***",
		},
		{
			name:     "Regular string - not sensitive",
			input:    "regular_string_value",
			expected: "regular_string_value",
		},
		{
			name:     "Integer value",
			input:    42,
			expected: 42,
		},
		{
			name:     "Boolean value",
			input:    true,
			expected: true,
		},
		{
			name:     "Nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Password field",
			input:    "my_secret_password_123",
			expected: "my_secret_password_123", // SanitizeValue doesn't handle field names
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeKeyValue(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         interface{}
		expectedKey   string
		expectedValue interface{}
	}{
		{
			name:          "GitHub token in Authorization header",
			key:           "Authorization",
			value:         "Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expectedKey:   "Authorization",
			expectedValue: "Bearer ***MASKED***",
		},
		{
			name:          "Password field",
			key:           "password",
			value:         "secret123",
			expectedKey:   "password",
			expectedValue: "***MASKED***",
		},
		{
			name:          "Token field",
			key:           "token",
			value:         "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expectedKey:   "token",
			expectedValue: "***MASKED***",
		},
		{
			name:          "API key field",
			key:           "api_key",
			value:         "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyz1234",
			expectedKey:   "api_key",
			expectedValue: "***MASKED***",
		},
		{
			name:          "Secret field",
			key:           "client_secret",
			value:         "secret_value_123",
			expectedKey:   "client_secret",
			expectedValue: "***MASKED***",
		},
		{
			name:          "Case insensitive key matching",
			key:           "GITHUB_TOKEN",
			value:         "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expectedKey:   "GITHUB_TOKEN",
			expectedValue: "***MASKED***",
		},
		{
			name:          "Regular field - not sensitive",
			key:           "user_name",
			value:         "john_doe",
			expectedKey:   "user_name",
			expectedValue: "john_doe",
		},
		{
			name:          "Non-string value in sensitive field",
			key:           "password",
			value:         123,
			expectedKey:   "password",
			expectedValue: "***MASKED***",
		},
		{
			name:          "GitHub token value with non-sensitive key",
			key:           "some_field",
			value:         "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expectedKey:   "some_field",
			expectedValue: "ghp_***MASKED***", // Value-based masking
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultKey, resultValue := SanitizeKeyValue(tt.key, tt.value)
			assert.Equal(t, tt.expectedKey, resultKey)
			assert.Equal(t, tt.expectedValue, resultValue)
		})
	}
}

func TestSanitizeArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    []interface{}
		expected []interface{}
	}{
		{
			name: "Key-value pairs with sensitive data",
			input: []interface{}{
				"user_id", 123,
				"token", "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
				"action", "merge",
			},
			expected: []interface{}{
				"user_id", 123,
				"token", "***MASKED***",
				"action", "merge",
			},
		},
		{
			name: "Odd number of arguments (malformed key-value pairs)",
			input: []interface{}{
				"user_id", 123,
				"incomplete_key",
			},
			expected: []interface{}{
				"user_id", 123,
				"incomplete_key",
			},
		},
		{
			name: "Mixed sensitive and non-sensitive data",
			input: []interface{}{
				"issue_number", 216,
				"github_token", "ghp_abcdef123456",
				"pr_number", 45,
				"api_key", "sk-ant-api03-test123",
			},
			expected: []interface{}{
				"issue_number", 216,
				"github_token", "***MASKED***",
				"pr_number", 45,
				"api_key", "***MASKED***",
			},
		},
		{
			name:     "Empty arguments",
			input:    []interface{}{},
			expected: []interface{}{},
		},
		{
			name: "Non-string keys",
			input: []interface{}{
				123, "value1",
				"token", "ghp_secret123",
			},
			expected: []interface{}{
				123, "value1",
				"token", "***MASKED***",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeArgs(tt.input...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		// Sensitive keys
		{name: "password", key: "password", expected: true},
		{name: "PASSWORD (uppercase)", key: "PASSWORD", expected: true},
		{name: "Password (mixed case)", key: "Password", expected: true},
		{name: "token", key: "token", expected: true},
		{name: "api_key", key: "api_key", expected: true},
		{name: "secret", key: "secret", expected: true},
		{name: "github_token", key: "github_token", expected: true},
		{name: "claude_api_key", key: "claude_api_key", expected: true},
		{name: "authorization", key: "authorization", expected: true},
		{name: "auth", key: "auth", expected: true},
		{name: "credential", key: "credential", expected: true},
		{name: "private_key", key: "private_key", expected: true},

		// Non-sensitive keys
		{name: "user_name", key: "user_name", expected: false},
		{name: "issue_number", key: "issue_number", expected: false},
		{name: "pr_number", key: "pr_number", expected: false},
		{name: "action", key: "action", expected: false},
		{name: "status", key: "status", expected: false},
		{name: "message", key: "message", expected: false},
		{name: "tokening", key: "tokening", expected: false}, // Contains "token" but not exact match
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		// GitHub tokens
		{name: "GitHub personal token", value: "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234", expected: true},
		{name: "GitHub app token", value: "ghs_1234567890abcdefghijklmnopqrstuvwxyz1234", expected: true},
		{name: "GitHub user token", value: "ghu_1234567890abcdefghijklmnopqrstuvwxyz1234", expected: true},
		{name: "GitHub installation token", value: "ghi_1234567890abcdefghijklmnopqrstuvwxyz1234", expected: true},

		// Claude API keys
		{name: "Claude API key", value: "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdefghijklmnopqrstuvwxyz1234-abcdefghij", expected: true},

		// Authorization headers
		{name: "Bearer token", value: "Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz1234", expected: true},
		{name: "Token header", value: "token ghp_1234567890abcdefghijklmnopqrstuvwxyz1234", expected: true},

		// Non-sensitive values
		{name: "Regular string", value: "regular_string", expected: false},
		{name: "Empty string", value: "", expected: false},
		{name: "Integer", value: 123, expected: false},
		{name: "Boolean", value: true, expected: false},
		{name: "Nil", value: nil, expected: false},
		{name: "Short string", value: "abc", expected: false},
		{name: "GitHub-like but invalid", value: "ghp_short", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSensitiveValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "GitHub token with prefix preservation",
			value:    "ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "ghp_***MASKED***",
		},
		{
			name:     "Claude API key with prefix preservation",
			value:    "sk-ant-api03-1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "sk-ant-api03-***MASKED***",
		},
		{
			name:     "Bearer token with prefix preservation",
			value:    "Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "Bearer ***MASKED***",
		},
		{
			name:     "Token header with prefix preservation",
			value:    "token ghp_1234567890abcdefghijklmnopqrstuvwxyz1234",
			expected: "token ***MASKED***",
		},
		{
			name:     "Non-string value",
			value:    123,
			expected: "***MASKED***",
		},
		{
			name:     "Empty string",
			value:    "",
			expected: "***MASKED***",
		},
		{
			name:     "Nil value",
			value:    nil,
			expected: "***MASKED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
