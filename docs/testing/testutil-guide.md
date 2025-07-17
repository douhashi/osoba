# testutil パッケージ使用ガイド

このドキュメントでは、osobaプロジェクトの`internal/testutil`パッケージの使用方法について説明します。testutilパッケージは、テストコードの重複を削減し、一貫性のあるテストを作成するための共通ユーティリティを提供します。

## 目次

1. [概要](#概要)
2. [パッケージ構成](#パッケージ構成)
3. [モック（Mocks）](#モックmocks)
4. [ビルダー（Builders）](#ビルダーbuilders)
5. [ヘルパー（Helpers）](#ヘルパーhelpers)
6. [使用例](#使用例)
7. [ベストプラクティス](#ベストプラクティス)

## 概要

testutilパッケージは、以下の3つのサブパッケージで構成されています：

- **mocks**: 各種インターフェースのモック実装
- **builders**: テストデータの構築を簡潔にするビルダーパターン実装
- **helpers**: テスト作成を支援する共通ヘルパー関数

## パッケージ構成

```
internal/testutil/
├── mocks/          # モック実装
│   ├── github.go   # GitHub APIクライアントのモック
│   ├── tmux.go     # Tmuxクライアントのモック
│   ├── git.go      # Gitクライアントのモック
│   └── ...
├── builders/       # テストデータビルダー
│   ├── github.go   # GitHub関連データのビルダー
│   └── config.go   # 設定データのビルダー
└── helpers/        # ヘルパー関数
    ├── time.go     # 時間関連ヘルパー
    ├── env.go      # 環境変数管理
    ├── errors.go   # エラー生成・検証
    └── command.go  # コマンド実行記録
```

## モック（Mocks）

### GitHubクライアントモック

```go
import "github.com/douhashi/osoba/internal/testutil/mocks"

// テスト内での使用例
func TestGitHubIntegration(t *testing.T) {
    // モックの作成
    mockClient := mocks.NewGitHubClient(t)
    
    // 期待する動作の設定
    mockClient.SetListIssuesFunc(func(owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, error) {
        return []*github.Issue{
            {Number: github.Int(1), Title: github.String("Test Issue")},
        }, nil
    })
    
    // モックを使用したテスト実行
    issues, err := mockClient.ListIssues("owner", "repo", nil)
    
    // 呼び出し回数の検証
    if mockClient.ListIssuesCallCount() != 1 {
        t.Errorf("expected ListIssues to be called once, got %d", mockClient.ListIssuesCallCount())
    }
}
```

### Tmuxクライアントモック

```go
// セッション存在チェックのモック
mockTmux := mocks.NewTmuxClient(t)
mockTmux.SetHasSessionFunc(func(name string) (bool, error) {
    return name == "existing-session", nil
})

// ウィンドウ作成のモック
mockTmux.SetNewWindowFunc(func(session, name, dir string) error {
    if session == "" {
        return errors.New("session name required")
    }
    return nil
})
```

### Gitクライアントモック

```go
// worktree作成のモック
mockGit := mocks.NewGitClient(t)
mockGit.SetWorktreeAddFunc(func(path, branch string, opts ...string) error {
    if path == "" {
        return errors.New("path required")
    }
    return nil
})

// リポジトリ存在チェックのモック
mockGit.SetIsRepositoryFunc(func(path string) bool {
    return path == "/valid/repo/path"
})
```

## ビルダー（Builders）

### GitHubデータビルダー

```go
import "github.com/douhashi/osoba/internal/testutil/builders"

// Issue作成
issue := builders.NewIssueBuilder().
    WithNumber(123).
    WithTitle("Fix bug in parser").
    WithStatusLabel("ready").
    WithPriorityLabel("high").
    WithUser("octocat").
    Build()

// Repository作成
repo := builders.NewRepositoryBuilder().
    WithName("osoba").
    WithOwner("douhashi").
    WithPrivate(false).
    Build()

// Label作成
statusLabel := builders.NewLabelBuilder().
    AsStatusLabel("implementing") // 自動的に適切な色が設定される

priorityLabel := builders.NewLabelBuilder().
    AsPriorityLabel("high") // 自動的に適切な色が設定される
```

### 設定ビルダー

```go
// 基本的な設定
config := builders.NewConfigBuilder().
    WithGitHubToken("test-token").
    WithRepoOwner("test-owner").
    WithRepoName("test-repo").
    Build()

// Claudeオプション付き設定
config := builders.NewConfigBuilder().
    WithClaudeOptions(builders.ClaudeOptions{
        Model: "claude-3",
        MaxTokens: 4000,
    }).
    Build()
```

## ヘルパー（Helpers）

### 時間関連ヘルパー

```go
import "github.com/douhashi/osoba/internal/testutil/helpers"

// 時間のパース（エラー時はテスト失敗）
createdAt := helpers.MustParseTime(t, "2023-01-01T00:00:00Z")

// 時間のポインタ作成
updatedAt := helpers.TimePtr(time.Now())

// 現在時刻のポインタ
now := helpers.NowPtr()

// Duration のパース
timeout := helpers.MustParseDuration(t, "30s")
```

### 環境変数管理

```go
// 単一の環境変数設定（自動的に復元）
cleanup := helpers.SetEnv(t, "GITHUB_TOKEN", "test-token")
defer cleanup()

// 複数の環境変数管理
guard := helpers.NewEnvGuard(t)
defer guard.Restore()

guard.Set("GITHUB_TOKEN", "test-token")
guard.Set("DEBUG", "true")
guard.Unset("UNWANTED_VAR")
```

### エラー生成と検証

```go
// 共通エラーの使用
err := helpers.ErrNotFound
err := helpers.ErrAPILimit

// テスト用エラーの生成
err := helpers.NewTestError("something went wrong")
err := helpers.NewTestErrorf("failed to connect to %s", "server")

// エラーメッセージの部分一致検証
if !helpers.ErrorContains(err, "connection refused") {
    t.Errorf("expected error to contain 'connection refused'")
}
```

### コマンド実行記録

```go
// コマンド実行の記録と検証
recorder := helpers.NewCommandRecorder(t)

// コマンド実行を記録
recorder.Record("git", []string{"status"}, helpers.CommandResult{
    Output: "On branch main",
    Error:  nil,
})

// 検証
recorder.AssertCalled("git")
recorder.AssertCallCount("git", 1)
recorder.AssertNotCalled("tmux")
```

## 使用例

### 完全なテスト例

```go
func TestWatcher_ProcessIssue(t *testing.T) {
    // 環境変数の設定
    guard := helpers.NewEnvGuard(t)
    defer guard.Restore()
    guard.Set("GITHUB_TOKEN", "test-token")
    
    // モックの準備
    mockGitHub := mocks.NewGitHubClient(t)
    mockTmux := mocks.NewTmuxClient(t)
    mockGit := mocks.NewGitClient(t)
    
    // テストデータの作成
    issue := builders.NewIssueBuilder().
        WithNumber(123).
        WithTitle("Implement new feature").
        WithStatusLabel("ready").
        WithCreatedAt(helpers.MustParseTime(t, "2023-01-01T00:00:00Z")).
        Build()
    
    // モックの動作設定
    mockGitHub.SetGetIssueFunc(func(owner, repo string, number int) (*github.Issue, error) {
        if number == 123 {
            return issue, nil
        }
        return nil, helpers.ErrNotFound
    })
    
    mockTmux.SetNewWindowFunc(func(session, name, dir string) error {
        return nil
    })
    
    // テスト対象の実行
    watcher := NewWatcher(mockGitHub, mockTmux, mockGit)
    err := watcher.ProcessIssue(123)
    
    // 結果の検証
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    // モックの呼び出し検証
    if mockGitHub.GetIssueCallCount() != 1 {
        t.Errorf("expected GetIssue to be called once")
    }
}
```

## ベストプラクティス

### 1. モックの適切な使用

- 外部依存関係（API、コマンド実行）には必ずモックを使用する
- モックの動作は明示的に設定し、デフォルト動作に依存しない
- 必要な呼び出し回数を検証する

### 2. ビルダーの活用

- テストデータは可能な限りビルダーを使用して作成する
- 必要最小限のフィールドのみ設定し、デフォルト値を活用する
- 複雑なテストデータは専用のヘルパー関数を作成する

### 3. ヘルパーの効果的な使用

- 環境変数は必ず復元する（defer文を使用）
- 時間関連の値はヘルパーを使用してパースする
- エラーメッセージは部分一致で検証し、柔軟性を保つ

### 4. テストの独立性

- 各テストは独立して実行可能にする
- グローバル状態に依存しない
- テスト間で共有される状態を避ける

### 5. 可読性の向上

- Arrange-Act-Assert パターンに従う
- テストデータの準備、実行、検証を明確に分ける
- 意図が明確なテスト名を付ける

## まとめ

testutilパッケージを活用することで：

1. **一貫性**: すべてのテストで同じパターンを使用
2. **保守性**: モックやビルダーの変更が一箇所で管理される
3. **可読性**: テストコードがシンプルで理解しやすくなる
4. **信頼性**: 共通のエラーパターンや検証方法により、テストの品質が向上

新しいテストを作成する際は、まずtestutilパッケージの既存機能を確認し、必要に応じて拡張することを推奨します。