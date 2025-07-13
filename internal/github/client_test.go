package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v50/github"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "正常系: 有効なトークンでクライアントを作成できる",
			token:   "test-token",
			wantErr: false,
		},
		{
			name:    "異常系: 空のトークンでエラーになる",
			token:   "",
			wantErr: true,
			errMsg:  "GitHub token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("NewClient() error = %v, want %v", err.Error(), tt.errMsg)
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestClient_GetRepository(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		owner    string
		repo     string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "異常系: ownerが空でエラーになる",
			owner:   "",
			repo:    "osoba",
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "owner is required"
			},
		},
		{
			name:    "異常系: repoが空でエラーになる",
			owner:   "douhashi",
			repo:    "",
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "repo is required"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// バリデーションエラーのテスト
			client := &Client{
				github: github.NewClient(nil),
			}

			repo, err := client.GetRepository(ctx, tt.owner, tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("GetRepository() error = %v, want specific error", err)
			}
			if !tt.wantErr && repo == nil && tt.owner != "" && tt.repo != "" {
				// 実際のAPIを呼ばないため、正常系でもnilが返ることを許容
				t.Skip("Skipping actual API call test")
			}
		})
	}
}

func TestClient_ListIssuesByLabels(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		owner    string
		repo     string
		labels   []string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "異常系: ownerが空でエラーになる",
			owner:   "",
			repo:    "osoba",
			labels:  []string{"status:needs-plan"},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "owner is required"
			},
		},
		{
			name:    "異常系: repoが空でエラーになる",
			owner:   "douhashi",
			repo:    "",
			labels:  []string{"status:needs-plan"},
			wantErr: true,
			errCheck: func(err error) bool {
				return err.Error() == "repo is required"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// バリデーションエラーのテスト
			client := &Client{
				github: github.NewClient(nil),
			}

			_, err := client.ListIssuesByLabels(ctx, tt.owner, tt.repo, tt.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListIssuesByLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errCheck != nil && !tt.errCheck(err) {
				t.Errorf("ListIssuesByLabels() error = %v, want specific error", err)
			}
		})
	}
}
