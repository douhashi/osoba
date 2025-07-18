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

// CreateWindowForIssueWithNewWindowDetection はIssue番号に基づいてウィンドウを作成し、新規作成かどうかを返す
//
// 戻り値:
//   - string: ウィンドウ名（"issue-{番号}"形式）
//   - bool: 新規作成された場合はtrue、既存の場合はfalse
//   - error: エラーが発生した場合
func CreateWindowForIssueWithNewWindowDetection(sessionName string, issueNumber int, executor CommandExecutor) (string, bool, error) {
	if sessionName == "" {
		return "", false, fmt.Errorf("session name cannot be empty")
	}
	if issueNumber <= 0 {
		return "", false, fmt.Errorf("issue number must be positive")
	}

	windowName := GetWindowNameForIssue(issueNumber)

	if logger := GetLogger(); logger != nil {
		logger.Info("Issue用ウィンドウ作成開始（新規判定付き）",
			"operation", "create_window_for_issue_with_detection",
			"session_name", sessionName,
			"issue_number", issueNumber,
			"window_name", windowName)
	}

	// ウィンドウが既に存在するかチェック
	exists, err := WindowExistsWithExecutor(sessionName, windowName, executor)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("ウィンドウ存在チェック失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"error", err)
		}
		return "", false, fmt.Errorf("failed to check window existence for issue %d: %w", issueNumber, err)
	}

	if exists {
		if logger := GetLogger(); logger != nil {
			logger.Info("既存ウィンドウを使用",
				"session_name", sessionName,
				"window_name", windowName,
				"is_new_window", false)
		}
		return windowName, false, nil
	}

	// 新規ウィンドウを作成
	err = CreateWindowWithExecutor(sessionName, windowName, executor)
	if err != nil {
		if logger := GetLogger(); logger != nil {
			logger.Error("新規ウィンドウ作成失敗",
				"session_name", sessionName,
				"window_name", windowName,
				"error", err)
		}
		return "", false, fmt.Errorf("failed to create window for issue %d: %w", issueNumber, err)
	}

	if logger := GetLogger(); logger != nil {
		logger.Info("新規ウィンドウ作成完了",
			"session_name", sessionName,
			"window_name", windowName,
			"is_new_window", true)
	}

	return windowName, true, nil
}
