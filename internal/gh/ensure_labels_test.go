package gh

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_EnsureLabelsExist(t *testing.T) {
	// 必要なラベル定義
	requiredLabels := map[string]struct {
		color       string
		description string
	}{
		"status:needs-plan":       {"0075ca", "Planning phase required"},
		"status:ready":            {"0e8a16", "Ready for implementation"},
		"status:review-requested": {"d93f0b", "Review requested"},
		"status:planning":         {"1d76db", "Currently in planning phase"},
		"status:implementing":     {"28a745", "Currently being implemented"},
		"status:reviewing":        {"e99695", "Currently under review"},
		"status:lgtm":             {"0e8a16", "Approved"},
		"status:requires-changes": {"fbca04", "Changes requested"},
		"status:revising":         {"f29513", "Currently addressing review feedback"},
	}

	tests := []struct {
		name          string
		owner         string
		repo          string
		setupMock     func(*MockCommandExecutor)
		expectedError string
	}{
		{
			name:  "正常系: 全ラベルが既に存在",
			owner: "douhashi",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if callCount == 1 {
						// 最初の呼び出し: ラベル一覧を取得
						expectedArgs := []string{"label", "list", "--repo", "douhashi/osoba", "--json", "name,color,description", "--limit", "100"}
						if command == "gh" && equalStringSlices(args, expectedArgs) {
							return `[
								{"name": "status:needs-plan", "color": "0075ca", "description": "Planning phase required"},
								{"name": "status:ready", "color": "0e8a16", "description": "Ready for implementation"},
								{"name": "status:review-requested", "color": "d93f0b", "description": "Review requested"},
								{"name": "status:planning", "color": "1d76db", "description": "Currently in planning phase"},
								{"name": "status:implementing", "color": "28a745", "description": "Currently being implemented"},
								{"name": "status:reviewing", "color": "e99695", "description": "Currently under review"},
								{"name": "status:lgtm", "color": "0e8a16", "description": "Approved"},
								{"name": "status:requires-changes", "color": "fbca04", "description": "Changes requested"},
								{"name": "status:revising", "color": "f29513", "description": "Currently addressing review feedback"},
								{"name": "bug", "color": "d73a4a", "description": "Something isn't working"}
							]`, nil
						}
					}
					return "", fmt.Errorf("unexpected call count: %d", callCount)
				}
			},
		},
		{
			name:  "正常系: 一部ラベルが存在しない",
			owner: "douhashi",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				createdLabels := make(map[string]bool)
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if callCount == 1 {
						// 最初の呼び出し: ラベル一覧を取得（一部のみ存在）
						return `[
							{"name": "status:needs-plan", "color": "0075ca", "description": "Planning phase required"},
							{"name": "status:ready", "color": "0e8a16", "description": "Ready for implementation"},
							{"name": "bug", "color": "d73a4a", "description": "Something isn't working"}
						]`, nil
					} else {
						// ラベル作成の呼び出し
						if len(args) >= 9 && args[0] == "label" && args[1] == "create" {
							labelName := args[2]
							// 必要なラベルかチェック
							if labelDef, ok := requiredLabels[labelName]; ok {
								// 引数の検証: --repo, --color, --description
								if args[3] == "--repo" && args[4] == "douhashi/osoba" &&
									args[5] == "--color" && args[6] == labelDef.color &&
									args[7] == "--description" && args[8] == labelDef.description {
									createdLabels[labelName] = true
									return "", nil
								}
							}
						}
					}
					return "", fmt.Errorf("unexpected call count: %d, args: %v", callCount, args)
				}
			},
		},
		{
			name:  "正常系: 全ラベルが存在しない",
			owner: "douhashi",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if callCount == 1 {
						// 最初の呼び出し: 空のラベル一覧
						return `[]`, nil
					} else if callCount <= 10 {
						// 9つのラベルを作成
						return "", nil
					}
					return "", fmt.Errorf("unexpected call count: %d", callCount)
				}
			},
		},
		{
			name:  "異常系: ownerが空",
			owner: "",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				// 呼ばれないはず
			},
			expectedError: "owner is required",
		},
		{
			name:  "異常系: repoが空",
			owner: "douhashi",
			repo:  "",
			setupMock: func(m *MockCommandExecutor) {
				// 呼ばれないはず
			},
			expectedError: "repo is required",
		},
		{
			name:  "異常系: ラベル一覧取得エラー",
			owner: "douhashi",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "", fmt.Errorf("failed to list labels")
				}
			},
			expectedError: "failed to list repository labels",
		},
		{
			name:  "異常系: ラベル作成エラー",
			owner: "douhashi",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				callCount := 0
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					callCount++
					if callCount == 1 {
						// 最初の呼び出し: 空のラベル一覧
						return `[]`, nil
					} else if callCount == 2 {
						// ラベル作成でエラー
						return "", fmt.Errorf("permission denied")
					}
					return "", fmt.Errorf("unexpected call")
				}
			},
			expectedError: "failed to create label",
		},
		{
			name:  "異常系: 不正なJSON応答",
			owner: "douhashi",
			repo:  "osoba",
			setupMock: func(m *MockCommandExecutor) {
				m.ExecuteFunc = func(ctx context.Context, command string, args ...string) (string, error) {
					return "invalid json", nil
				}
			},
			expectedError: "failed to parse label list",
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
			err = client.EnsureLabelsExist(context.Background(), tt.owner, tt.repo)

			// エラーの検証
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			// 成功の検証
			assert.NoError(t, err)
		})
	}
}
