package logger

import (
	"regexp"
	"strings"
)

// センシティブなキーのパターン（大文字小文字を区別しない）
var sensitiveKeyPatterns = []string{
	"password",
	"token",
	"api_key",
	"apikey",
	"secret",
	"github_token",
	"claude_api_key",
	"authorization",
	"auth",
	"credential",
	"private_key",
	"access_token",
	"refresh_token",
	"client_secret",
}

// センシティブな値のパターン（正規表現）
var sensitiveValuePatterns = []*regexp.Regexp{
	// GitHub personal access tokens (ghp_ + 36文字)
	regexp.MustCompile(`^ghp_[A-Za-z0-9]{36,}$`),
	// GitHub app tokens (ghs_ + 36文字)
	regexp.MustCompile(`^ghs_[A-Za-z0-9]{36,}$`),
	// GitHub user access tokens (ghu_ + 36文字)
	regexp.MustCompile(`^ghu_[A-Za-z0-9]{36,}$`),
	// GitHub installation tokens (ghi_ + 36文字)
	regexp.MustCompile(`^ghi_[A-Za-z0-9]{36,}$`),
	// Claude API keys (実際のパターンに合わせる)
	regexp.MustCompile(`^sk-ant-api03-[A-Za-z0-9\-_]{20,}$`),
	// Authorization Bearer tokens (大文字小文字を区別しない)
	regexp.MustCompile(`(?i)^Bearer\s+[A-Za-z0-9\-_\.]{20,}$`),
	// Token headers (大文字小文字を区別しない)
	regexp.MustCompile(`(?i)^token\s+[A-Za-z0-9\-_\.]{20,}$`),
}

// SanitizeValue は値がセンシティブかどうかを判定し、必要に応じてマスクする
func SanitizeValue(value interface{}) interface{} {
	if isSensitiveValue(value) {
		return maskValue(value)
	}
	return value
}

// SanitizeKeyValue はキーと値の組み合わせをチェックし、センシティブな情報をマスクする
func SanitizeKeyValue(key string, value interface{}) (string, interface{}) {
	// キーがセンシティブな場合は値をマスク（プレフィックスを保持する場合もある）
	if isSensitiveKey(key) {
		// Authorization や token のようなキーの場合、プレフィックスを保持
		if strings.ToLower(key) == "authorization" && isSensitiveValue(value) {
			return key, maskValue(value)
		}
		return key, "***MASKED***"
	}

	// 値がセンシティブな場合のみマスク
	if isSensitiveValue(value) {
		return key, maskValue(value)
	}

	return key, value
}

// SanitizeArgs はログ引数（key-valueペア）をサニタイズする
func SanitizeArgs(args ...interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}

	sanitized := make([]interface{}, len(args))
	copy(sanitized, args)

	// key-valueペアを処理（偶数インデックスがkey、奇数インデックスがvalue）
	for i := 0; i < len(sanitized)-1; i += 2 {
		if key, ok := sanitized[i].(string); ok {
			_, sanitizedValue := SanitizeKeyValue(key, sanitized[i+1])
			sanitized[i+1] = sanitizedValue
		}
	}

	return sanitized
}

// isSensitiveKey はキーがセンシティブかどうかを判定する
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)

	for _, pattern := range sensitiveKeyPatterns {
		// 完全一致または単語境界での一致をチェック
		if lowerKey == pattern ||
			strings.HasPrefix(lowerKey, pattern+"_") ||
			strings.HasSuffix(lowerKey, "_"+pattern) ||
			strings.Contains(lowerKey, "_"+pattern+"_") {
			return true
		}
	}

	return false
}

// isSensitiveValue は値がセンシティブかどうかを判定する
func isSensitiveValue(value interface{}) bool {
	// 文字列以外の値はセンシティブではないと判定
	str, ok := value.(string)
	if !ok || str == "" {
		return false
	}

	// 各パターンでチェック
	for _, pattern := range sensitiveValuePatterns {
		if pattern.MatchString(str) {
			return true
		}
	}

	return false
}

// maskValue はセンシティブな値をマスクする（プレフィックスを保持）
func maskValue(value interface{}) string {
	str, ok := value.(string)
	if !ok {
		return "***MASKED***"
	}

	if str == "" {
		return "***MASKED***"
	}

	// GitHub トークンのプレフィックスを保持
	if strings.HasPrefix(str, "ghp_") {
		return "ghp_***MASKED***"
	}
	if strings.HasPrefix(str, "ghs_") {
		return "ghs_***MASKED***"
	}
	if strings.HasPrefix(str, "ghu_") {
		return "ghu_***MASKED***"
	}
	if strings.HasPrefix(str, "ghi_") {
		return "ghi_***MASKED***"
	}

	// Claude API keyのプレフィックスを保持
	if strings.HasPrefix(str, "sk-ant-api03-") {
		return "sk-ant-api03-***MASKED***"
	}

	// Authorization headerのプレフィックスを保持
	if strings.HasPrefix(str, "Bearer ") {
		return "Bearer ***MASKED***"
	}
	if strings.HasPrefix(str, "token ") {
		return "token ***MASKED***"
	}

	return "***MASKED***"
}
