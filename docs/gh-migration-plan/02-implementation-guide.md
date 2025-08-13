# 実装ガイド

## 概要
このドキュメントは、internal/ghパッケージに不足している機能を実装するための具体的なガイドです。

## 実装原則

### 1. インターフェース互換性
- internal/github.GitHubClientインターフェースを完全に実装
- メソッドシグネチャの完全一致
- 戻り値の型の一致

### 2. ghコマンド利用方針
- 可能な限りghコマンドのJSON出力を使用
- エラーハンドリングの統一
- タイムアウト処理の実装

## メソッド実装詳細

### 1. GetPullRequestForIssue

```go
// internal/gh/pull_request.go

func (c *Client) GetPullRequestForIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
    // 実装戦略1: GraphQLでリンクされたPRを検索
    query := `
    query($owner: String!, $repo: String!, $issue: Int!) {
        repository(owner: $owner, name: $repo) {
            issue(number: $issue) {
                timelineItems(first: 100, itemTypes: [CONNECTED_EVENT, CROSS_REFERENCED_EVENT]) {
                    nodes {
                        ... on ConnectedEvent {
                            subject {
                                ... on PullRequest {
                                    number
                                    state
                                    title
                                }
                            }
                        }
                    }
                }
            }
        }
    }`
    
    args := []string{
        "api", "graphql",
        "-f", fmt.Sprintf("query=%s", query),
        "-f", fmt.Sprintf("owner=%s", c.owner),
        "-f", fmt.Sprintf("repo=%s", c.repo),
        "-f", fmt.Sprintf("issue=%d", issueNumber),
    }
    
    output, err := c.executor.Execute(ctx, args...)
    if err != nil {
        // フォールバック: PR検索で探す
        return c.searchPullRequestByIssue(ctx, issueNumber)
    }
    
    // JSON解析とPullRequest構造体へのマッピング
    return parsePullRequestFromGraphQL(output)
}

// フォールバック実装
func (c *Client) searchPullRequestByIssue(ctx context.Context, issueNumber int) (*github.PullRequest, error) {
    // ブランチ名パターンで検索
    args := []string{
        "pr", "list",
        "--search", fmt.Sprintf("head:issue-%d", issueNumber),
        "--json", "number,state,title,body,headRefName,baseRefName",
    }
    
    output, err := c.executor.Execute(ctx, args...)
    if err != nil {
        return nil, fmt.Errorf("no pull request found for issue #%d", issueNumber)
    }
    
    return parsePullRequestList(output)
}
```

### 2. MergePullRequest

```go
func (c *Client) MergePullRequest(ctx context.Context, prNumber int) error {
    // マージ可能性の確認
    pr, err := c.GetPullRequestStatus(ctx, prNumber)
    if err != nil {
        return fmt.Errorf("failed to get PR status: %w", err)
    }
    
    if !pr.Mergeable {
        return fmt.Errorf("PR #%d is not mergeable", prNumber)
    }
    
    // マージ実行
    args := []string{
        "pr", "merge", fmt.Sprintf("%d", prNumber),
        "--merge",  // マージコミット方式
        "--delete-branch=false",  // ブランチは削除しない
    }
    
    _, err = c.executor.Execute(ctx, args...)
    if err != nil {
        return fmt.Errorf("failed to merge PR #%d: %w", prNumber, err)
    }
    
    return nil
}
```

### 3. GetPullRequestStatus

```go
func (c *Client) GetPullRequestStatus(ctx context.Context, prNumber int) (*github.PullRequest, error) {
    args := []string{
        "pr", "view", fmt.Sprintf("%d", prNumber),
        "--json", "number,state,title,body,mergeable,mergeStateStatus,statusCheckRollup,reviews",
    }
    
    output, err := c.executor.Execute(ctx, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to get PR status: %w", err)
    }
    
    var prData struct {
        Number           int    `json:"number"`
        State            string `json:"state"`
        Title            string `json:"title"`
        Body             string `json:"body"`
        Mergeable        bool   `json:"mergeable"`
        MergeStateStatus string `json:"mergeStateStatus"`
        StatusCheckRollup []struct {
            Status string `json:"status"`
        } `json:"statusCheckRollup"`
        Reviews []struct {
            State string `json:"state"`
        } `json:"reviews"`
    }
    
    if err := json.Unmarshal(output, &prData); err != nil {
        return nil, fmt.Errorf("failed to parse PR data: %w", err)
    }
    
    // github.PullRequestへの変換
    return &github.PullRequest{
        Number:    prData.Number,
        State:     prData.State,
        Title:     prData.Title,
        Body:      prData.Body,
        Mergeable: prData.Mergeable,
        // 追加フィールドのマッピング
    }, nil
}
```

### 4. ListPullRequestsByLabels

