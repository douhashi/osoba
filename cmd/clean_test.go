package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/spf13/cobra"
)

func TestCleanCmd(t *testing.T) {
	tests := []struct {
		name                     string
		args                     []string
		allFlag                  bool
		forceFlag                bool
		checkTmuxErr             error
		getRepoNameErr           error
		repoName                 string
		sessionExists            bool
		sessionExistsErr         error
		listWindowsErr           error
		windowList               []*tmux.WindowInfo
		killWindowsErr           error
		confirmResponse          string
		listWorktreesErr         error
		worktreeList             []git.WorktreeInfo
		hasUncommittedChangesErr error
		uncommittedChangesMap    map[string]bool
		removeWorktreeErr        error
		expectedOutput           string
		expectedError            string
	}{
		{
			name:          "正常系: 特定のIssue番号のウィンドウとworktreeを削除",
			args:          []string{"83"},
			repoName:      "test-repo",
			sessionExists: true,
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
				{Name: "83-implement"},
				{Name: "83-review"},
			},
			worktreeList: []git.WorktreeInfo{
				{Path: "/repo/.git/osoba/worktrees/issue-83", Branch: "osoba/#83", Commit: "abc123"},
			},
			uncommittedChangesMap: map[string]bool{
				"/repo/.git/osoba/worktrees/issue-83": false,
			},
			expectedOutput: "Issue #83 のリソースを削除しました:\n  ウィンドウ:\n    - 83-plan\n    - 83-implement\n    - 83-review\n  worktree:\n    - /repo/.git/osoba/worktrees/issue-83\n",
		},
		{
			name:           "正常系: 該当するリソースがない場合",
			args:           []string{"999"},
			repoName:       "test-repo",
			sessionExists:  true,
			windowList:     []*tmux.WindowInfo{},
			worktreeList:   []git.WorktreeInfo{},
			expectedOutput: "Issue #999 に関連するリソースが見つかりませんでした。\n",
		},
		{
			name:          "正常系: 未コミット変更がある場合の確認",
			args:          []string{"83"},
			repoName:      "test-repo",
			sessionExists: true,
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
			},
			worktreeList: []git.WorktreeInfo{
				{Path: "/repo/.git/osoba/worktrees/issue-83", Branch: "osoba/#83", Commit: "abc123"},
			},
			uncommittedChangesMap: map[string]bool{
				"/repo/.git/osoba/worktrees/issue-83": true,
			},
			confirmResponse: "yes\n",
			expectedOutput:  "警告: 以下のworktreeに未コミットの変更があります:\n  - /repo/.git/osoba/worktrees/issue-83\nIssue #83 のリソースを削除しました:\n  ウィンドウ:\n    - 83-plan\n  worktree:\n    - /repo/.git/osoba/worktrees/issue-83\n",
		},
		{
			name:          "正常系: 未コミット変更がある場合のキャンセル",
			args:          []string{"83"},
			repoName:      "test-repo",
			sessionExists: true,
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
			},
			worktreeList: []git.WorktreeInfo{
				{Path: "/repo/.git/osoba/worktrees/issue-83", Branch: "osoba/#83", Commit: "abc123"},
			},
			uncommittedChangesMap: map[string]bool{
				"/repo/.git/osoba/worktrees/issue-83": true,
			},
			confirmResponse: "no\n",
			expectedOutput:  "警告: 以下のworktreeに未コミットの変更があります:\n  - /repo/.git/osoba/worktrees/issue-83\n削除をキャンセルしました。\n",
		},
		{
			name:          "正常系: --forceオプションで未コミット変更を無視",
			args:          []string{"83"},
			forceFlag:     true,
			repoName:      "test-repo",
			sessionExists: true,
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
			},
			worktreeList: []git.WorktreeInfo{
				{Path: "/repo/.git/osoba/worktrees/issue-83", Branch: "osoba/#83", Commit: "abc123"},
			},
			uncommittedChangesMap: map[string]bool{
				"/repo/.git/osoba/worktrees/issue-83": true,
			},
			expectedOutput: "警告: 以下のworktreeに未コミットの変更があります:\n  - /repo/.git/osoba/worktrees/issue-83\nIssue #83 のリソースを削除しました:\n  ウィンドウ:\n    - 83-plan\n  worktree:\n    - /repo/.git/osoba/worktrees/issue-83\n",
		},
		{
			name:            "正常系: --allオプションで全ウィンドウを削除（確認でyes）",
			allFlag:         true,
			repoName:        "test-repo",
			sessionExists:   true,
			confirmResponse: "yes\n",
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
				{Name: "83-implement"},
				{Name: "84-review"},
			},
			expectedOutput: "以下のリソースを削除します:\n  ウィンドウ:\n    - 83-plan\n    - 83-implement\n    - 84-review\n以下のリソースを削除しました:\n  ウィンドウ:\n    - 83-plan\n    - 83-implement\n    - 84-review\n",
		},
		{
			name:            "正常系: --allオプションで削除をキャンセル（確認でno）",
			allFlag:         true,
			repoName:        "test-repo",
			sessionExists:   true,
			confirmResponse: "no\n",
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
			},
			expectedOutput: "以下のリソースを削除します:\n  ウィンドウ:\n    - 83-plan\n削除をキャンセルしました。\n",
		},
		{
			name:          "異常系: tmuxがインストールされていない",
			args:          []string{"83"},
			checkTmuxErr:  errors.New("tmux not found"),
			expectedError: "tmux not found",
		},
		{
			name:           "異常系: Gitリポジトリではない",
			args:           []string{"83"},
			getRepoNameErr: errors.New("not a git repository"),
			expectedError:  "not a git repository",
		},
		{
			name:          "異常系: セッションが存在しない",
			args:          []string{"83"},
			repoName:      "test-repo",
			sessionExists: false,
			expectedError: "セッション 'osoba-test-repo' が見つかりません",
		},
		{
			name:             "異常系: セッション確認でエラー",
			args:             []string{"83"},
			repoName:         "test-repo",
			sessionExistsErr: errors.New("tmux error"),
			expectedError:    "セッションの確認に失敗しました: tmux error",
		},
		{
			name:          "異常系: Issue番号が不正",
			args:          []string{"invalid"},
			repoName:      "test-repo",
			sessionExists: true,
			expectedError: "Issue番号は正の整数で指定してください",
		},
		{
			name:          "異常系: 引数が多すぎる",
			args:          []string{"83", "84"},
			expectedError: "引数は1つだけ指定してください",
		},
		{
			name:          "異常系: --allオプションと引数を同時に指定",
			args:          []string{"83"},
			allFlag:       true,
			expectedError: "--all オプションを使用する場合は引数を指定しないでください",
		},
		{
			name:           "異常系: ウィンドウ一覧取得エラー",
			args:           []string{"83"},
			repoName:       "test-repo",
			sessionExists:  true,
			listWindowsErr: errors.New("list windows error"),
			expectedError:  "ウィンドウ一覧の取得に失敗しました: list windows error",
		},
		{
			name:          "異常系: ウィンドウ削除エラー",
			args:          []string{"83"},
			repoName:      "test-repo",
			sessionExists: true,
			windowList: []*tmux.WindowInfo{
				{Name: "83-plan"},
			},
			worktreeList:   []git.WorktreeInfo{},
			killWindowsErr: errors.New("kill windows error"),
			expectedOutput: "Issue #83 のリソースを削除しました:\n  ウィンドウ:\n    - 83-plan\n\n以下のエラーが発生しました:\n  - ウィンドウの削除に失敗しました: kill windows error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 元の関数を保存
			origCheckTmux := checkTmuxInstalledFunc
			origGetRepoName := getRepositoryNameFunc
			origSessionExists := sessionExistsFunc
			origListWindows := listWindowsForIssueFunc
			origListAllWindows := listWindowsByPatternFunc
			origKillWindows := killWindowsForIssueFunc
			origKillWindowsSlice := killWindowsFunc
			origConfirmPrompt := confirmPromptFunc
			origListWorktreesForIssue := listWorktreesForIssueFunc
			origListAllWorktrees := listAllWorktreesFunc
			origHasUncommittedChanges := hasUncommittedChangesFunc
			origRemoveWorktree := removeWorktreeFunc

			// テスト後に復元
			defer func() {
				checkTmuxInstalledFunc = origCheckTmux
				getRepositoryNameFunc = origGetRepoName
				sessionExistsFunc = origSessionExists
				listWindowsForIssueFunc = origListWindows
				listWindowsByPatternFunc = origListAllWindows
				killWindowsForIssueFunc = origKillWindows
				killWindowsFunc = origKillWindowsSlice
				confirmPromptFunc = origConfirmPrompt
				listWorktreesForIssueFunc = origListWorktreesForIssue
				listAllWorktreesFunc = origListAllWorktrees
				hasUncommittedChangesFunc = origHasUncommittedChanges
				removeWorktreeFunc = origRemoveWorktree
			}()

			// モック設定
			checkTmuxInstalledFunc = func() error {
				return tt.checkTmuxErr
			}
			getRepositoryNameFunc = func() (string, error) {
				return tt.repoName, tt.getRepoNameErr
			}
			sessionExistsFunc = func(name string) (bool, error) {
				return tt.sessionExists, tt.sessionExistsErr
			}
			listWindowsForIssueFunc = func(sessionName string, issueNumber int) ([]*tmux.WindowInfo, error) {
				if tt.listWindowsErr != nil {
					return nil, tt.listWindowsErr
				}
				return tt.windowList, nil
			}
			listWindowsByPatternFunc = func(sessionName, pattern string) ([]*tmux.WindowInfo, error) {
				if tt.listWindowsErr != nil {
					return nil, tt.listWindowsErr
				}
				return tt.windowList, nil
			}
			killWindowsForIssueFunc = func(sessionName string, issueNumber int) error {
				return tt.killWindowsErr
			}
			killWindowsFunc = func(sessionName string, windowNames []string) error {
				return tt.killWindowsErr
			}
			confirmPromptFunc = func(prompt string) (bool, error) {
				if tt.confirmResponse == "" {
					return false, nil
				}
				return strings.TrimSpace(tt.confirmResponse) == "yes", nil
			}
			listWorktreesForIssueFunc = func(ctx context.Context, issueNumber int) ([]git.WorktreeInfo, error) {
				if tt.listWorktreesErr != nil {
					return nil, tt.listWorktreesErr
				}
				return tt.worktreeList, nil
			}
			listAllWorktreesFunc = func(ctx context.Context) ([]git.WorktreeInfo, error) {
				if tt.listWorktreesErr != nil {
					return nil, tt.listWorktreesErr
				}
				return tt.worktreeList, nil
			}
			hasUncommittedChangesFunc = func(ctx context.Context, worktreePath string) (bool, error) {
				if tt.hasUncommittedChangesErr != nil {
					return false, tt.hasUncommittedChangesErr
				}
				if tt.uncommittedChangesMap != nil {
					return tt.uncommittedChangesMap[worktreePath], nil
				}
				return false, nil
			}
			removeWorktreeFunc = func(ctx context.Context, worktreePath string) error {
				return tt.removeWorktreeErr
			}

			// コマンド実行
			cmd := newCleanCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			if tt.allFlag {
				cmd.Flags().Set("all", "true")
			}
			if tt.forceFlag {
				cmd.Flags().Set("force", "true")
			}

			err := cmd.Execute()

			// エラーチェック
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, but got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// 出力チェック
			if tt.expectedOutput != "" {
				output := buf.String()
				if output != tt.expectedOutput {
					t.Errorf("expected output:\n%s\nbut got:\n%s", tt.expectedOutput, output)
				}
			}
		})
	}
}

