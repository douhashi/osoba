package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/spf13/cobra"
)

func TestStartCmd_ConfigFileCheck(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()

	// 元の環境変数を保存
	origHome := os.Getenv("HOME")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// テスト用の環境変数を設定
	os.Setenv("HOME", tmpDir)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)
	}()

	tests := []struct {
		name               string
		setupFiles         func()
		setupEnv           func()
		setupGitRepo       func(t *testing.T) (string, func())
		args               []string
		wantErr            bool
		wantErrContains    string
		wantOutputContains []string
	}{
		{
			name: "設定ファイルが存在しない場合、エラーメッセージが表示される",
			setupFiles: func() {
				// 設定ファイルを作成しない
			},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				// Gitリポジトリを作成
				gitDir := filepath.Join(tmpDir, "test-repo")
				os.MkdirAll(gitDir, 0755)

				cmd := exec.Command("git", "init")
				cmd.Dir = gitDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = gitDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(gitDir)
				}

				return gitDir, cleanup
			},
			args:            []string{"start", "--foreground"},
			wantErr:         true,
			wantErrContains: "設定ファイルが見つかりません",
			wantOutputContains: []string{
				"エラー: 設定ファイルが見つかりません",
				"osoba init",
			},
		},
		{
			name: "helpフラグの場合は設定ファイルチェックをスキップ",
			setupFiles: func() {
				// 設定ファイルを作成しない
			},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				// Gitリポジトリを作成
				gitDir := filepath.Join(tmpDir, "test-repo")
				os.MkdirAll(gitDir, 0755)

				cmd := exec.Command("git", "init")
				cmd.Dir = gitDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git init failed: %v", err)
				}

				cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
				cmd.Dir = gitDir
				if err := cmd.Run(); err != nil {
					t.Fatalf("git remote add failed: %v", err)
				}

				cleanup := func() {
					os.RemoveAll(gitDir)
				}

				return gitDir, cleanup
			},
			args:    []string{"start", "--help"},
			wantErr: false,
			wantOutputContains: []string{
				"Usage:",
				"osoba start",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クリーンアップ
			os.RemoveAll(filepath.Join(tmpDir, ".config"))
			os.RemoveAll(filepath.Join(tmpDir, "xdg"))
			os.Remove(filepath.Join(tmpDir, ".osoba.yml"))
			os.Remove(filepath.Join(tmpDir, ".osoba.yaml"))

			// テスト環境のセットアップ
			tt.setupFiles()
			tt.setupEnv()

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

			// コマンドを実行
			buf := new(bytes.Buffer)
			errBuf := new(bytes.Buffer)

			rootCmd := newRootCmd()
			rootCmd.AddCommand(newStartCmd())
			rootCmd.SetOut(buf)
			rootCmd.SetErr(errBuf)
			rootCmd.SetArgs(tt.args)

			err = rootCmd.Execute()

			// エラーチェック
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// エラーメッセージのチェック
			if tt.wantErrContains != "" {
				errOutput := errBuf.String()
				if !bytes.Contains([]byte(errOutput), []byte(tt.wantErrContains)) {
					t.Errorf("Execute() error output = %v, want to contain %v", errOutput, tt.wantErrContains)
				}
			}

			// 出力内容のチェック
			output := buf.String() + errBuf.String()
			for _, want := range tt.wantOutputContains {
				if !bytes.Contains([]byte(output), []byte(want)) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}

func TestStartCmd_BackgroundMode_ConfigFileCheck(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()

	// 元の環境変数を保存
	origHome := os.Getenv("HOME")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// テスト用の環境変数を設定
	os.Setenv("HOME", tmpDir)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)
	}()

	// Gitリポジトリを作成
	gitDir := filepath.Join(tmpDir, "test-repo")
	os.MkdirAll(gitDir, 0755)

	cmd := exec.Command("git", "init")
	cmd.Dir = gitDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/douhashi/test-repo.git")
	cmd.Dir = gitDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	// 現在のディレクトリを保存して、テスト後に戻す
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// テスト用ディレクトリに移動
	err = os.Chdir(gitDir)
	if err != nil {
		t.Fatal(err)
	}

	// モックのセットアップ
	mocker := helpers.NewFunctionMocker()
	defer mocker.Restore()

	// バックグラウンドモードのテスト（設定ファイルなし）
	t.Run("バックグラウンドモードで設定ファイルが存在しない場合", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")

		// startInBackgroundをモックして実際のバックグラウンド実行を防ぐ
		backgroundCalled := false
		mocker.MockFunc(&startInBackgroundFunc, func(cmd *cobra.Command, args []string) error {
			backgroundCalled = true
			// 設定ファイルチェックはstartInBackground内で行われる
			return checkConfigFileExists(cmd.OutOrStderr())
		})

		// コマンドを実行
		buf := new(bytes.Buffer)
		errBuf := new(bytes.Buffer)

		rootCmd := newRootCmd()
		rootCmd.AddCommand(newStartCmd())
		rootCmd.SetOut(buf)
		rootCmd.SetErr(errBuf)
		rootCmd.SetArgs([]string{"start"})

		err := rootCmd.Execute()

		// エラーが発生することを確認
		if err == nil {
			t.Error("Expected error when config file not found, but got nil")
		}

		// エラーメッセージを確認
		errOutput := errBuf.String()
		if !bytes.Contains([]byte(errOutput), []byte("設定ファイルが見つかりません")) {
			t.Errorf("Expected error message containing '設定ファイルが見つかりません', got: %v", errOutput)
		}

		// バックグラウンド関数が呼ばれたことを確認
		if !backgroundCalled {
			t.Error("startInBackground was not called")
		}
	})
}