```go
func (c *Client) ListPullRequestsByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.PullRequest, error) {
    // ラベルをカンマ区切りに変換
    labelStr := strings.Join(labels, ",")
    
    args := []string{
        "pr", "list",
        "--repo", fmt.Sprintf("%s/%s", owner, repo),
        "--label", labelStr,
        "--json", "number,state,title,body,labels",
        "--limit", "100",  // 最大100件取得
    }
    
    output, err := c.executor.Execute(ctx, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to list PRs by labels: %w", err)
    }
    
    var prs []struct {
        Number int    `json:"number"`
        State  string `json:"state"`
        Title  string `json:"title"`
        Body   string `json:"body"`
        Labels []struct {
            Name string `json:"name"`
        } `json:"labels"`
    }
    
    if err := json.Unmarshal(output, &prs); err != nil {
        return nil, fmt.Errorf("failed to parse PR list: %w", err)
    }
    
    // github.PullRequestのスライスに変換
    result := make([]*github.PullRequest, 0, len(prs))
    for _, pr := range prs {
        labels := make([]string, 0, len(pr.Labels))
        for _, label := range pr.Labels {
            labels = append(labels, label.Name)
        }
        
        result = append(result, &github.PullRequest{
            Number: pr.Number,
            State:  pr.State,
            Title:  pr.Title,
            Body:   pr.Body,
            Labels: labels,
        })
    }
    
    return result, nil
}
```

### 5. GetClosingIssueNumber

```go
func (c *Client) GetClosingIssueNumber(ctx context.Context, prNumber int) (int, error) {
    // PR情報を取得（closing issuesを含む）
    args := []string{
        "pr", "view", fmt.Sprintf("%d", prNumber),
        "--json", "closingIssuesReferences",
    }
    
    output, err := c.executor.Execute(ctx, args...)
    if err != nil {
        // フォールバック: PR本文から解析
        return c.parseClosingIssueFromBody(ctx, prNumber)
    }
    
    var prData struct {
        ClosingIssuesReferences []struct {
            Number int `json:"number"`
        } `json:"closingIssuesReferences"`
    }
    
    if err := json.Unmarshal(output, &prData); err != nil {
        return 0, fmt.Errorf("failed to parse closing issues: %w", err)
    }
    
    if len(prData.ClosingIssuesReferences) > 0 {
        return prData.ClosingIssuesReferences[0].Number, nil
    }
    
    return 0, fmt.Errorf("no closing issue found for PR #%d", prNumber)
}

// フォールバック: PR本文から解析
func (c *Client) parseClosingIssueFromBody(ctx context.Context, prNumber int) (int, error) {
    args := []string{
        "pr", "view", fmt.Sprintf("%d", prNumber),
        "--json", "body",
    }
    
    output, err := c.executor.Execute(ctx, args...)
    if err != nil {
        return 0, err
    }
    
    var prData struct {
        Body string `json:"body"`
    }
    
    if err := json.Unmarshal(output, &prData); err != nil {
        return 0, err
    }
    
    // 正規表現でIssue番号を抽出
    patterns := []string{
        `(?i)closes?\s+#(\d+)`,
        `(?i)fixes?\s+#(\d+)`,
        `(?i)resolves?\s+#(\d+)`,
    }
    
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        if matches := re.FindStringSubmatch(prData.Body); len(matches) > 1 {
            issueNumber, _ := strconv.Atoi(matches[1])
            return issueNumber, nil
        }
    }
    
    return 0, fmt.Errorf("no closing issue found in PR body")
}
```

### 6. TransitionLabels

```go
func (c *Client) TransitionLabels(ctx context.Context, owner, repo string, issueNumber int, removeLabel, addLabel string) error {
    // 注意: ghコマンドでは原子的な操作ができないため、
    // エラー時のロールバック処理を実装
    
    // 現在のラベルを取得（ロールバック用）
    currentLabels, err := c.getIssueLabels(ctx, owner, repo, issueNumber)
    if err != nil {
        return fmt.Errorf("failed to get current labels: %w", err)
    }
    
    // ラベル削除
    if removeLabel != "" {
        removeArgs := []string{
            "issue", "edit", fmt.Sprintf("%d", issueNumber),
            "--repo", fmt.Sprintf("%s/%s", owner, repo),
            "--remove-label", removeLabel,
        }
        
        if _, err := c.executor.Execute(ctx, removeArgs...); err != nil {
            return fmt.Errorf("failed to remove label %s: %w", removeLabel, err)
        }
    }
    
    // ラベル追加
    if addLabel != "" {
        addArgs := []string{
            "issue", "edit", fmt.Sprintf("%d", issueNumber),
            "--repo", fmt.Sprintf("%s/%s", owner, repo),
            "--add-label", addLabel,
        }
        
        if _, err := c.executor.Execute(ctx, addArgs...); err != nil {
            // ロールバック: 削除したラベルを復元
            if removeLabel != "" {
                rollbackArgs := []string{
                    "issue", "edit", fmt.Sprintf("%d", issueNumber),
                    "--repo", fmt.Sprintf("%s/%s", owner, repo),
                    "--add-label", removeLabel,
                }
                c.executor.Execute(ctx, rollbackArgs...)
            }
            return fmt.Errorf("failed to add label %s: %w", addLabel, err)
        }
    }
    
    return nil
}
```

## エラーハンドリング

### エラー型の定義

```go
// internal/gh/errors.go

