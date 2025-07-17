# テスト戦略とモック使用ガイドライン

## 概要

本ドキュメントは、osobaプロジェクトにおけるテスト戦略とモック使用の指針を定めます。テストピラミッドに基づき、適切なレベルで適切なテスト手法を採用することで、信頼性が高く保守しやすいテストスイートを構築します。

## テストピラミッド

```
    /\
   /  \ E2E Tests (少数)
  /    \ 
 /______\ Integration Tests (中程度)
/________\ Unit Tests (多数)
```

### 1. ユニットテスト (Unit Tests)
- **目的**: 単一のモジュールや関数の動作を検証
- **モック方針**: **外部依存は積極的にモック化**
- **実行速度**: 高速 (< 10ms/test)
- **ファイル命名**: `*_test.go`

### 2. 統合テスト (Integration Tests)
- **目的**: 複数のコンポーネント間の連携を検証
- **モック方針**: **外部サービスのみモック化、内部コンポーネントは実物を使用**
- **実行速度**: 中程度 (< 1s/test)
- **ファイル命名**: `*_integration_test.go`
- **ビルドタグ**: `//go:build integration`

### 3. E2Eテスト (End-to-End Tests)
- **目的**: アプリケーション全体のフローを検証
- **モック方針**: **最小限、外部サービスのみ**
- **実行速度**: 低速 (< 30s/test)
- **ファイル命名**: `*_e2e_test.go`
- **ビルドタグ**: `//go:build e2e`

## モック使用の基本原則

### ✅ モックすべきもの

1. **外部サービス**
   - GitHub API
   - ファイルシステム操作
   - ネットワーク通信
   - データベース

2. **時間依存の処理**
   - `time.Now()`
   - タイマー
   - ポーリング処理

3. **重い処理**
   - 長時間実行される処理
   - リソース集約的な処理

### ❌ モックすべきでないもの

1. **内部コンポーネント間の連携** (統合テストにおいて)
   - 同一プロセス内のモジュール間通信
   - データ変換・フォーマット処理
   - ビジネスロジック

2. **軽量な処理**
   - 文字列操作
   - 単純な計算
   - 設定オブジェクトの生成

3. **標準ライブラリ**
   - `strings`, `strconv`, `fmt`など

## レイヤー別モック戦略

### コマンドレイヤー (`cmd/`)

```go
// ❌ 悪い例: 統合テストなのに内部コンポーネントをモック化
func TestIntegration_WatchCommand(t *testing.T) {
    mockGH := mocks.NewMockGitHubClient()
    mockTmux := mocks.NewMockTmuxManager()
    // ...
}

// ✅ 良い例: 外部サービスのみモック化
func TestIntegration_WatchCommand(t *testing.T) {
    // GitHub APIのみモック化
    mockHTTPClient := &mockHTTPClient{}
    realGitHubClient := github.NewClientWithHTTP(mockHTTPClient)
    
    // 実際のtmux, git worktreeコンポーネントを使用
    realTmuxManager := tmux.NewManager()
    realGitManager := git.NewWorktreeManager()
    // ...
}
```

### GitHubクライアントレイヤー (`internal/github/`)

```go
// ユニットテスト: HTTP通信をモック化
func TestGitHubClient_ListIssues(t *testing.T) {
    mockHTTP := &MockHTTPClient{}
    client := NewClientWithHTTP(mockHTTP)
    // ...
}

// 統合テスト: 実際のHTTP通信（テスト環境）
func TestIntegration_GitHubClient(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        t.Skip("GITHUB_TOKEN not set")
    }
    
    client := NewClient(token)
    // 実際のGitHub APIを呼び出し
}
```

### tmuxマネージャーレイヤー (`internal/tmux/`)

```go
// ユニットテスト: コマンド実行をモック化
func TestTmuxManager_CreateSession(t *testing.T) {
    mockExecutor := mocks.NewMockCommandExecutor()
    manager := NewManagerWithExecutor(mockExecutor)
    // ...
}

// 統合テスト: 実際のtmuxコマンド
func TestIntegration_TmuxManager(t *testing.T) {
    if !isTmuxAvailable() {
        t.Skip("tmux not available")
    }
    
    manager := NewManager()
    // 実際のtmuxコマンドを実行
}
```

## テストの分類とファイル構成

### ディレクトリ構造

