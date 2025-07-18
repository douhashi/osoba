package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// GetWindowNameForIssue はIssue番号からウィンドウ名を生成する（フェーズを含まない）
func GetWindowNameForIssue(issueNumber int) string {
	return fmt.Sprintf("issue-%d", issueNumber)
}

// ParseWindowNameForIssue はウィンドウ名からIssue番号を抽出する（フェーズを含まない形式）
func ParseWindowNameForIssue(windowName string) (int, error) {
	// "issue-123" 形式からIssue番号を抽出
	if !strings.HasPrefix(windowName, "issue-") {
		return 0, fmt.Errorf("invalid window name format: %s", windowName)
	}

	issueStr := strings.TrimPrefix(windowName, "issue-")
	issueNumber, err := strconv.Atoi(issueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid issue number in window name: %s", windowName)
	}

	return issueNumber, nil
}

// IsNewFormatIssueWindow はウィンドウ名が新形式のIssue用かどうかを判定する
func IsNewFormatIssueWindow(windowName string) bool {
	return strings.HasPrefix(windowName, "issue-")
}
