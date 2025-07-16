package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	internalGitHub "github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_TransitionIssueLabel(t *testing.T) {
	tests := []struct {
		name          string
		owner         string
		repo          string
		issueNumber   int
		setupMock     func(*MockCommandExecutor)
		expected      bool
		expectedError string
	}{
		{
			name:        "正常系: status:needs-plan から status:planning への遷移",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if command != "gh" {
						return "", fmt.Errorf("unexpected command: %s", command)
					}

					switch callCount {
					case 1:
						// 最初の呼び出し: 現在のラベルを取得
						expectedArgs := []string{"issue", "view", "123", "--repo", "douhashi/osoba", "--json", "labels"}
						if equalStringSlices(args, expectedArgs) {
							return `{
								"labels": [
									{"name": "status:needs-plan", "color": "0075ca"},
									{"name": "bug", "color": "d73a4a"}
								]
							}`, nil
						}
					case 2:
						// 2回目の呼び出し: status:needs-plan を削除
						expectedArgs := []string{"issue", "edit", "123", "--repo", "douhashi/osoba", "--remove-label", "status:needs-plan"}
						if equalStringSlices(args, expectedArgs) {
							return "", nil
						}
					case 3:
						// 3回目の呼び出し: status:planning を追加
						expectedArgs := []string{"issue", "edit", "123", "--repo", "douhashi/osoba", "--add-label", "status:planning"}
						if equalStringSlices(args, expectedArgs) {
							return "", nil
						}
					}
					return "", fmt.Errorf("unexpected call count: %d", callCount)
				}
			},
			expected: true,
		},
		{
			name:        "正常系: status:ready から status:implementing への遷移",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 456,
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					switch callCount {
					case 1:
						return `{
							"labels": [
								{"name": "status:ready", "color": "0e8a16"},
								{"name": "enhancement", "color": "a2eeef"}
							]
						}`, nil
					case 2:
						// remove status:ready
						return "", nil
					case 3:
						// add status:implementing
						return "", nil
					}
					return "", fmt.Errorf("unexpected call count: %d", callCount)
				}
			},
			expected: true,
		},
		{
			name:        "正常系: 既に実行中ラベルがある場合はスキップ",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 789,
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					// 既に status:implementing がある
					return `{
						"labels": [
							{"name": "status:implementing", "color": "28a745"},
							{"name": "bug", "color": "d73a4a"}
						]
					}`, nil
				}
			},
			expected: false,
		},
		{
			name:        "正常系: トリガーラベルがない場合",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 111,
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					// トリガーラベルなし
					return `{
						"labels": [
							{"name": "bug", "color": "d73a4a"},
							{"name": "documentation", "color": "0052cc"}
						]
					}`, nil
				}
			},
			expected: false,
		},
		{
			name:        "異常系: Issue番号が無効",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 0,
			setupMock: func(m *MockCommandExecutor) {
				// 呼ばれないはず
			},
			expectedError: "issue number must be positive",
		},
		{
			name:        "異常系: ownerが空",
			owner:       "",
			repo:        "osoba",
			issueNumber: 123,
			setupMock: func(m *MockCommandExecutor) {
				// 呼ばれないはず
			},
			expectedError: "owner is required",
		},
		{
			name:        "異常系: repoが空",
			owner:       "douhashi",
			repo:        "",
			issueNumber: 123,
			setupMock: func(m *MockCommandExecutor) {
				// 呼ばれないはず
			},
			expectedError: "repo is required",
		},
		{
			name:        "異常系: ラベル取得エラー",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "", fmt.Errorf("failed to get issue")
				}
			},
			expectedError: "failed to get issue labels",
		},
		{
			name:        "異常系: ラベル削除エラー",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					switch callCount {
					case 1:
						// 現在のラベル
						return `{
							"labels": [
								{"name": "status:needs-plan", "color": "0075ca"}
							]
						}`, nil
					case 2:
						// ラベル削除でエラー
						return "", fmt.Errorf("permission denied")
					}
					return "", fmt.Errorf("unexpected call")
				}
			},
			expectedError: "failed to remove label",
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
			transitioned, err := client.TransitionIssueLabel(context.Background(), tt.owner, tt.repo, tt.issueNumber)

			// エラーの検証
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			// 成功の検証
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, transitioned)
		})
	}
}

func TestClient_TransitionIssueLabelWithInfo(t *testing.T) {
	tests := []struct {
		name          string
		owner         string
		repo          string
		issueNumber   int
		setupMock     func(*MockCommandExecutor)
		expected      bool
		expectedInfo  *internalGitHub.TransitionInfo
		expectedError string
	}{
		{
			name:        "正常系: status:needs-plan から status:planning への遷移（情報付き）",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 123,
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					switch callCount {
					case 1:
						// 現在のラベルを取得
						return `{
							"labels": [
								{"name": "status:needs-plan", "color": "0075ca"},
								{"name": "bug", "color": "d73a4a"}
							]
						}`, nil
					case 2:
						// status:needs-plan を削除
						return "", nil
					case 3:
						// status:planning を追加
						return "", nil
					}
					return "", fmt.Errorf("unexpected call count: %d", callCount)
				}
			},
			expected: true,
			expectedInfo: &internalGitHub.TransitionInfo{
				From: "status:needs-plan",
				To:   "status:planning",
			},
		},
		{
			name:        "正常系: トリガーラベルがない場合（情報なし）",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: 456,
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return `{
						"labels": [
							{"name": "bug", "color": "d73a4a"}
						]
					}`, nil
				}
			},
			expected:     false,
			expectedInfo: nil,
		},
		{
			name:        "異常系: Issue番号が無効",
			owner:       "douhashi",
			repo:        "osoba",
			issueNumber: -1,
			setupMock: func(m *MockCommandExecutor) {
				// 呼ばれないはず
			},
			expectedError: "issue number must be positive",
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
			transitioned, info, err := client.TransitionIssueLabelWithInfo(context.Background(), tt.owner, tt.repo, tt.issueNumber)

			// エラーの検証
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			// 成功の検証
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, transitioned)

			if tt.expectedInfo != nil {
				assert.NotNil(t, info)
				assert.Equal(t, tt.expectedInfo.From, info.From)
				assert.Equal(t, tt.expectedInfo.To, info.To)
			} else {
				assert.Nil(t, info)
			}
		})
	}
}

// テスト用のラベル情報構造体
type issueLabelsResponse struct {
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
}

// JSONレスポンスをパースするヘルパー関数
func parseIssueLabels(response string) ([]string, error) {
	var data issueLabelsResponse
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		return nil, err
	}

	labels := make([]string, len(data.Labels))
	for i, label := range data.Labels {
		labels[i] = label.Name
	}
	return labels, nil
}