func TestCleanCmd_ValidateArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		allFlag       bool
		expectedError string
	}{
		{
			name: "正常系: Issue番号のみ",
			args: []string{"123"},
		},
		{
			name:    "正常系: --allフラグのみ",
			allFlag: true,
		},
		{
			name:          "異常系: 引数なしで--allフラグなし",
			args:          []string{},
			expectedError: "Issue番号を指定するか、--all オプションを使用してください",
		},
		{
			name:          "異常系: 複数の引数",
			args:          []string{"123", "456"},
			expectedError: "引数は1つだけ指定してください",
		},
		{
			name:          "異常系: --allフラグと引数を同時指定",
			args:          []string{"123"},
			allFlag:       true,
			expectedError: "--all オプションを使用する場合は引数を指定しないでください",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			if tt.allFlag {
				allFlag = true
			} else {
				allFlag = false
			}

			err := validateCleanArgs(cmd, tt.args)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, but got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, but got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseIssueNumber(t *testing.T) {
	tests := []struct {
		name          string
		arg           string
		expectedNum   int
		expectedError bool
	}{
		{
			name:        "正常系: 有効な数値",
			arg:         "123",
			expectedNum: 123,
		},
		{
			name:        "正常系: 1桁の数値",
			arg:         "1",
			expectedNum: 1,
		},
		{
			name:          "異常系: 負の数",
			arg:           "-123",
			expectedError: true,
		},
		{
			name:          "異常系: ゼロ",
			arg:           "0",
			expectedError: true,
		},
		{
			name:          "異常系: 数値以外",
			arg:           "abc",
			expectedError: true,
		},
		{
			name:          "異常系: 空文字列",
			arg:           "",
			expectedError: true,
		},
		{
			name:          "異常系: 小数",
			arg:           "12.3",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, err := parseIssueNumber(tt.arg)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if num != tt.expectedNum {
					t.Errorf("expected %d, but got %d", tt.expectedNum, num)
				}
			}
		})
	}
}

func TestGetWindowNames(t *testing.T) {
	tests := []struct {
		name     string
		windows  []*tmux.WindowInfo
		expected []string
	}{
		{
			name: "正常系: 複数のウィンドウ",
			windows: []*tmux.WindowInfo{
				{Name: "83-plan"},
				{Name: "83-implement"},
				{Name: "83-review"},
			},
			expected: []string{"83-plan", "83-implement", "83-review"},
		},
		{
			name:     "正常系: 空のリスト",
			windows:  []*tmux.WindowInfo{},
			expected: []string{},
		},
		{
			name:     "正常系: nil",
			windows:  nil,
			expected: []string{},
		},
		{
			name: "正常系: 1つのウィンドウ",
			windows: []*tmux.WindowInfo{
				{Name: "42-plan"},
			},
			expected: []string{"42-plan"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWindowNames(tt.windows)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d names, but got %d", len(tt.expected), len(result))
			}

			for i, name := range tt.expected {
				if i >= len(result) || result[i] != name {
					t.Errorf("expected name[%d] = %q, but got %q", i, name, result[i])
				}
			}
		})
	}
}
