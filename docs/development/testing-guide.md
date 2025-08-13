# テスティングガイド

このドキュメントは、osobaプロジェクトにおけるテストの書き方と、`internal/testutil`パッケージの使用方法について説明します。

## 目次

1. [概要](#概要)
2. [testutilパッケージの構成](#testutilパッケージの構成)
3. [モックの使い方](#モックの使い方)
4. [テストデータビルダーの使い方](#テストデータビルダーの使い方)
5. [ベストプラクティス](#ベストプラクティス)
6. [テスト実行時の安全性確保](#テスト実行時の安全性確保)
7. [よくある質問](#よくある質問)

## 概要

`internal/testutil`パッケージは、osobaプロジェクト全体で共通して使用できるテストユーティリティを提供します。このパッケージを使用することで：

- モック実装の重複を削減
- テストの記述を簡潔に
- テストデータの作成を効率化
- テストの保守性を向上

## testutilパッケージの構成

```
internal/testutil/
├── mocks/           # モック実装
│   ├── github.go    # GitHubClientモック
│   ├── logger.go    # Loggerモック（新log.Logger用）
│   ├── legacy_logger.go # Loggerモック（旧logger.Logger用）
│   ├── tmux.go      # TmuxManagerモック
│   ├── git.go       # Repositoryモック
│   └── claude.go    # ClaudeExecutorモック
├── builders/        # テストデータビルダー
│   ├── github.go    # Issue、Repository等のビルダー
│   └── config.go    # Config、TemplateVariablesビルダー
└── helpers/         # その他のヘルパー関数
```

## モックの使い方

### GitHubClientモック

```go
import (
    "testing"
    "github.com/douhashi/osoba/internal/testutil/mocks"
    "github.com/stretchr/testify/mock"
)

func TestSomething(t *testing.T) {
    // モックの作成
    mockGH := mocks.NewMockGitHubClient()
    
    // 期待する動作を設定
    mockGH.On("GetIssue", mock.Anything, "owner", "repo", 123).
        Return(&github.Issue{Number: github.Int(123)}, nil)
    
    // テスト対象のコードを実行
    result := YourFunction(mockGH)
    
    // アサーション
    assert.NoError(t, result)
    
    // モックの呼び出しを検証
    mockGH.AssertExpectations(t)
}
```

#### デフォルト動作の使用

```go
// 一般的な動作を事前設定
mockGH := mocks.NewMockGitHubClient().WithDefaultBehavior()

// GetRateLimitは自動的に正常な値を返す
rateLimit, err := mockGH.GetRateLimit(ctx)
assert.NoError(t, err)
assert.NotNil(t, rateLimit)
```

### Loggerモック

新しい統一ログシステム（`log.Logger`）用：

```go
mockLogger := mocks.NewMockLogger().WithDefaultBehavior()

// ログ出力は記録されるが、実際には出力されない
mockLogger.Debug("test message")
mockLogger.WithField("key", "value").Info("with field")
```

旧ログシステム（`logger.Logger`）用：

```go
mockLogger := mocks.NewMockLegacyLogger().WithDefaultBehavior()

// 可変長引数でフィールドを渡す
mockLogger.Info("test message", "key", "value")
```

### TmuxManagerモック

```go
mockTmux := mocks.NewMockTmuxManager()

// セッション操作
mockTmux.On("SessionExists", "osoba-test").Return(true, nil)
mockTmux.On("CreateWindow", "osoba-test", "issue-123").Return(nil)

// Issue番号からウィンドウ名を生成
mockTmux.On("GetIssueWindow", 123).Return("issue-123")
```

## テストデータビルダーの使い方

### Issueビルダー

```go
import "github.com/douhashi/osoba/internal/testutil/builders"

// デフォルト値でIssueを作成
issue := builders.NewIssueBuilder().Build()

// カスタマイズしたIssueを作成
issue := builders.NewIssueBuilder().
    WithNumber(123).
    WithTitle("Bug: Something is broken").
    WithState("open").
    WithLabels([]string{"bug", "priority:high"}).
    WithUser("testuser").
    Build()
```

### Configビルダー

```go
// デフォルト設定
config := builders.NewConfigBuilder().Build()

// カスタマイズした設定
config := builders.NewConfigBuilder().
    WithGitHubToken("test-token").
    WithPollingInterval(10 * time.Minute).
    WithTmuxSessionPrefix("test-").
    WithPlanPrompt("Custom plan prompt {{issue-number}}").
    Build()
```

### TemplateVariablesビルダー

```go
// 基本的な使い方
vars := builders.NewTemplateVariablesBuilder().
    WithIssueNumber(123).
    WithIssueTitle("Test Issue").
    WithRepoName("test-repo").
    Build()

// Issueから自動的に設定
issue := builders.NewIssueBuilder().
    WithNumber(456).
    WithTitle("From Issue").
    Build()

vars := builders.NewTemplateVariablesBuilder().
    FromIssue(issue).
    WithRepoName("repo").
    Build()
```

## ベストプラクティス

### 1. テストの独立性

各テストは独立して実行できるようにしてください：

```go
func TestFeature(t *testing.T) {
    t.Run("scenario 1", func(t *testing.T) {
        // 各サブテストで新しいモックを作成
        mockGH := mocks.NewMockGitHubClient()
        // ...
    })
    
    t.Run("scenario 2", func(t *testing.T) {
        // 独立したモックインスタンス
        mockGH := mocks.NewMockGitHubClient()
        // ...
    })
}
```

### 2. 明示的な期待値設定

モックの動作は明示的に設定してください：

```go
// 良い例：明確な期待値
mockGH.On("GetIssue", mock.Anything, "owner", "repo", 123).
    Return(specificIssue, nil)

// 避けるべき：曖昧な設定
mockGH.On("GetIssue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
    Return(mock.Anything, nil)
```

### 3. ビルダーの活用

テストデータの作成にはビルダーを使用してください：

```go
// 良い例：ビルダーを使用
issue := builders.NewIssueBuilder().
    WithNumber(123).
    WithLabels([]string{"bug"}).
    Build()

// 避けるべき：手動で構造体を作成
issue := &github.Issue{
    Number: github.Int(123),
    Labels: []*github.Label{
        {Name: github.String("bug")},
    },
}
```

### 4. エラーケースのテスト

正常系だけでなく、エラーケースも必ずテストしてください：

```go
t.Run("API error", func(t *testing.T) {
    mockGH.On("GetIssue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
        Return(nil, errors.New("API error"))
    
    _, err := service.GetIssue(ctx, 123)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "API error")
})
```

### 5. 並行テストの考慮

Goのテストは並行実行される可能性があるため、グローバル状態を避けてください：

```go
func TestParallel(t *testing.T) {
    t.Parallel() // 並行実行を許可
    
    // 各テストで独立したリソースを使用
    mockGH := mocks.NewMockGitHubClient()
    // ...
}
```

## よくある質問

### Q: 既存のテストをリファクタリングする際の注意点は？

A: 段階的に移行することをお勧めします：

1. 新しいテストから`testutil`パッケージを使用
2. 既存テストは、修正が必要になったタイミングで移行
3. 全てのテストが通ることを確認しながら進める

### Q: モックの`Maybe()`メソッドは何ですか？

A: `Maybe()`は、そのメソッドが0回以上呼ばれることを許可します。デフォルト動作の設定時に使用します：

```go
// 呼ばれるかもしれないし、呼ばれないかもしれない
mockGH.On("GetRateLimit", mock.Anything).Maybe().Return(rateLimit, nil)
```

### Q: カスタムマッチャーの使い方は？

A: `mock.MatchedBy`を使用して、複雑な条件でマッチングできます：

```go
mockGH.On("CreateIssueComment", mock.Anything, mock.Anything, mock.Anything,
    mock.MatchedBy(func(n int) bool {
        return n > 0 && n < 100
    }),
    mock.MatchedBy(func(s string) bool {
        return len(s) > 0
    }),
).Return(nil)
```

### Q: テストヘルパー関数はどこに置くべきですか？

A: 以下の基準で判断してください：

- 特定のパッケージ専用：そのパッケージ内の`test_helpers.go`
- 複数パッケージで共有：`internal/testutil/helpers`
- モックやビルダー：それぞれ`internal/testutil/mocks`、`internal/testutil/builders`

## テスト実行時の安全性確保

### 本番セッションの保護

osoba開発ではosobaを使った開発（dogfooding）を行っているため、テスト実行時に本番のtmuxセッションを誤って削除しないよう、以下の仕組みを実装しています：

#### 1. セッション命名規則の分離

- **本番セッション**: `osoba-`プレフィックスを使用（例：`osoba-repo-name`）
- **テストセッション**: `test-osoba-`プレフィックスを使用（例：`test-osoba-session-20240101-120000`）

#### 2. 環境変数によるテストモード制御

```bash
# テストモードを有効にする
export OSOBA_TEST_MODE=true

# CI環境を明示する（GitHub Actions等で自動設定）
export CI=true
```

#### 3. テスト用設定ファイル

`.osoba.test.yml`ファイルにテスト専用の設定を定義：

```yaml
tmux:
  session_prefix: "test-osoba-"
```

#### 4. 統合テストでの安全性チェック

統合テスト実行前に自動的に以下のチェックが実行されます：

```go
// internal/tmux/safety_check.go
func SafetyCheckBeforeTests() error {
    // 本番セッションの存在確認
    prodSessions, err := CheckProductionSessions()
    if len(prodSessions) > 0 {
        // 警告を表示
        fmt.Fprintf(os.Stderr, "⚠️  WARNING: Found production sessions\n")
        // CI環境では続行、ローカルでは注意喚起
    }
    return nil
}
```

### テスト実行のベストプラクティス

1. **統合テスト実行前**：
   ```bash
   # テストモードを設定
   export OSOBA_TEST_MODE=true
   
   # 統合テストを実行
   make integration-test
   ```

2. **本番セッションの確認**：
   ```bash
   # 現在のtmuxセッション一覧を確認
   tmux list-sessions
   ```

3. **テストセッションのクリーンアップ**：
   ```bash
   # test-osoba-プレフィックスのセッションのみ削除
   tmux list-sessions -F "#{session_name}" | grep "^test-osoba-" | xargs -I {} tmux kill-session -t {}
   ```

### CI/CD環境での設定

GitHub Actionsなどのでは、以下の環境変数が自動設定されます：

```yaml
env:
  OSOBA_TEST_MODE: true
  CI: true
```

これにより、CI環境でのテスト実行が安全に行われます。

## まとめ

`internal/testutil`パッケージを活用することで、テストの記述が簡潔になり、保守性が向上します。新しいテストを作成する際は、まずこのパッケージの既存の機能を確認し、必要に応じて拡張してください。また、テスト実行時は本番環境への影響を防ぐため、適切な環境変数の設定とセッション命名規則の遵守を心がけてください。