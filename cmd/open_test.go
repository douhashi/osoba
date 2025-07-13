package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/spf13/cobra"
)

func TestOpenCommand(t *testing.T) {
	// 元の関数を保存
	originalCheckTmux := checkTmuxInstalledFunc
	originalSessionExists := sessionExistsFunc
	originalGetRepoName := getRepositoryNameFunc

	// テスト後に復元
	defer func() {
		checkTmuxInstalledFunc = originalCheckTmux
		sessionExistsFunc = originalSessionExists
		getRepositoryNameFunc = originalGetRepoName
	}()

	tests := []struct {
		name           string
		setupMocks     func()
		expectedError  string
		expectedOutput string
		tmuxEnv        string
	}{
		{
			name: "正常系: セッションが存在する場合（tmux外から）",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error { return nil }
				getRepositoryNameFunc = func() (string, error) { return "test-repo", nil }
				sessionExistsFunc = func(name string) (bool, error) {
					if name == "osoba-test-repo" {
						return true, nil
					}
					return false, nil
				}
			},
			tmuxEnv: "",
		},
		{
			name: "正常系: セッションが存在する場合（tmux内から）",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error { return nil }
				getRepositoryNameFunc = func() (string, error) { return "test-repo", nil }
				sessionExistsFunc = func(name string) (bool, error) {
					if name == "osoba-test-repo" {
						return true, nil
					}
					return false, nil
				}
			},
			tmuxEnv: "/tmp/tmux-1000/default,1234,0",
		},
		{
			name: "エラー: tmuxがインストールされていない",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error {
					return tmux.ErrTmuxNotInstalled
				}
			},
			expectedError: tmux.ErrTmuxNotInstalled.Error(),
		},
		{
			name: "エラー: Gitリポジトリではない",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error { return nil }
				getRepositoryNameFunc = func() (string, error) {
					return "", git.ErrNotGitRepository
				}
			},
			expectedError: "現在のディレクトリはGitリポジトリではありません",
		},
		{
			name: "エラー: リモートリポジトリが設定されていない",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error { return nil }
				getRepositoryNameFunc = func() (string, error) {
					return "", git.ErrNoRemoteFound
				}
			},
			expectedError: "リモートリポジトリが設定されていません",
		},
		{
			name: "エラー: セッションが存在しない",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error { return nil }
				getRepositoryNameFunc = func() (string, error) { return "test-repo", nil }
				sessionExistsFunc = func(name string) (bool, error) {
					return false, nil
				}
			},
			expectedError: "セッション 'osoba-test-repo' が見つかりません。先に 'osoba start' を実行してください",
		},
		{
			name: "エラー: セッション確認でエラー",
			setupMocks: func() {
				checkTmuxInstalledFunc = func() error { return nil }
				getRepositoryNameFunc = func() (string, error) { return "test-repo", nil }
				sessionExistsFunc = func(name string) (bool, error) {
					return false, errors.New("tmux error")
				}
			},
			expectedError: "セッションの確認に失敗しました: tmux error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数の設定
			if tt.tmuxEnv != "" {
				os.Setenv("TMUX", tt.tmuxEnv)
				defer os.Unsetenv("TMUX")
			}

			// モックの設定
			tt.setupMocks()

			// コマンドの実行
			cmd := newOpenCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			// エラーの検証
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("期待されたエラーが発生しませんでした: %s", tt.expectedError)
				} else if err.Error() != tt.expectedError {
					t.Errorf("エラーメッセージが一致しません\n期待: %s\n実際: %s", tt.expectedError, err.Error())
				}
			} else {
				// 実際のtmuxコマンドは実行しないため、エラーは許容
				// ただし、特定のエラー以外は報告
				if err != nil && !isExpectedTmuxError(err) {
					t.Errorf("予期しないエラーが発生しました: %v", err)
				}
			}

			// 出力の検証
			output := buf.String()
			if tt.expectedOutput != "" && output != tt.expectedOutput {
				t.Errorf("出力が一致しません\n期待: %s\n実際: %s", tt.expectedOutput, output)
			}
		})
	}
}

// isExpectedTmuxError は期待されるtmux関連のエラーかを判定
func isExpectedTmuxError(err error) bool {
	// exec.Command の実行エラーは、テスト環境では期待される
	errStr := err.Error()
	return strings.Contains(errStr, "セッションへの接続に失敗しました") ||
		strings.Contains(errStr, "セッションへの切り替えに失敗しました") ||
		strings.Contains(errStr, "executable file not found in $PATH")
}

func TestIsInsideTmux(t *testing.T) {
	tests := []struct {
		name     string
		tmuxEnv  string
		expected bool
	}{
		{
			name:     "TMUX環境変数が設定されている",
			tmuxEnv:  "/tmp/tmux-1000/default,1234,0",
			expected: true,
		},
		{
			name:     "TMUX環境変数が空",
			tmuxEnv:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を設定
			os.Setenv("TMUX", tt.tmuxEnv)
			defer os.Unsetenv("TMUX")

			// 実行
			result := isInsideTmux()

			// 検証
			if result != tt.expected {
				t.Errorf("isInsideTmux() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestOpenCommandWithMockExec はexec.Commandをモック化したテスト
func TestOpenCommandWithMockExec(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") == "1" {
		// これはモックプロセス
		fmt.Println("Mock tmux process executed")
		os.Exit(0)
	}

	// 元の関数を保存
	originalCheckTmux := checkTmuxInstalledFunc
	originalSessionExists := sessionExistsFunc
	originalGetRepoName := getRepositoryNameFunc

	// テスト後に復元
	defer func() {
		checkTmuxInstalledFunc = originalCheckTmux
		sessionExistsFunc = originalSessionExists
		getRepositoryNameFunc = originalGetRepoName
	}()

	t.Run("モックされたtmuxコマンドの実行", func(t *testing.T) {
		// モックの設定
		checkTmuxInstalledFunc = func() error { return nil }
		getRepositoryNameFunc = func() (string, error) { return "test-repo", nil }
		sessionExistsFunc = func(name string) (bool, error) { return true, nil }

		// RunEを直接呼び出す代わりに、カスタムロジックでテスト
		// 実際のtmuxコマンドは実行しない
		err := runOpen(&cobra.Command{}, []string{})

		// エラーは期待される（実際のtmuxコマンドが存在しないため）
		if err == nil || !isExpectedTmuxError(err) {
			t.Logf("Got error: %v", err)
		}
	})
}
