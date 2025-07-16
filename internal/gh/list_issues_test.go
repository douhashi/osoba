package gh

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListIssuesByLabels(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		labels         []string
		setupMock      func(*MockCommandExecutor)
		expectedIssues []*github.Issue
		expectedError  string
	}{
		{
			name:   "単一ラベルでのIssue取得",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"bug"},
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					expectedCmd := []string{"issue", "list", "--repo", "douhashi/osoba", "--label", "bug", "--state", "open", "--json", "number,title,state,url,body,createdAt,updatedAt,author,labels"}
					if command == "gh" && equalStringSlices(args, expectedCmd) {
						return `[
							{
								"number": 1,
								"title": "Bug: Something is broken",
								"state": "OPEN",
								"url": "https://github.com/douhashi/osoba/issues/1",
								"body": "This is a bug report",
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-02T00:00:00Z",
								"author": {"login": "user1"},
								"labels": [{"name": "bug", "description": "Something isn't working", "color": "d73a4a"}]
							}
						]`, nil
					}
					return "", assert.AnError
				}
			},
			expectedIssues: []*github.Issue{
				{
					Number:    github.Int(1),
					Title:     github.String("Bug: Something is broken"),
					State:     github.String("open"),
					HTMLURL:   github.String("https://github.com/douhashi/osoba/issues/1"),
					Body:      github.String("This is a bug report"),
					CreatedAt: &github.Timestamp{Time: mustParseTime("2024-01-01T00:00:00Z")},
					UpdatedAt: &github.Timestamp{Time: mustParseTime("2024-01-02T00:00:00Z")},
					User: &github.User{
						Login: github.String("user1"),
					},
					Labels: []*github.Label{
						{
							Name:        github.String("bug"),
							Description: github.String("Something isn't working"),
							Color:       github.String("d73a4a"),
						},
					},
				},
			},
		},
		{
			name:   "複数ラベルでのIssue取得（OR条件）",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"bug", "enhancement"},
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if command != "gh" {
						return "", assert.AnError
					}

					// 最初の呼び出し: bugラベル
					if callCount == 1 {
						expectedCmd := []string{"issue", "list", "--repo", "douhashi/osoba", "--label", "bug", "--state", "open", "--json", "number,title,state,url,body,createdAt,updatedAt,author,labels"}
						if equalStringSlices(args, expectedCmd) {
							return `[
								{
									"number": 1,
									"title": "Bug: Something is broken",
									"state": "OPEN",
									"url": "https://github.com/douhashi/osoba/issues/1",
									"body": "This is a bug report",
									"createdAt": "2024-01-01T00:00:00Z",
									"updatedAt": "2024-01-02T00:00:00Z",
									"author": {"login": "user1"},
									"labels": [{"name": "bug", "description": "Something isn't working", "color": "d73a4a"}]
								}
							]`, nil
						}
					}

					// 2番目の呼び出し: enhancementラベル
					if callCount == 2 {
						expectedCmd := []string{"issue", "list", "--repo", "douhashi/osoba", "--label", "enhancement", "--state", "open", "--json", "number,title,state,url,body,createdAt,updatedAt,author,labels"}
						if equalStringSlices(args, expectedCmd) {
							return `[
								{
									"number": 2,
									"title": "Feature: Add new feature",
									"state": "OPEN",
									"url": "https://github.com/douhashi/osoba/issues/2",
									"body": "This is a feature request",
									"createdAt": "2024-01-03T00:00:00Z",
									"updatedAt": "2024-01-04T00:00:00Z",
									"author": {"login": "user2"},
									"labels": [{"name": "enhancement", "description": "New feature or request", "color": "a2eeef"}]
								}
							]`, nil
						}
					}

					return "", assert.AnError
				}
			},
			expectedIssues: []*github.Issue{
				{
					Number:    github.Int(1),
					Title:     github.String("Bug: Something is broken"),
					State:     github.String("open"),
					HTMLURL:   github.String("https://github.com/douhashi/osoba/issues/1"),
					Body:      github.String("This is a bug report"),
					CreatedAt: &github.Timestamp{Time: mustParseTime("2024-01-01T00:00:00Z")},
					UpdatedAt: &github.Timestamp{Time: mustParseTime("2024-01-02T00:00:00Z")},
					User: &github.User{
						Login: github.String("user1"),
					},
					Labels: []*github.Label{
						{
							Name:        github.String("bug"),
							Description: github.String("Something isn't working"),
							Color:       github.String("d73a4a"),
						},
					},
				},
				{
					Number:    github.Int(2),
					Title:     github.String("Feature: Add new feature"),
					State:     github.String("open"),
					HTMLURL:   github.String("https://github.com/douhashi/osoba/issues/2"),
					Body:      github.String("This is a feature request"),
					CreatedAt: &github.Timestamp{Time: mustParseTime("2024-01-03T00:00:00Z")},
					UpdatedAt: &github.Timestamp{Time: mustParseTime("2024-01-04T00:00:00Z")},
					User: &github.User{
						Login: github.String("user2"),
					},
					Labels: []*github.Label{
						{
							Name:        github.String("enhancement"),
							Description: github.String("New feature or request"),
							Color:       github.String("a2eeef"),
						},
					},
				},
			},
		},
		{
			name:   "CLOSEDステートの正規化",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"bug"},
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					expectedCmd := []string{"issue", "list", "--repo", "douhashi/osoba", "--label", "bug", "--state", "open", "--json", "number,title,state,url,body,createdAt,updatedAt,author,labels"}
					if command == "gh" && equalStringSlices(args, expectedCmd) {
						return `[
							{
								"number": 3,
								"title": "Fixed bug",
								"state": "CLOSED",
								"url": "https://github.com/douhashi/osoba/issues/3",
								"body": "This was fixed",
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-02T00:00:00Z",
								"author": {"login": "user1"},
								"labels": [{"name": "bug", "description": "Something isn't working", "color": "d73a4a"}]
							}
						]`, nil
					}
					return "", assert.AnError
				}
			},
			expectedIssues: []*github.Issue{
				{
					Number:    github.Int(3),
					Title:     github.String("Fixed bug"),
					State:     github.String("closed"),
					HTMLURL:   github.String("https://github.com/douhashi/osoba/issues/3"),
					Body:      github.String("This was fixed"),
					CreatedAt: &github.Timestamp{Time: mustParseTime("2024-01-01T00:00:00Z")},
					UpdatedAt: &github.Timestamp{Time: mustParseTime("2024-01-02T00:00:00Z")},
					User: &github.User{
						Login: github.String("user1"),
					},
					Labels: []*github.Label{
						{
							Name:        github.String("bug"),
							Description: github.String("Something isn't working"),
							Color:       github.String("d73a4a"),
						},
					},
				},
			},
		},
		{
			name:          "空のラベルリスト",
			owner:         "douhashi",
			repo:          "osoba",
			labels:        []string{},
			setupMock:     func(m *MockCommandExecutor) {},
			expectedError: "at least one label is required",
		},
		{
			name:   "ghコマンドのエラー（エラーが発生しても空の結果を返す）",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"bug"},
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "", assert.AnError
				}
			},
			expectedIssues: []*github.Issue{}, // エラーが発生しても空の結果を返す
		},
		{
			name:   "不正なJSON出力（パースエラーが発生しても空の結果を返す）",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"bug"},
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "invalid json", nil
				}
			},
			expectedIssues: []*github.Issue{}, // パースエラーが発生しても空の結果を返す
		},
		{
			name:   "Issue無し",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"nonexistent"},
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "[]", nil
				}
			},
			expectedIssues: []*github.Issue{},
		},
		{
			name:   "重複するIssueの排除",
			owner:  "douhashi",
			repo:   "osoba",
			labels: []string{"bug", "high-priority"},
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if command != "gh" {
						return "", assert.AnError
					}

					// 両方のラベルで同じIssue #1を返す
					issueData := `[
						{
							"number": 1,
							"title": "Critical Bug",
							"state": "OPEN",
							"url": "https://github.com/douhashi/osoba/issues/1",
							"body": "This is a critical bug",
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-02T00:00:00Z",
							"author": {"login": "user1"},
							"labels": [
								{"name": "bug", "description": "Something isn't working", "color": "d73a4a"},
								{"name": "high-priority", "description": "High priority issue", "color": "ff0000"}
							]
						}
					]`

					if callCount == 1 || callCount == 2 {
						return issueData, nil
					}

					return "", assert.AnError
				}
			},
			expectedIssues: []*github.Issue{
				{
					Number:    github.Int(1),
					Title:     github.String("Critical Bug"),
					State:     github.String("open"),
					HTMLURL:   github.String("https://github.com/douhashi/osoba/issues/1"),
					Body:      github.String("This is a critical bug"),
					CreatedAt: &github.Timestamp{Time: mustParseTime("2024-01-01T00:00:00Z")},
					UpdatedAt: &github.Timestamp{Time: mustParseTime("2024-01-02T00:00:00Z")},
					User: &github.User{
						Login: github.String("user1"),
					},
					Labels: []*github.Label{
						{
							Name:        github.String("bug"),
							Description: github.String("Something isn't working"),
							Color:       github.String("d73a4a"),
						},
						{
							Name:        github.String("high-priority"),
							Description: github.String("High priority issue"),
							Color:       github.String("ff0000"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックエグゼキューターを設定
			mockExec := &MockCommandExecutor{}
			tt.setupMock(mockExec)

			// クライアントを作成
			client, err := NewClient(mockExec)
			require.NoError(t, err)

			// メソッドを実行
			issues, err := client.ListIssuesByLabels(context.Background(), tt.owner, tt.repo, tt.labels)

			// エラーの検証
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			// 成功の検証
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expectedIssues), len(issues))

			for i, expected := range tt.expectedIssues {
				actual := issues[i]
				assert.Equal(t, *expected.Number, *actual.Number)
				assert.Equal(t, *expected.Title, *actual.Title)
				assert.Equal(t, *expected.State, *actual.State)
				assert.Equal(t, *expected.HTMLURL, *actual.HTMLURL)
				assert.Equal(t, *expected.Body, *actual.Body)
				assert.Equal(t, expected.CreatedAt.Time, actual.CreatedAt.Time)
				assert.Equal(t, expected.UpdatedAt.Time, actual.UpdatedAt.Time)
				assert.Equal(t, *expected.User.Login, *actual.User.Login)

				assert.Equal(t, len(expected.Labels), len(actual.Labels))
				for j, expectedLabel := range expected.Labels {
					actualLabel := actual.Labels[j]
					assert.Equal(t, *expectedLabel.Name, *actualLabel.Name)
					assert.Equal(t, *expectedLabel.Description, *actualLabel.Description)
					assert.Equal(t, *expectedLabel.Color, *actualLabel.Color)
				}
			}
		})
	}
}

// equalStringSlices は2つの文字列スライスが等しいかを比較する
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
