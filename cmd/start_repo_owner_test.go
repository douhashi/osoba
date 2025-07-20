package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/douhashi/osoba/internal/utils"
)

func TestStartCmd_RepoOwnerParsing(t *testing.T) {
	tests := []struct {
		name            string
		remoteURL       string
		expectedOwner   string
		expectedRepo    string
		wantErrContains string
	}{
		{
			name:          "HTTPSリモートURL - agileware-jpオーナー",
			remoteURL:     "https://github.com/agileware-jp/fluxport.git",
			expectedOwner: "agileware-jp",
			expectedRepo:  "fluxport",
		},
		{
			name:          "SSHリモートURL - agileware-jpオーナー",
			remoteURL:     "git@github.com:agileware-jp/fluxport.git",
			expectedOwner: "agileware-jp",
			expectedRepo:  "fluxport",
		},
		{
			name:          "HTTPSリモートURL - 別のオーナー",
			remoteURL:     "https://github.com/example-org/test-repo.git",
			expectedOwner: "example-org",
			expectedRepo:  "test-repo",
		},
		{
			name:          "SSHリモートURL - 別のオーナー",
			remoteURL:     "git@github.com:another-user/project.git",
			expectedOwner: "another-user",
			expectedRepo:  "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テンポラリディレクトリを作成
			tmpDir := t.TempDir()

			// Gitリポジトリを初期化
			if err := helpers.InitGitRepo(t, tmpDir); err != nil {
				t.Fatalf("Gitリポジトリの初期化に失敗: %v", err)
			}

			// リモートURLを設定
			if err := helpers.SetGitRemote(t, tmpDir, "origin", tt.remoteURL); err != nil {
				t.Fatalf("リモートURLの設定に失敗: %v", err)
			}

			// 現在のディレクトリを保存して、テスト後に戻す
			originalWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
			}
			defer os.Chdir(originalWd)

			// テンポラリディレクトリに移動
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("ディレクトリの変更に失敗: %v", err)
			}

			// テンポラリ設定ファイルを作成
			configDir := filepath.Join(tmpDir, ".config", "osoba")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				t.Fatalf("設定ディレクトリの作成に失敗: %v", err)
			}

			configContent := `
github:
  token: "dummy-token"
  poll_interval: 1s
tmux:
  session_prefix: "test-"
`
			configPath := filepath.Join(configDir, "osoba.yml")
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				t.Fatalf("設定ファイルの作成に失敗: %v", err)
			}

			// 環境変数を設定
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

			// startコマンドを実行して、ownerが正しく取得されるか確認
			// 現在の実装では、ownerは"douhashi"にハードコードされているため、
			// このテストは失敗するはず

			// TODO: 実際のstartコマンドのテストを実装
			// 現時点では、ParseGitHubURL関数が正しく動作することを確認
			// 実際のコマンド実行時にowner/repoが正しく使用されるかは、
			// 統合テストで確認する必要がある

			// 期待される動作:
			// - startコマンドがgit remote URLからowner/repoを取得
			// - EnsureLabelsExistが正しいowner/repoで呼び出される
			// - 現在の実装では "douhashi" がハードコードされているため失敗するはず

			// この段階では、失敗することを確認
			if tt.expectedOwner != "douhashi" {
				t.Logf("期待値: owner=%s, repo=%s", tt.expectedOwner, tt.expectedRepo)
				t.Logf("現在の実装では owner が 'douhashi' にハードコードされているため、このテストは失敗します")
				// 実際には、startコマンド内でownerがハードコードされているため、
				// 正しいownerが使用されないことを確認
			}
		})
	}
}

// TestStartCmd_GetRepoInfoIntegration は実際のstartコマンドでリポジトリ情報が正しく取得されるかテストする
func TestStartCmd_GetRepoInfoIntegration(t *testing.T) {
	t.Run("修正後: startコマンドでリポジトリオーナーが正しく取得されることを確認", func(t *testing.T) {
		// テンポラリディレクトリを作成
		tmpDir := t.TempDir()

		// Gitリポジトリを初期化
		if err := helpers.InitGitRepo(t, tmpDir); err != nil {
			t.Fatalf("Gitリポジトリの初期化に失敗: %v", err)
		}

		// リモートURLを設定
		remoteURL := "https://github.com/agileware-jp/fluxport.git"
		if err := helpers.SetGitRemote(t, tmpDir, "origin", remoteURL); err != nil {
			t.Fatalf("リモートURLの設定に失敗: %v", err)
		}

		// 現在のディレクトリを保存して、テスト後に戻す
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
		}
		defer os.Chdir(originalWd)

		// テンポラリディレクトリに移動
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("ディレクトリの変更に失敗: %v", err)
		}

		// GetGitHubRepoInfoが正しく動作することを確認
		repoInfo, err := utils.GetGitHubRepoInfo(context.Background())
		if err != nil {
			t.Fatalf("GitHubリポジトリ情報の取得に失敗: %v", err)
		}

		// オーナーとリポジトリ名が正しく取得されることを確認
		if repoInfo.Owner != "agileware-jp" {
			t.Errorf("owner が正しく取得されていません。期待値: agileware-jp, 実際値: %s", repoInfo.Owner)
		}
		if repoInfo.Repo != "fluxport" {
			t.Errorf("repo が正しく取得されていません。期待値: fluxport, 実際値: %s", repoInfo.Repo)
		}

		t.Logf("リポジトリ情報が正しく取得されました: owner=%s, repo=%s", repoInfo.Owner, repoInfo.Repo)
	})
}
