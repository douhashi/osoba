package utils

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGetOwnerAndRepoFromGitHubURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantOwner  string
		wantRepo   string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "正常系: HTTPS URL with .git",
			url:       "https://github.com/douhashi/osoba.git",
			wantOwner: "douhashi",
			wantRepo:  "osoba",
			wantErr:   false,
		},
		{
			name:      "正常系: HTTPS URL without .git",
			url:       "https://github.com/douhashi/osoba",
			wantOwner: "douhashi",
			wantRepo:  "osoba",
			wantErr:   false,
		},
		{
			name:      "正常系: SSH URL with .git",
			url:       "git@github.com:douhashi/osoba.git",
			wantOwner: "douhashi",
			wantRepo:  "osoba",
			wantErr:   false,
		},
		{
			name:      "正常系: SSH URL without .git",
			url:       "git@github.com:douhashi/osoba",
			wantOwner: "douhashi",
			wantRepo:  "osoba",
			wantErr:   false,
		},
		{
			name:       "エラー系: 不正なURL",
			url:        "invalid-url",
			wantErr:    true,
			wantErrMsg: "invalid GitHub URL format",
		},
		{
			name:       "エラー系: GitHub以外のURL",
			url:        "https://gitlab.com/user/repo.git",
			wantErr:    true,
			wantErrMsg: "invalid GitHub URL format",
		},
		{
			name:       "エラー系: 空のURL",
			url:        "",
			wantErr:    true,
			wantErrMsg: "invalid GitHub URL format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := GetOwnerAndRepoFromGitHubURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetOwnerAndRepoFromGitHubURL() error = nil, want error")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("GetOwnerAndRepoFromGitHubURL() error = %v, want error containing %v", err, tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("GetOwnerAndRepoFromGitHubURL() error = %v, want nil", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("GetOwnerAndRepoFromGitHubURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("GetOwnerAndRepoFromGitHubURL() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestGetGitHubRepoInfoError(t *testing.T) {
	tests := []struct {
		name     string
		err      *GetGitHubRepoInfoError
		wantMsg  string
		wantStep string
	}{
		{
			name: "working_directory error",
			err: &GetGitHubRepoInfoError{
				Step:    "working_directory",
				Cause:   errors.New("permission denied"),
				Message: "作業ディレクトリの取得に失敗しました",
			},
			wantMsg:  "作業ディレクトリの取得に失敗しました: permission denied",
			wantStep: "working_directory",
		},
		{
			name: "git_directory error",
			err: &GetGitHubRepoInfoError{
				Step:    "git_directory",
				Cause:   errors.New("no .git directory found"),
				Message: "Gitリポジトリが見つかりません",
			},
			wantMsg:  "Gitリポジトリが見つかりません: no .git directory found",
			wantStep: "git_directory",
		},
		{
			name: "remote_url error",
			err: &GetGitHubRepoInfoError{
				Step:    "remote_url",
				Cause:   errors.New("fatal: No such remote 'origin'"),
				Message: "リモートURL取得に失敗しました",
			},
			wantMsg:  "リモートURL取得に失敗しました: fatal: No such remote 'origin'",
			wantStep: "remote_url",
		},
		{
			name: "url_parsing error",
			err: &GetGitHubRepoInfoError{
				Step:    "url_parsing",
				Cause:   errors.New("invalid GitHub URL format"),
				Message: "GitHub URL解析に失敗しました",
			},
			wantMsg:  "GitHub URL解析に失敗しました: invalid GitHub URL format",
			wantStep: "url_parsing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.wantMsg {
				t.Errorf("GetGitHubRepoInfoError.Error() = %v, want %v", tt.err.Error(), tt.wantMsg)
			}

			if tt.err.Step != tt.wantStep {
				t.Errorf("GetGitHubRepoInfoError.Step = %v, want %v", tt.err.Step, tt.wantStep)
			}

			// Unwrapのテスト
			if unwrapped := tt.err.Unwrap(); unwrapped != tt.err.Cause {
				t.Errorf("GetGitHubRepoInfoError.Unwrap() = %v, want %v", unwrapped, tt.err.Cause)
			}
		})
	}
}

func TestFindGitDirectory(t *testing.T) {
	tests := []struct {
		name      string
		startPath string
		want      string
	}{
		// Note: これらのテストは実際のファイルシステムに依存するため、
		// テスト環境でのセットアップが必要です。
		// ここではテストケースの構造のみを定義し、
		// 実装時に適切なモックやテンポラリディレクトリを使用してテストします。
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト実装は実際の修正時に追加
			t.Logf("Testing findGitDirectory with startPath: %s", tt.startPath)
		})
	}
}

// TestGetGitHubRepoInfo_Integration は統合テストの雛形
// 実際のgitリポジトリが必要なため、テスト環境でのセットアップが必要
func TestGetGitHubRepoInfo_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T) string // テスト用のディレクトリを返す
		cleanup func(string)
		wantErr bool
		errStep string
	}{
		// 実際のテストケースは実装時に追加
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.setup != nil {
				dir := tt.setup(t)
				if tt.cleanup != nil {
					defer tt.cleanup(dir)
				}
			}

			_, err := GetGitHubRepoInfo(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetGitHubRepoInfo() error = nil, want error")
					return
				}

				var repoErr *GetGitHubRepoInfoError
				if errors.As(err, &repoErr) && tt.errStep != "" {
					if repoErr.Step != tt.errStep {
						t.Errorf("GetGitHubRepoInfo() error step = %v, want %v", repoErr.Step, tt.errStep)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetGitHubRepoInfo() error = %v, want nil", err)
				}
			}
		})
	}
}
