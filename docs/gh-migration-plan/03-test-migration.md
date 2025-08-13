# テスト移行戦略

## 概要
internal/githubの76個のテストをinternal/ghで動作させるための包括的な戦略。

## テスト移行の原則

### 1. 完全互換性
- 入力と出力が同一であることを保証
- エラーケースも同様に処理
- タイムアウトやリトライ動作の一致

### 2. テスト駆動移行
- テストが通ることが実装完了の条件
- カバレッジ85%以上を維持
- リグレッションテストの自動化

## テスト分類と移行優先度

### 優先度1: コア機能テスト（25個）
```
client_test.go
- TestListIssuesByLabels
- TestListAllOpenIssues
- TestGetRepository
- TestGetRateLimit
- TestEnsureLabelsExist
- TestCreateIssueComment
- TestRemoveLabel
- TestAddLabel
```

### 優先度2: PR関連テスト（20個）
```
pull_request_test.go
- TestGetPullRequestForIssue
- TestMergePullRequest
- TestGetPullRequestStatus
- TestListPullRequestsByLabels

pull_request_graphql_closing_issue_test.go
- TestGetClosingIssueNumber
- TestParseClosingIssueFromBody
```

### 優先度3: エラー処理テスト（15個）
```
errors_test.go
- TestGitHubError
- TestParseGHError

error_parser_test.go
- TestParseAPIError
- TestRateLimitError
```

### 優先度4: 統合テスト（16個）
```
integration_test.go
- TestEndToEndWorkflow
- TestConcurrentOperations
```

## テスト移行手順

### Step 1: テストヘルパーの移植

```go
// internal/gh/test_helpers.go

// internal/githubから移植するヘルパー関数
func setupTestClient(t *testing.T) *Client {
    executor := NewMockExecutor()
    client, err := NewClient(executor)
    require.NoError(t, err)
    return client
}

func mockGHResponse(command string, response interface{}) []byte {
    data, _ := json.Marshal(response)
    return data
}

// テスト用のIssue作成
func createTestIssue(number int, labels ...string) *github.Issue {
    return &github.Issue{
        Number: number,
        State:  "open",
        Labels: labels,
    }
}
```

### Step 2: モック実装の調整

```go
// internal/gh/mock_executor.go

type MockExecutor struct {
    t         *testing.T
    responses map[string]mockResponse
    calls     []string
}

type mockResponse struct {
    output []byte
    error  error
}

func NewMockExecutor(t *testing.T) *MockExecutor {
    return &MockExecutor{
        t:         t,
        responses: make(map[string]mockResponse),
        calls:     make([]string, 0),
    }
}

func (m *MockExecutor) SetResponse(command string, output []byte, err error) {
    m.responses[command] = mockResponse{
        output: output,
        error:  err,
    }
}

func (m *MockExecutor) Execute(ctx context.Context, args ...string) ([]byte, error) {
    command := strings.Join(args, " ")
    m.calls = append(m.calls, command)
    
    if response, ok := m.responses[command]; ok {
        return response.output, response.error
    }
    
    m.t.Fatalf("unexpected command: %s", command)
    return nil, nil
}

func (m *MockExecutor) AssertCalled(t *testing.T, command string) {
    for _, call := range m.calls {
        if call == command {
            return
        }
    }
    t.Errorf("command not called: %s", command)
}
```

### Step 3: テストケースの移行

#### 例: ListIssuesByLabelsのテスト移行

```go
// internal/gh/list_issues_test.go

func TestListIssuesByLabels(t *testing.T) {
    tests := []struct {
        name      string
        labels    []string
        mockSetup func(*MockExecutor)
        want      []*github.Issue
        wantErr   bool
    }{
        {
            name:   "single label",
            labels: []string{"bug"},
            mockSetup: func(m *MockExecutor) {
                response := []byte(`[
                    {
                        "number": 1,
                        "state": "open",
                        "title": "Bug issue",
                        "labels": [{"name": "bug"}]
                    }
                ]`)
                m.SetResponse("issue list --label bug --json number,state,title,labels", response, nil)
            },
            want: []*github.Issue{
                {
                    Number: 1,
                    State:  "open",
                    Title:  "Bug issue",
                    Labels: []string{"bug"},
                },
            },
            wantErr: false,
        },
        {
            name:   "multiple labels",
            labels: []string{"bug", "urgent"},
            mockSetup: func(m *MockExecutor) {
                response := []byte(`[
                    {
                        "number": 2,
                        "state": "open",
                        "title": "Urgent bug",
                        "labels": [{"name": "bug"}, {"name": "urgent"}]
                    }
                ]`)
                m.SetResponse("issue list --label bug,urgent --json number,state,title,labels", response, nil)
            },
            want: []*github.Issue{
                {
                    Number: 2,
                    State:  "open",
                    Title:  "Urgent bug",
                    Labels: []string{"bug", "urgent"},
                },
            },
            wantErr: false,
        },
        {
            name:   "API error",
            labels: []string{"bug"},
            mockSetup: func(m *MockExecutor) {
                m.SetResponse("issue list --label bug --json number,state,title,labels", 
                    nil, fmt.Errorf("API rate limit exceeded"))
            },
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            executor := NewMockExecutor(t)
            tt.mockSetup(executor)
            
            client, _ := NewClient(executor)
            got, err := client.ListIssuesByLabels(context.Background(), "owner", "repo", tt.labels)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ListIssuesByLabels() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ListIssuesByLabels() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Step 4: 統合テストの実装

```go
// internal/gh/integration_test.go

