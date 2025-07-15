package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateIssueComment(t *testing.T) {
	tests := []struct {
		name        string
		owner       string
		repo        string
		issueNumber int
		comment     string
		setupServer func() *httptest.Server
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "正常なコメント投稿",
			owner:       "test-owner",
			repo:        "test-repo",
			issueNumber: 123,
			comment:     "osoba: 計画を作成します",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// リクエストの検証
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "/repos/test-owner/test-repo/issues/123/comments", r.URL.Path)

					// レスポンスを返す
					w.WriteHeader(http.StatusCreated)
					fmt.Fprintf(w, `{
						"id": 1,
						"body": "osoba: 計画を作成します",
						"user": {
							"login": "test-user"
						}
					}`)
				}))
			},
			wantErr: false,
		},
		{
			name:        "ownerが空の場合",
			owner:       "",
			repo:        "test-repo",
			issueNumber: 123,
			comment:     "test comment",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("APIが呼ばれてはいけない")
				}))
			},
			wantErr: true,
			errMsg:  "owner is required",
		},
		{
			name:        "repoが空の場合",
			owner:       "test-owner",
			repo:        "",
			issueNumber: 123,
			comment:     "test comment",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("APIが呼ばれてはいけない")
				}))
			},
			wantErr: true,
			errMsg:  "repo is required",
		},
		{
			name:        "issueNumberが無効な場合",
			owner:       "test-owner",
			repo:        "test-repo",
			issueNumber: 0,
			comment:     "test comment",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("APIが呼ばれてはいけない")
				}))
			},
			wantErr: true,
			errMsg:  "issue number must be positive",
		},
		{
			name:        "commentが空の場合",
			owner:       "test-owner",
			repo:        "test-repo",
			issueNumber: 123,
			comment:     "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					t.Fatal("APIが呼ばれてはいけない")
				}))
			},
			wantErr: true,
			errMsg:  "comment is required",
		},
		{
			name:        "GitHub APIがエラーを返す場合",
			owner:       "test-owner",
			repo:        "test-repo",
			issueNumber: 123,
			comment:     "test comment",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{
						"message": "Internal Server Error",
						"documentation_url": "https://docs.github.com/rest"
					}`)
				}))
			},
			wantErr: true,
			errMsg:  "500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストサーバーのセットアップ
			server := tt.setupServer()
			defer server.Close()

			// GitHub クライアントの作成（テストサーバーを使用）
			httpClient := &http.Client{}
			ghClient := github.NewClient(httpClient)
			ghClient.BaseURL, _ = ghClient.BaseURL.Parse(server.URL + "/")

			// テスト対象のクライアントを作成
			client := &Client{
				github: ghClient,
			}

			// テスト実行
			err := client.CreateIssueComment(context.Background(), tt.owner, tt.repo, tt.issueNumber, tt.comment)

			// アサーション
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
