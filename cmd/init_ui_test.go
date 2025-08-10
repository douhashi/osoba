package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestInitCmd_ProgressDisplay(t *testing.T) {
	// モック関数を保存しておく
	origIsGitRepo := isGitRepositoryFunc
	origCheckCommand := checkCommandFunc
	origGetEnv := getEnvFunc
	origExecCommand := execCommandFunc
	origMkdirAll := mkdirAllFunc
	origWriteFile := writeFileFunc
	origGetRemoteURL := getRemoteURLFunc
	origGitHubClient := createGitHubClientFunc
	defer func() {
		isGitRepositoryFunc = origIsGitRepo
		checkCommandFunc = origCheckCommand
		getEnvFunc = origGetEnv
		execCommandFunc = origExecCommand
		mkdirAllFunc = origMkdirAll
		writeFileFunc = origWriteFile
		getRemoteURLFunc = origGetRemoteURL
		createGitHubClientFunc = origGitHubClient
	}()

	// 基本的なモックを設定
	isGitRepositoryFunc = func(path string) (bool, error) {
		return true, nil
	}
	checkCommandFunc = func(cmd string) error {
		return nil
	}
	execCommandFunc = func(name string, args ...string) ([]byte, error) {
		if name == "gh" {
			return []byte("success"), nil
		}
		return []byte{}, nil
	}
	mkdirAllFunc = func(path string, perm os.FileMode) error {
		return nil
	}
	writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
		return nil
	}
	getRemoteURLFunc = func(remoteName string) (string, error) {
		return "https://github.com/douhashi/osoba.git", nil
	}
	getEnvFunc = func(key string) string {
		if key == "GITHUB_TOKEN" || key == "OSOBA_GITHUB_TOKEN" {
			return "test-token"
		}
		return ""
	}
	mockClient := &mockInitGitHubClient{
		ensureLabelsFunc: func(ctx context.Context, owner, repo string) error {
			return nil
		},
	}
	createGitHubClientFunc = func(token string) githubInterface {
		return mockClient
	}

	tests := []struct {
		name               string
		wantOutputContains []string
	}{
		{
			name: "正常系: 進行状況表示とチェックマークが表示される",
			wantOutputContains: []string{
				"🚀 osobaの初期化を開始します",
				"[1/8] Gitリポジトリの確認",
				"[2/9] 必要なツールの確認",
				"[3/9] GitHub CLI (gh)の確認",
				"[4/9] GitHub認証の確認",
				"[5/9] GitHubリポジトリへのアクセス確認",
				"[6/9] 設定ファイルの作成",
				"[7/9] Claude commandsの配置",
				"[8/9] ドキュメントシステムの配置",
				"[9/9] GitHubラベルの作成",
				"✅ 初期化が完了しました！",
				"次のステップ:",
				"1. osoba start",
				"2. osoba open",
				"3. GitHubでIssueを作成し",
				"status:needs-plan",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd = newRootCmd()
			rootCmd.AddCommand(newInitCmd())
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"init"})

			err := rootCmd.Execute()

			if err != nil {
				t.Errorf("Execute() error = %v, want nil", err)
				return
			}

			output := buf.String()
			for _, want := range tt.wantOutputContains {
				if !strings.Contains(output, want) {
					t.Errorf("Execute() output = %v, want to contain %v", output, want)
				}
			}
		})
	}
}
