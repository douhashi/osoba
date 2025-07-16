package tmux

import (
	"fmt"
	"testing"
)

// TestListWindowsByPattern はパターンに一致するウィンドウのリストを取得するテスト
func TestListWindowsByPattern(t *testing.T) {
	tests := []struct {
		name          string
		sessionName   string
		pattern       string
		windowList    string
		executorError error
		expectedNames []string
		expectedError bool
	}{
		{
			name:        "正常系: Issue番号パターンに一致するウィンドウを取得",
			sessionName: "test-session",
			pattern:     `^\d+-\w+$`,
			windowList: `0:main:0:1
1:83-plan:0:1
2:83-implement:1:1
3:83-review:0:1
4:other-window:0:1`,
			expectedNames: []string{"83-plan", "83-implement", "83-review"},
			expectedError: false,
		},
		{
			name:        "正常系: 特定のIssue番号のウィンドウを取得",
			sessionName: "test-session",
			pattern:     `^83-`,
			windowList: `0:main:0:1
1:83-plan:0:1
2:83-implement:1:1
3:83-review:0:1
4:84-plan:0:1`,
			expectedNames: []string{"83-plan", "83-implement", "83-review"},
			expectedError: false,
		},
		{
			name:        "正常系: パターンに一致するウィンドウがない",
			sessionName: "test-session",
			pattern:     `^999-`,
			windowList: `0:main:0:1
1:83-plan:0:1
2:83-implement:1:1`,
			expectedNames: []string{},
			expectedError: false,
		},
		{
			name:          "異常系: セッション名が空",
			sessionName:   "",
			pattern:       `^\d+-\w+$`,
			expectedNames: nil,
			expectedError: true,
		},
		{
			name:          "異常系: パターンが空",
			sessionName:   "test-session",
			pattern:       "",
			expectedNames: nil,
			expectedError: true,
		},
		{
			name:          "異常系: 無効な正規表現パターン",
			sessionName:   "test-session",
			pattern:       `[`,
			expectedNames: nil,
			expectedError: true,
		},
		{
			name:          "異常系: tmuxコマンドのエラー",
			sessionName:   "test-session",
			pattern:       `^\d+-\w+$`,
			executorError: fmt.Errorf("session not found"),
			expectedNames: nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &MockCommandExecutor{}
			executor.On("Execute", "tmux", "list-windows", "-t", tt.sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}").
				Return(tt.windowList, tt.executorError)

			windows, err := ListWindowsByPatternWithExecutor(tt.sessionName, tt.pattern, executor)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if len(windows) != len(tt.expectedNames) {
					t.Errorf("expected %d windows, got %d", len(tt.expectedNames), len(windows))
				}

				windowNames := make([]string, len(windows))
				for i, w := range windows {
					windowNames[i] = w.Name
				}

				for _, expected := range tt.expectedNames {
					found := false
					for _, actual := range windowNames {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected window '%s' not found", expected)
					}
				}
			}
		})
	}
}

// TestListWindowsForIssue は特定のIssue番号に関連するウィンドウのリストを取得するテスト
func TestListWindowsForIssue(t *testing.T) {
	tests := []struct {
		name          string
		sessionName   string
		issueNumber   int
		windowList    string
		executorError error
		expectedNames []string
		expectedError bool
	}{
		{
			name:        "正常系: Issue番号83のウィンドウを取得",
			sessionName: "test-session",
			issueNumber: 83,
			windowList: `0:main:0:1
1:83-plan:0:1
2:83-implement:1:1
3:83-review:0:1
4:84-plan:0:1`,
			expectedNames: []string{"83-plan", "83-implement", "83-review"},
			expectedError: false,
		},
		{
			name:        "正常系: 該当するウィンドウがない",
			sessionName: "test-session",
			issueNumber: 999,
			windowList: `0:main:0:1
1:83-plan:0:1
2:83-implement:1:1`,
			expectedNames: []string{},
			expectedError: false,
		},
		{
			name:          "異常系: セッション名が空",
			sessionName:   "",
			issueNumber:   83,
			expectedNames: nil,
			expectedError: true,
		},
		{
			name:          "異常系: Issue番号が0以下",
			sessionName:   "test-session",
			issueNumber:   0,
			expectedNames: nil,
			expectedError: true,
		},
		{
			name:          "異常系: tmuxコマンドのエラー",
			sessionName:   "test-session",
			issueNumber:   83,
			executorError: fmt.Errorf("session not found"),
			expectedNames: nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &MockCommandExecutor{}
			executor.On("Execute", "tmux", "list-windows", "-t", tt.sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}").
				Return(tt.windowList, tt.executorError)

			windows, err := ListWindowsForIssueWithExecutor(tt.sessionName, tt.issueNumber, executor)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if len(windows) != len(tt.expectedNames) {
					t.Errorf("expected %d windows, got %d", len(tt.expectedNames), len(windows))
				}

				windowNames := make([]string, len(windows))
				for i, w := range windows {
					windowNames[i] = w.Name
				}

				for _, expected := range tt.expectedNames {
					found := false
					for _, actual := range windowNames {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected window '%s' not found", expected)
					}
				}
			}
		})
	}
}

