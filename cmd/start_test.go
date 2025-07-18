package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/spf13/cobra"
)

func TestStartCmd(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		setup              func(t *testing.T) (string, func())
		wantErr            bool
		wantOutputContains []string
		wantErrContains    string
	}{
		{
			name:    "正常系: startコマンドヘルプ",
			args:    []string{"start", "--help"},
			wantErr: false,
			wantOutputContains: []string{
				"start",
				"GitHub Issueの監視を開始",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			rootCmd = newRootCmd()
			rootCmd.AddCommand(newStartCmd())
			rootCmd.SetOut(buf)
			rootCmd.SetErr(errBuf)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			errOutput := errBuf.String()

			for _, want := range tt.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}

			if tt.wantErrContains != "" && !strings.Contains(errOutput, tt.wantErrContains) {
				t.Errorf("Execute() error output = %v, want to contain %v", errOutput, tt.wantErrContains)
			}
		})
	}
}

// 実際の機能をテストするユニットテスト
func TestStartCmdExecution(t *testing.T) {
	tests := []struct {
		name            string
		setupMock       func(t *testing.T)
		cleanupMock     func()
		setupGitRepo    func(t *testing.T) (string, func())
		setupEnv        func() func()
		wantErr         bool
		wantContains    []string
		wantErrContains string
	}{
		{
			name: "正常系: デフォルトでIssue監視モードが開始される",
			setupMock: func(t *testing.T) {
				// FunctionMockerを使用してモック
				mocker := helpers.NewFunctionMocker()
				t.Cleanup(mocker.Restore)

				// テスト環境であることを設定
				os.Setenv("GO_TEST", "1")
				t.Cleanup(func() { os.Unsetenv("GO_TEST") })

				// デフォルトではバックグラウンド実行になるため、
				// フォアグラウンドで実行するようにモック
				mocker.MockFunc(&runWatchWithFlagsFunc, func(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
					// Issue監視モードが呼ばれたことを出力で確認
					cmd.OutOrStdout().Write([]byte("Issue監視モードを開始します\n"))
					return nil
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				// git initとremote設定を実行
				cmd := exec.Command("git", "init")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			setupEnv: func() func() {
				// GitHub Tokenを設定
				os.Setenv("GITHUB_TOKEN", "test-token")
				return func() {
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			wantErr: false,
			wantContains: []string{
				"Issue監視モードを開始します",
			},
		},
		{
			name: "正常系: デフォルト設定ファイルが自動読み込みされる",
			setupMock: func(t *testing.T) {
				// FunctionMockerを使用してモック
				mocker := helpers.NewFunctionMocker()
				t.Cleanup(mocker.Restore)

				mocker.MockFunc(&runWatchWithFlagsFunc, func(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
					// 設定ファイルが自動読み込みされることを確認
					cmd.OutOrStdout().Write([]byte("Issue監視モードを開始します\n"))
					// configFlagが空でも動作することを確認
					if configFlag == "" {
						cmd.OutOrStdout().Write([]byte("デフォルト設定ファイル読み込み成功\n"))
					}
					return nil
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				// git initとremote設定を実行
				cmd := exec.Command("git", "init")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			setupEnv: func() func() {
				// GitHub Tokenを設定
				os.Setenv("GITHUB_TOKEN", "test-token")
				return func() {
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			wantErr: false,
			wantContains: []string{
				"Issue監視モードを開始します",
				"デフォルト設定ファイル読み込み成功",
			},
		},
		{
			name: "正常系: -cフラグで指定された設定ファイルが優先される",
			setupMock: func(t *testing.T) {
				// FunctionMockerを使用してモック
				mocker := helpers.NewFunctionMocker()
				t.Cleanup(mocker.Restore)

				mocker.MockFunc(&runWatchWithFlagsFunc, func(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
					cmd.OutOrStdout().Write([]byte("Issue監視モードを開始します\n"))
					// 指定された設定ファイルが使用されることを確認
					if configFlag == "custom.yml" {
						cmd.OutOrStdout().Write([]byte("設定ファイル: custom.yml\n"))
					}
					return nil
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				// git initとremote設定を実行
				cmd := exec.Command("git", "init")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			setupEnv: func() func() {
				// GitHub Tokenを設定
				os.Setenv("GITHUB_TOKEN", "test-token")
				return func() {
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			wantErr: false,
			wantContains: []string{
				"Issue監視モードを開始します",
				"設定ファイル: custom.yml",
			},
		},
		{
			name: "異常系: Gitリポジトリではない",
			setupMock: func(t *testing.T) {
				// FunctionMockerを使用してモック
				mocker := helpers.NewFunctionMocker()
				t.Cleanup(mocker.Restore)

				mocker.MockFunc(&runWatchWithFlagsFunc, func(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
					// リポジトリ情報を取得（ここでエラーになることを期待）
					_, err := git.GetRepositoryName()
					if err != nil {
						return fmt.Errorf("リポジトリ名の取得に失敗: %w", err)
					}

					return nil
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				cleanup := func() {
					os.RemoveAll(tmpDir)
				}
				return tmpDir, cleanup
			},
			setupEnv: func() func() {
				// トークンは設定しない（Gitリポジトリチェックのみをテスト）
				return func() {}
			},
			wantErr:         true,
			wantErrContains: "リポジトリ名の取得に失敗",
		},
		{
			name: "正常系: バックグラウンド実行時のフィードバック表示",
			setupMock: func(t *testing.T) {
				mocker := helpers.NewFunctionMocker()
				t.Cleanup(mocker.Restore)

				// バックグラウンド実行時の動作をモック
				mocker.MockFunc(&startInBackgroundFunc, func(cmd *cobra.Command, args []string) error {
					// 期待されるフィードバックメッセージを出力
					fmt.Fprintln(cmd.OutOrStdout(), "Issue監視を開始しています...")
					fmt.Fprintln(cmd.OutOrStdout(), "リポジトリ: douhashi/test-repo")
					fmt.Fprintln(cmd.OutOrStdout(), "ポーリング間隔: 5s")
					fmt.Fprintln(cmd.OutOrStdout(), "バックグラウンドで起動しました")
					fmt.Fprintln(cmd.OutOrStdout(), "PID: 12345")
					return nil
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				cmd := exec.Command("git", "init")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			setupEnv: func() func() {
				os.Setenv("GITHUB_TOKEN", "test-token")
				return func() {
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			wantErr: false,
			wantContains: []string{
				"Issue監視を開始しています...",
				"リポジトリ: douhashi/test-repo",
				"ポーリング間隔: 5s",
				"バックグラウンドで起動しました",
				"PID: 12345",
			},
		},
		{
			name: "正常系: 既に実行中の場合の詳細メッセージ",
			setupMock: func(t *testing.T) {
				mocker := helpers.NewFunctionMocker()
				t.Cleanup(mocker.Restore)

				// 既に実行中の場合をモック
				mocker.MockFunc(&startInBackgroundFunc, func(cmd *cobra.Command, args []string) error {
					fmt.Fprintln(cmd.OutOrStdout(), "Issue監視を開始しています...")
					fmt.Fprintln(cmd.OutOrStdout(), "リポジトリ: douhashi/test-repo")
					return fmt.Errorf("既に実行中です (PID: 9999)")
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()

				cmd := exec.Command("git", "init")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = tmpDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			setupEnv: func() func() {
				os.Setenv("GITHUB_TOKEN", "test-token")
				return func() {
					os.Unsetenv("GITHUB_TOKEN")
				}
			},
			wantErr: true,
			wantContains: []string{
				"Issue監視を開始しています...",
				"リポジトリ: douhashi/test-repo",
			},
			wantErrContains: "既に実行中です (PID: 9999)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Gitリポジトリのセットアップ
			dir, cleanup := tt.setupGitRepo(t)
			defer cleanup()

			// 現在のディレクトリを保存して、テスト後に戻す
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			// テスト用ディレクトリに移動
			err = os.Chdir(dir)
			if err != nil {
				t.Fatal(err)
			}

			// モックのセットアップ（ディレクトリ移動後に実行）
			if tt.setupMock != nil {
				tt.setupMock(t)
			}

			// 環境変数のセットアップ
			if tt.setupEnv != nil {
				cleanup := tt.setupEnv()
				defer cleanup()
			}

			// コマンドを実行
			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			cmd := newStartCmd()
			cmd.SetOut(buf)
			cmd.SetErr(errBuf)

			// テストケースに応じてコマンド引数を決定
			var args []string

			// バックグラウンド実行テストケースかチェック
			isBackgroundTest := strings.Contains(tt.name, "バックグラウンド実行") || strings.Contains(tt.name, "既に実行中")

			if !isBackgroundTest {
				// 従来のテストはフォアグラウンド実行
				args = []string{"--foreground"}
			}

			// -cフラグが必要なテストケースの判定
			if tt.name == "正常系: -cフラグで指定された設定ファイルが優先される" {
				args = append(args, "-c", "custom.yml")
			}

			cmd.SetArgs(args)

			err = cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErrContains != "" && err != nil {
				// エラーメッセージを確認
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Execute() error = %v, want to contain %v", err, tt.wantErrContains)
				}
			}

			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}