func TestIntegration_IssueWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // 実際のghコマンドを使用
    executor := NewRealCommandExecutor()
    client, err := NewClient(executor)
    require.NoError(t, err)
    
    ctx := context.Background()
    
    // テスト用のリポジトリ情報（環境変数から取得）
    owner := os.Getenv("TEST_GITHUB_OWNER")
    repo := os.Getenv("TEST_GITHUB_REPO")
    
    if owner == "" || repo == "" {
        t.Skip("TEST_GITHUB_OWNER and TEST_GITHUB_REPO must be set")
    }
    
    // 1. リポジトリ情報取得
    repository, err := client.GetRepository(ctx, owner, repo)
    assert.NoError(t, err)
    assert.NotNil(t, repository)
    
    // 2. Issue一覧取得
    issues, err := client.ListAllOpenIssues(ctx, owner, repo)
    assert.NoError(t, err)
    assert.NotNil(t, issues)
    
    // 3. レート制限確認
    rateLimit, err := client.GetRateLimit(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, rateLimit)
    
    // 4. ラベル確認
    err = client.EnsureLabelsExist(ctx, owner, repo)
    assert.NoError(t, err)
}
```

## テストカバレッジ戦略

### カバレッジ目標
- 全体: 85%以上
- コア機能: 95%以上
- PR機能: 90%以上
- エラー処理: 80%以上

### カバレッジ測定

```bash
# カバレッジ測定コマンド
go test -coverprofile=coverage.out ./internal/gh/...
go tool cover -html=coverage.out -o coverage.html

# カバレッジレポート生成
go test -covermode=atomic -coverprofile=coverage.txt ./internal/gh/...
```

### カバレッジ改善

```go
// internal/gh/coverage_test.go

// エッジケースのテスト追加
func TestEdgeCases(t *testing.T) {
    tests := []struct {
        name string
        test func(*testing.T)
    }{
        {"empty response", testEmptyResponse},
        {"malformed JSON", testMalformedJSON},
        {"timeout", testTimeout},
        {"large response", testLargeResponse},
        {"concurrent access", testConcurrentAccess},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, tt.test)
    }
}
```

## テスト実行戦略

### 並列実行

```go
func TestParallel(t *testing.T) {
    t.Parallel() // テストを並列実行
    
    // サブテストも並列化
    t.Run("subtest1", func(t *testing.T) {
        t.Parallel()
        // テスト実装
    })
}
```

### テストグループ化

```bash
# 単体テストのみ実行
go test -short ./internal/gh/...

# 統合テストを含む全テスト
go test ./internal/gh/...

# 特定のテストのみ実行
go test -run TestListIssues ./internal/gh/...
```

## 移行検証チェックリスト

### Phase 1: 基本機能テスト
- [ ] ListIssuesByLabels: 10テスト
- [ ] ListAllOpenIssues: 5テスト
- [ ] GetRepository: 3テスト
- [ ] GetRateLimit: 3テスト
- [ ] EnsureLabelsExist: 4テスト

### Phase 2: PR機能テスト
- [ ] GetPullRequestForIssue: 5テスト
- [ ] MergePullRequest: 4テスト
- [ ] GetPullRequestStatus: 4テスト
- [ ] ListPullRequestsByLabels: 4テスト
- [ ] GetClosingIssueNumber: 3テスト

### Phase 3: ラベル操作テスト
- [ ] TransitionIssueLabel: 5テスト
- [ ] TransitionIssueLabelWithInfo: 3テスト
- [ ] RemoveLabel: 3テスト
- [ ] AddLabel: 3テスト
- [ ] TransitionLabels: 4テスト

### Phase 4: エラー処理テスト
- [ ] API エラー: 5テスト
- [ ] タイムアウト: 3テスト
- [ ] レート制限: 3テスト
- [ ] 認証エラー: 2テスト
- [ ] ネットワークエラー: 2テスト

### Phase 5: 統合テスト
- [ ] E2E ワークフロー: 5テスト
- [ ] 並行処理: 3テスト
- [ ] パフォーマンス: 3テスト
- [ ] リトライ機構: 3テスト
- [ ] キャッシング: 2テスト

## 成功基準

### 定量的基準
1. **テスト数**: 76個すべてのテストが移行完了
2. **成功率**: 100%のテストが成功
3. **カバレッジ**: 85%以上のコードカバレッジ
4. **実行時間**: 既存テストの1.5倍以内

### 定性的基準
1. **保守性**: テストコードの可読性向上
2. **拡張性**: 新規テスト追加が容易
3. **信頼性**: フレーキーなテストの排除
4. **ドキュメント**: テストの意図が明確