// TestKillWindows は複数のウィンドウを一括削除するテスト
func TestKillWindows(t *testing.T) {
	tests := []struct {
		name          string
		sessionName   string
		windowNames   []string
		killErrors    []error // 各ウィンドウに対するエラー
		expectedError bool
	}{
		{
			name:          "正常系: 複数ウィンドウの削除",
			sessionName:   "test-session",
			windowNames:   []string{"83-plan", "83-implement", "83-review"},
			killErrors:    []error{nil, nil, nil},
			expectedError: false,
		},
		{
			name:          "正常系: 空のリスト",
			sessionName:   "test-session",
			windowNames:   []string{},
			expectedError: false,
		},
		{
			name:          "異常系: セッション名が空",
			sessionName:   "",
			windowNames:   []string{"83-plan"},
			expectedError: true,
		},
		{
			name:          "異常系: ウィンドウ名に空文字列が含まれる",
			sessionName:   "test-session",
			windowNames:   []string{"83-plan", "", "83-review"},
			killErrors:    []error{nil},
			expectedError: true,
		},
		{
			name:          "異常系: 一部のウィンドウ削除に失敗",
			sessionName:   "test-session",
			windowNames:   []string{"83-plan", "83-implement", "83-review"},
			killErrors:    []error{nil, fmt.Errorf("window not found"), nil},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &MockCommandExecutor{}

			// 各ウィンドウに対するkill-windowのモック設定
			killIdx := 0
			for _, windowName := range tt.windowNames {
				if windowName != "" {
					var err error
					if killIdx < len(tt.killErrors) {
						err = tt.killErrors[killIdx]
					}
					executor.On("Execute", "tmux", "kill-window", "-t", fmt.Sprintf("%s:%s", tt.sessionName, windowName)).
						Return("", err)
					killIdx++
				}
			}

			err := KillWindowsWithExecutor(tt.sessionName, tt.windowNames, executor)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// モックの呼び出し確認
			if tt.sessionName != "" {
				executor.AssertExpectations(t)
			}
		})
	}
}

// TestKillWindowsForIssue は特定のIssue番号に関連するウィンドウを一括削除するテスト
func TestKillWindowsForIssue(t *testing.T) {
	tests := []struct {
		name          string
		sessionName   string
		issueNumber   int
		windowList    string
		listError     error
		killErrors    []error
		expectedError bool
		expectedKills []string
	}{
		{
			name:        "正常系: Issue番号83のウィンドウを削除",
			sessionName: "test-session",
			issueNumber: 83,
			windowList: `0:main:0:1
1:83-plan:0:1
2:83-implement:1:1
3:83-review:0:1
4:84-plan:0:1`,
			expectedKills: []string{"83-plan", "83-implement", "83-review"},
			killErrors:    []error{nil, nil, nil},
			expectedError: false,
		},
		{
			name:        "正常系: 該当するウィンドウがない",
			sessionName: "test-session",
			issueNumber: 999,
			windowList: `0:main:0:1
1:83-plan:0:1`,
			expectedKills: []string{},
			expectedError: false,
		},
		{
			name:          "異常系: セッション名が空",
			sessionName:   "",
			issueNumber:   83,
			expectedKills: []string{},
			expectedError: true,
		},
		{
			name:          "異常系: Issue番号が0以下",
			sessionName:   "test-session",
			issueNumber:   0,
			expectedKills: []string{},
			expectedError: true,
		},
		{
			name:          "異常系: list-windowsでエラー",
			sessionName:   "test-session",
			issueNumber:   83,
			listError:     fmt.Errorf("session not found"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &MockCommandExecutor{}

			// list-windowsのモック設定
			if tt.sessionName != "" && tt.issueNumber > 0 {
				executor.On("Execute", "tmux", "list-windows", "-t", tt.sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}").
					Return(tt.windowList, tt.listError)
			}

			// kill-windowのモック設定
			for i, windowName := range tt.expectedKills {
				var err error
				if i < len(tt.killErrors) {
					err = tt.killErrors[i]
				}
				executor.On("Execute", "tmux", "kill-window", "-t", fmt.Sprintf("%s:%s", tt.sessionName, windowName)).
					Return("", err)
			}

			err := KillWindowsForIssueWithExecutor(tt.sessionName, tt.issueNumber, executor)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// モックの呼び出し確認
			if tt.sessionName != "" && tt.issueNumber > 0 {
				executor.AssertExpectations(t)
			}
		})
	}
}