```
pkg/
├── feature/
│   ├── feature.go
│   ├── feature_test.go           # ユニットテスト
│   ├── feature_integration_test.go  # 統合テスト
│   └── feature_e2e_test.go       # E2Eテスト
└── testutil/                     # テストユーティリティ
    ├── mocks/                    # モック実装
    ├── builders/                 # テストデータビルダー
    └── fixtures/                 # テストフィクスチャ
```

### ビルドタグの使用

```go
//go:build integration
// +build integration

package github

// 統合テストのみ実行
```

```go
//go:build e2e
// +build e2e

package cmd

// E2Eテストのみ実行
```

## テスト実行方法

### 開発時

```bash
# ユニットテストのみ（高速）
go test ./...

# 統合テストを含む
go test -tags=integration ./...

# 全テスト（CI環境）
go test -tags="integration e2e" ./...
```

### CI/CD

```yaml
# .github/workflows/test.yml
- name: Unit Tests
  run: go test -race ./...

- name: Integration Tests  
  run: go test -race -tags=integration ./...
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

- name: E2E Tests
  run: go test -race -tags=e2e ./...
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## モック実装のベストプラクティス

### 1. インターフェース設計

```go
// 良い例: 小さく、焦点を絞ったインターフェース
type IssueReader interface {
    ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*Issue, error)
}

type LabelManager interface {
    AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
    RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error
}
```

### 2. モック生成

```go
//go:generate mockery --name=GitHubClient --dir=. --output=../testutil/mocks
type GitHubClient interface {
    // メソッド定義
}
```

### 3. デフォルト動作の提供

```go
func (m *MockGitHubClient) WithDefaultBehavior() *MockGitHubClient {
    m.On("GetRateLimit", mock.Anything).Return(&RateLimits{
        Core: &RateLimit{Remaining: 5000, Limit: 5000},
    }, nil)
    return m
}
```

## 統合テストの環境構築

### TestContainersの活用

```go
func setupTestEnvironment(t *testing.T) *TestEnvironment {
    // Docker環境でGitHubサーバーのモックを起動
    githubContainer := testcontainers.StartGitHubMock(t)
    
    return &TestEnvironment{
        GitHubURL: githubContainer.GetBaseURL(),
        TempDir:   t.TempDir(),
    }
}
```

### 環境変数による制御

```go
func TestIntegration_GitHubAPI(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // ghコマンドが認証済みかチェック
    if err := exec.Command("gh", "auth", "status").Run(); err != nil {
        t.Skip("gh command not authenticated")
    }
    
    // 実際のAPI呼び出し
}
```

## エラーケースのテスト

### ネットワークエラー

```go
func TestGitHubClient_NetworkError(t *testing.T) {
    // タイムアウト、接続エラーなどをシミュレート
    mockHTTP := &MockHTTPClient{}
    mockHTTP.On("Do", mock.Anything).Return(nil, errors.New("network error"))
    
    client := NewClientWithHTTP(mockHTTP)
    _, err := client.ListIssues(context.Background(), "owner", "repo")
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "network error")
}
```

### レート制限

```go
func TestGitHubClient_RateLimit(t *testing.T) {
    mockHTTP := &MockHTTPClient{}
    mockHTTP.SetupRateLimitResponse(403, "API rate limit exceeded")
    
    client := NewClientWithHTTP(mockHTTP)
    _, err := client.ListIssues(context.Background(), "owner", "repo")
    
    assert.Error(t, err)
    assert.IsType(t, &RateLimitError{}, err)
}
```

## パフォーマンステスト

### ベンチマーク

```go
func BenchmarkIssueProcessing(b *testing.B) {
    mockGH := mocks.NewMockGitHubClient().WithDefaultBehavior()
    watcher := NewIssueWatcher(mockGH, "owner", "repo", []string{"bug"})
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        watcher.ProcessIssue(testIssue)
    }
}
```

### メモリ使用量

```go
func TestMemoryUsage(t *testing.T) {
    var m1, m2 runtime.MemStats
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    // テスト実行
    processLargeDataSet()
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    memoryUsed := m2.TotalAlloc - m1.TotalAlloc
    if memoryUsed > 10*1024*1024 { // 10MB
        t.Errorf("Memory usage too high: %d bytes", memoryUsed)
    }
}
```

## まとめ

- **ユニットテスト**: 外部依存は積極的にモック化
- **統合テスト**: 外部サービスのみモック化、内部コンポーネントは実物
- **E2Eテスト**: 最小限のモック化
- **明確な境界**: ビルドタグとファイル命名で分離
- **CI/CD対応**: 段階的なテスト実行

この指針に従うことで、テストの信頼性と保守性を大幅に向上させることができます。