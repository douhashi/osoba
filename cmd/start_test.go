package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

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
				// runWatchWithFlagsをモック
				origRunWatch := runWatchWithFlagsFunc
				runWatchWithFlagsFunc = func(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
					// Issue監視モードが呼ばれたことを出力で確認
					cmd.OutOrStdout().Write([]byte("Issue監視モードを開始します\n"))
					return nil
				}

				t.Cleanup(func() {
					runWatchWithFlagsFunc = origRunWatch
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
			name: "異常系: Gitリポジトリではない",
			setupMock: func(t *testing.T) {
				// 特にモックは不要
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				cleanup := func() {
					os.RemoveAll(tmpDir)
				}
				return tmpDir, cleanup
			},
			wantErr:         true,
			wantErrContains: "現在のディレクトリはGitリポジトリではありません",
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
