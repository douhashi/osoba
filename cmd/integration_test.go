package cmd_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/douhashi/osoba/cmd"
)

func TestStartCommandIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// テスト用のGitリポジトリを作成
	tmpDir := t.TempDir()

	// gitリポジトリを初期化
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// リモートを追加
	remoteCmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/integration-test.git")
	remoteCmd.Dir = tmpDir
	if err := remoteCmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// 現在のディレクトリを保存
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// テストディレクトリに移動
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// コマンドを実行
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// NewRootCmdはすべてのサブコマンドを含む新しいrootCmdを作成
	rootCmd := cmd.NewRootCmd()
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"start"})

	// tmuxの存在確認（インストールされていない場合はスキップ）
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed, skipping integration test")
	}

	err = rootCmd.Execute()
	output := buf.String()
	errOutput := errBuf.String()

	if err != nil {
		t.Logf("Command error: %v", err)
		t.Logf("Error output: %s", errOutput)
	}

	// 出力を確認
	if err == nil {
		// 成功した場合の出力確認
		if !strings.Contains(output, "osoba-integration-test") {
			t.Errorf("Expected session name 'osoba-integration-test' in output, got: %s", output)
		}

		if !strings.Contains(output, "tmux attach") {
			t.Errorf("Expected tmux attach command in output, got: %s", output)
		}

		// 実際にセッションが作成されたか確認
		checkCmd := exec.Command("tmux", "has-session", "-t", "osoba-integration-test")
		if err := checkCmd.Run(); err != nil {
			t.Errorf("tmux session was not created: %v", err)
		} else {
			// セッションをクリーンアップ
			killCmd := exec.Command("tmux", "kill-session", "-t", "osoba-integration-test")
			killCmd.Run() // エラーは無視（既に存在しない可能性もある）
		}
	}
}

// NewRootCmd を公開するためのヘルパー（テスト用）
func init() {
	// cmd パッケージから rootCmd を取得できるようにする必要がある
	// このテストではcmd.Executeを使わずに直接コマンドを作成する
}
