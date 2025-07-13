package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/tmux"
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
				"tmuxセッションを作成",
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
		name         string
		setupMock    func(t *testing.T)
		cleanupMock  func()
		setupGitRepo func(t *testing.T) (string, func())
		wantErr      bool
		wantContains []string
		wantErrType  error
	}{
		{
			name: "正常系: 新規セッション作成",
			setupMock: func(t *testing.T) {
				// tmuxがインストールされている
				origCheckTmux := checkTmuxInstalled
				checkTmuxInstalled = func() error {
					return nil
				}

				// セッションが存在しない
				origSessionExists := sessionExists
				sessionExists = func(name string) (bool, error) {
					return false, nil
				}

				// セッション作成成功
				origCreateSession := createSession
				createSession = func(name string) error {
					return nil
				}

				t.Cleanup(func() {
					checkTmuxInstalled = origCheckTmux
					sessionExists = origSessionExists
					createSession = origCreateSession
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				configDir := filepath.Join(gitDir, "config")

				err := os.MkdirAll(filepath.Dir(configDir), 0755)
				if err != nil {
					t.Fatal(err)
				}

				configContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/douhashi/test-repo.git
`
				err = os.WriteFile(configDir, []byte(configContent), 0644)
				if err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			wantErr: false,
			wantContains: []string{
				"tmuxセッション 'osoba-test-repo' を作成しました",
				"tmux attach -t osoba-test-repo",
			},
		},
		{
			name: "正常系: 既存セッションがある場合",
			setupMock: func(t *testing.T) {
				// tmuxがインストールされている
				origCheckTmux := checkTmuxInstalled
				checkTmuxInstalled = func() error {
					return nil
				}

				// セッションが既に存在する
				origSessionExists := sessionExists
				sessionExists = func(name string) (bool, error) {
					return true, nil
				}

				t.Cleanup(func() {
					checkTmuxInstalled = origCheckTmux
					sessionExists = origSessionExists
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				configDir := filepath.Join(gitDir, "config")

				err := os.MkdirAll(filepath.Dir(configDir), 0755)
				if err != nil {
					t.Fatal(err)
				}

				configContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/douhashi/test-repo.git
`
				err = os.WriteFile(configDir, []byte(configContent), 0644)
				if err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			wantErr: false,
			wantContains: []string{
				"tmuxセッション 'osoba-test-repo' は既に存在します",
				"tmux attach -t osoba-test-repo",
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
			wantErr:     true,
			wantErrType: git.ErrNotGitRepository,
		},
		{
			name: "異常系: tmuxがインストールされていない",
			setupMock: func(t *testing.T) {
				// tmuxがインストールされていない
				origCheckTmux := checkTmuxInstalled
				checkTmuxInstalled = func() error {
					return tmux.ErrTmuxNotInstalled
				}

				t.Cleanup(func() {
					checkTmuxInstalled = origCheckTmux
				})
			},
			setupGitRepo: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				configDir := filepath.Join(gitDir, "config")

				err := os.MkdirAll(filepath.Dir(configDir), 0755)
				if err != nil {
					t.Fatal(err)
				}

				configContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/douhashi/test-repo.git
`
				err = os.WriteFile(configDir, []byte(configContent), 0644)
				if err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			wantErr:     true,
			wantErrType: tmux.ErrTmuxNotInstalled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			if tt.setupMock != nil {
				tt.setupMock(t)
			}

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

			cmd := newStartCmd()
			cmd.SetOut(buf)
			cmd.SetErr(errBuf)

			err = cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErrType != nil && err != nil {
				// エラーの型を確認
				if !strings.Contains(err.Error(), tt.wantErrType.Error()) {
					t.Errorf("Execute() error = %v, wantErrType %v", err, tt.wantErrType)
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