type GitHubError struct {
    StatusCode int
    Message    string
    Command    string
}

func (e *GitHubError) Error() string {
    return fmt.Sprintf("GitHub CLI error (status %d): %s [command: %s]", 
        e.StatusCode, e.Message, e.Command)
}

// ghコマンドのエラー出力を解析
func parseGHError(stderr []byte, command string) error {
    // ghコマンドのエラー形式に応じて解析
    errorStr := string(stderr)
    
    if strings.Contains(errorStr, "HTTP 404") {
        return &GitHubError{
            StatusCode: 404,
            Message:    "Not Found",
            Command:    command,
        }
    }
    
    if strings.Contains(errorStr, "HTTP 403") {
        return &GitHubError{
            StatusCode: 403,
            Message:    "Forbidden - API rate limit exceeded",
            Command:    command,
        }
    }
    
    return &GitHubError{
        StatusCode: 500,
        Message:    errorStr,
        Command:    command,
    }
}
```

## パフォーマンス最適化

### 1. 並列実行

```go
func (c *Client) GetMultiplePRs(ctx context.Context, prNumbers []int) ([]*github.PullRequest, error) {
    var wg sync.WaitGroup
    results := make([]*github.PullRequest, len(prNumbers))
    errors := make([]error, len(prNumbers))
    
    for i, prNumber := range prNumbers {
        wg.Add(1)
        go func(index int, num int) {
            defer wg.Done()
            pr, err := c.GetPullRequestStatus(ctx, num)
            results[index] = pr
            errors[index] = err
        }(i, prNumber)
    }
    
    wg.Wait()
    
    // エラーチェック
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }
    
    return results, nil
}
```

### 2. キャッシング

```go
type CachedClient struct {
    *Client
    cache map[string]interface{}
    mu    sync.RWMutex
}

func (c *CachedClient) GetPullRequestStatus(ctx context.Context, prNumber int) (*github.PullRequest, error) {
    key := fmt.Sprintf("pr:%d", prNumber)
    
    // キャッシュチェック
    c.mu.RLock()
    if cached, ok := c.cache[key]; ok {
        c.mu.RUnlock()
        return cached.(*github.PullRequest), nil
    }
    c.mu.RUnlock()
    
    // 実際の取得
    pr, err := c.Client.GetPullRequestStatus(ctx, prNumber)
    if err != nil {
        return nil, err
    }
    
    // キャッシュに保存
    c.mu.Lock()
    c.cache[key] = pr
    c.mu.Unlock()
    
    return pr, nil
}
```

## テスト実装

### モックExecutorの実装

```go
// internal/gh/executor_mock.go

type MockExecutor struct {
    responses map[string][]byte
    errors    map[string]error
}

func (m *MockExecutor) Execute(ctx context.Context, args ...string) ([]byte, error) {
    key := strings.Join(args, " ")
    
    if err, ok := m.errors[key]; ok {
        return nil, err
    }
    
    if response, ok := m.responses[key]; ok {
        return response, nil
    }
    
    return nil, fmt.Errorf("unexpected command: %s", key)
}
```

## 実装チェックリスト

### 各メソッドの実装確認

- [ ] GetPullRequestForIssue
  - [ ] GraphQL実装
  - [ ] フォールバック検索
  - [ ] エラーハンドリング
  - [ ] テスト作成

- [ ] MergePullRequest
  - [ ] マージ可能性チェック
  - [ ] マージ実行
  - [ ] エラーハンドリング
  - [ ] テスト作成

- [ ] GetPullRequestStatus
  - [ ] ステータス取得
  - [ ] JSON解析
  - [ ] エラーハンドリング
  - [ ] テスト作成

- [ ] ListPullRequestsByLabels
  - [ ] ラベルフィルタリング
  - [ ] ページネーション
  - [ ] エラーハンドリング
  - [ ] テスト作成

- [ ] GetClosingIssueNumber
  - [ ] API取得
  - [ ] 本文解析フォールバック
  - [ ] エラーハンドリング
  - [ ] テスト作成

- [ ] TransitionLabels
  - [ ] 原子性の確保
  - [ ] ロールバック処理
  - [ ] エラーハンドリング
  - [ ] テスト作成