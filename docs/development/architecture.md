# アーキテクチャ設計書

## 概要

osobaは、GitHub Issue検知から自動的なラベル遷移まで、開発プロセス全体を自律的に管理するGoベースのCLIツールです。tmux + git worktree + Claude AI の統合により、計画・実装・レビューの3フェーズを自動実行します。

## システム全体構成

### 全体構成図

```mermaid
graph TB
    subgraph "コマンドレイヤー"
        CLI[CLI Commands]
        ROOT[root.go]
        INIT[init.go]
        START[start.go]
        OPEN[open.go]
        STATUS[status.go]
        CLEAN[clean.go]
    end

    subgraph "コアエンジンレイヤー"
        WATCHER[IssueWatcher]
        AM[ActionManager]
        AF[ActionFactory]
    end

    subgraph "アクションレイヤー"
        PA[PlanAction]
        IA[ImplementationAction]
        RA[ReviewAction]
        PT[PhaseTransitioner]
    end

    subgraph "統合レイヤー"
        GH[GitHub Client]
        TMUX[Tmux Manager]
        GIT[Git Worktree]
        CLAUDE[Claude Executor]
    end

    subgraph "外部システム"
        GITHUB[(GitHub API)]
        TMUX_SYS[tmux System]
        GIT_SYS[Git Repository]
        CLAUDE_AI[Claude AI]
    end

    CLI --> ROOT
    ROOT --> WATCHER
    WATCHER --> AM
    AM --> AF
    AF --> PA
    AF --> IA
    AF --> RA
    PA --> PT
    IA --> PT
    RA --> PT
    PA --> GH
    PA --> TMUX
    PA --> GIT
    PA --> CLAUDE
    IA --> GH
    IA --> TMUX
    IA --> GIT
    IA --> CLAUDE
    RA --> GH
    RA --> TMUX
    RA --> GIT
    RA --> CLAUDE
    GH --> GITHUB
    TMUX --> TMUX_SYS
    GIT --> GIT_SYS
    CLAUDE --> CLAUDE_AI
```

## コンポーネント詳細

### 1. コマンドレイヤー (cmd/)

ユーザーインターフェースとCLIコマンドの定義を担当。

| コマンド | 責務 | 実装ファイル |
|----------|------|-------------|
| `osoba init` | プロジェクト初期化 | cmd/init.go |
| `osoba start` | Issue監視開始 | cmd/start.go |
| `osoba open` | tmuxセッション接続 | cmd/open.go |
| `osoba status` | システム状況確認 | cmd/status.go |
| `osoba clean` | リソースクリーンアップ | cmd/clean.go |

### internal/watcher

システムの中核となるIssue監視とアクション実行エンジン。

```mermaid
classDiagram
    class IssueWatcher {
        -client GitHubClient
        -owner string
        -repo string
        -labels []string
        -pollInterval time.Duration
        -actionManager ActionManagerInterface
        +Start(ctx, callback)
        +StartWithActions(ctx)
        +checkIssues(ctx, callback)
    }

    class ActionManager {
        -sessionName string
        -stateManager IssueStateManager
        -factory ActionFactory
        +ExecuteAction(ctx, issue) error
        +GetActionForIssue(issue) ActionExecutor
    }

    class ActionFactory {
        +CreatePlanAction() ActionExecutor
        +CreateImplementationAction() ActionExecutor
        +CreateReviewAction() ActionExecutor
    }

    IssueWatcher --> ActionManager
    ActionManager --> ActionFactory
```

#### 主要コンポーネント

- **IssueWatcher**: GitHub Issue監視の中核
  - ポーリングによるIssue検知
  - ラベル変更追跡
  - ヘルスチェック機能
  - イベント通知システム

- **ActionManager**: アクション実行の統括
  - Issue状態管理
  - アクション実行調整
  - フェーズ間の遷移制御

- **ActionFactory**: アクションインスタンス生成
  - ラベルベースのアクション選択
  - 依存性注入
  - テスト容易性の確保

### internal/github

GitHub API統合とラベル管理を担当。

### internal/tmux

tmuxセッション・ウィンドウ管理を担当。

### internal/git

git worktree操作とブランチ管理を担当。

### internal/claude

Claude AI実行管理とテンプレート処理を担当。

## 依存関係

各コンポーネント間の依存関係と呼び出し関係を示します。

```mermaid
classDiagram
    class GitHubClient {
        +ListIssuesByLabels() []Issue
        +AddLabel(issueNumber, label) error
        +RemoveLabel(issueNumber, label) error
        +CreateComment(issueNumber, body) error
        +GetRateLimit() RateLimits
    }

    class LabelManager {
        +TransitionLabel(from, to) error
        +EnsureLabelsExist() error
        +ValidateLabels() error
    }

    class WorktreeManager {
        +CreateWorktree(issueNumber, phase) error
        +GetWorktreePath(issueNumber, phase) string
        +UpdateMainBranch() error
        +CleanupWorktree(issueNumber) error
    }

    class TmuxManager {
        +CreateSession(name) error
        +CreateWindowForIssue(session, issue, phase) error
        +SessionExists(name) bool
    }

    class ClaudeExecutor {
        +ExecuteInTmux(config, vars, session, window, path) error
        +ExecuteCommand(config, vars) CommandResult
    }

    GitHubClient --> LabelManager
    IssueWatcher --> GitHubClient
    ActionManager --> WorktreeManager
    ActionManager --> TmuxManager  
    ActionManager --> ClaudeExecutor
```

## データフロー

### Issue検知からラベル遷移までの詳細フロー

```mermaid
flowchart TD
    START([開始]) --> POLL[GitHub API ポーリング]
    POLL --> CHECK{新しいIssue?}
    CHECK -->|No| WAIT[ポーリング間隔待機]
    WAIT --> POLL
    CHECK -->|Yes| VALIDATE[Issue検証]
    VALIDATE --> LABEL{ラベル判定}
    
    LABEL -->|status:needs-plan| PLAN[PlanAction実行]
    LABEL -->|status:ready| IMPL[ImplementationAction実行]
    LABEL -->|status:review-requested| REVIEW[ReviewAction実行]
    LABEL -->|その他| WAIT
    
    PLAN --> PLAN_STEPS[計画フェーズ処理]
    IMPL --> IMPL_STEPS[実装フェーズ処理]
    REVIEW --> REVIEW_STEPS[レビューフェーズ処理]
    
    PLAN_STEPS --> PLAN_TRANSITION[ラベル遷移: needs-plan → ready]
    IMPL_STEPS --> IMPL_TRANSITION[ラベル遷移: ready → review-requested]
    REVIEW_STEPS --> REVIEW_TRANSITION[ラベル遷移: review-requested → completed]
    
    PLAN_TRANSITION --> WAIT
    IMPL_TRANSITION --> WAIT
    REVIEW_TRANSITION --> WAIT

    subgraph PLAN_STEPS [計画フェーズ処理]
        P1[tmux window作成]
        P2[main branch更新]
        P3[worktree作成]
        P4[Claude計画実行]
        P1 --> P2 --> P3 --> P4
    end

    subgraph IMPL_STEPS [実装フェーズ処理]
        I1[tmux window作成]
        I2[main branch更新]
        I3[worktree作成]
        I4[Claude実装実行]
        I1 --> I2 --> I3 --> I4
    end

    subgraph REVIEW_STEPS [レビューフェーズ処理]
        R1[tmux window作成]
        R2[worktree準備]
        R3[Claudeレビュー実行]
        R4[PR作成]
        R1 --> R2 --> R3 --> R4
    end
```

### エラーハンドリングフロー

```mermaid
flowchart TD
    ACTION[アクション実行] --> SUCCESS{成功?}
    SUCCESS -->|Yes| COMPLETE[処理完了]
    SUCCESS -->|No| ERROR[エラー発生]
    
    ERROR --> RECOVERABLE{回復可能?}
    RECOVERABLE -->|Yes| RETRY[リトライ実行]
    RECOVERABLE -->|No| FAIL[処理失敗]
    
    RETRY --> RETRY_CHECK{リトライ回数チェック}
    RETRY_CHECK -->|上限内| ACTION
    RETRY_CHECK -->|上限超過| FAIL
    
    FAIL --> CLEANUP[リソースクリーンアップ]
    CLEANUP --> LOG[エラーログ記録]
    LOG --> STATE[状態更新: Failed]
    STATE --> NOTIFY[通知送信]
    
    COMPLETE --> STATE_SUCCESS[状態更新: Completed]
    STATE_SUCCESS --> NEXT[次フェーズ準備]
```

## 状態管理

### Issue状態遷移

```mermaid
stateDiagram-v2
    [*] --> Detected: Issue検知
    Detected --> Planning: PlanAction開始
    Planning --> Planned: 計画完了
    Planned --> Implementing: ImplementationAction開始
    Implementing --> Implemented: 実装完了
    Implemented --> Reviewing: ReviewAction開始
    Reviewing --> Completed: レビュー完了
    Completed --> [*]: 処理終了
    
    Planning --> Failed: エラー発生
    Implementing --> Failed: エラー発生
    Reviewing --> Failed: エラー発生
    Failed --> [*]: 処理終了
```

### ラベル遷移マッピング

| 現在のラベル | 実行アクション | 遷移先ラベル | 条件 |
|-------------|---------------|-------------|------|
| `status:needs-plan` | PlanAction | `status:planning` → `status:ready` | 計画フェーズ完了 |
| `status:ready` | ImplementationAction | `status:implementing` → `status:review-requested` | 実装フェーズ完了 |
| `status:review-requested` | ReviewAction | `status:reviewing` → 完了 | レビューフェーズ完了 |

## セキュリティ考慮事項

### 1. 認証・認可

- **GitHub Token**: 環境変数または設定ファイルで管理
- **Claude API Key**: 暗号化して保存
- **最小権限の原則**: 必要最小限のスコープのみ

### 2. 入力検証

- **Issue内容の検証**: XSS、インジェクション攻撃防止
- **ラベル名の検証**: 予期しないラベル操作防止
- **コマンド実行の検証**: 任意コマンド実行防止

### 3. ログ・監査

- **機密情報のマスキング**: トークン、APIキーの自動マスキング
- **操作ログ**: 全ての重要操作をログ記録
- **アクセス制御**: ログファイルのアクセス権限管理

## パフォーマンス設計

### 1. 並行処理

- **goroutine活用**: 複数Issue並行処理
- **チャネル通信**: 安全な状態同期
- **コンテキスト制御**: タイムアウト・キャンセル処理

### 2. リソース管理

- **GitHub API レート制限**: 自動調整機能
- **tmux セッション**: 自動クリーンアップ
- **git worktree**: 容量監視・自動削除

### 3. キャッシュ戦略

- **Issue情報**: 短期メモリキャッシュ
- **GitHub APIレスポンス**: 適切なTTL設定
- **テンプレート**: 初回読み込み後キャッシュ

## 拡張性設計

### 1. プラグインアーキテクチャ

```mermaid
classDiagram
    class PluginManager {
        +LoadPlugin(path) Plugin
        +RegisterPlugin(plugin) error
        +ExecuteHook(event, data) error
    }

    class Plugin {
        <<interface>>
        +Initialize() error
        +Execute(context) result
        +Cleanup() error
    }

    class CustomAction {
        +CanExecute(issue) bool
        +Execute(ctx, issue) error
    }

    PluginManager --> Plugin
    Plugin <|-- CustomAction
```

### 2. 設定システム

- **階層的設定**: プロジェクト → ユーザー → システム
- **動的リロード**: 設定変更の即座反映
- **検証機能**: 設定値の妥当性確認

### 3. 外部システム連携

- **Webhook対応**: GitHub Events即座反映
- **Slack/Discord**: 通知システム拡張
- **CI/CD統合**: Jenkins、GitHub Actions連携

## テスト戦略

### 1. テストピラミッド

```mermaid
pyramid
    top: E2E Tests
    middle: Integration Tests  
    bottom: Unit Tests
```

- **Unit Tests**: 各コンポーネントの単体テスト
- **Integration Tests**: コンポーネント間結合テスト
- **E2E Tests**: エンドツーエンドシナリオテスト

### 2. テスト環境

- **Mock使用**: 外部API依存性の排除
- **Test Containers**: 統合テスト環境
- **CI/CD**: 自動テスト実行

### 3. テストデータ管理

- **Fixture**: 一貫したテストデータ
- **Factory Pattern**: テストオブジェクト生成
- **Cleanup**: テスト後の環境リセット

## 運用・監視

### 1. ヘルスチェック

- **システム死活監視**: プロセス・リソース監視
- **API応答監視**: GitHub API、Claude API監視
- **パフォーマンス監視**: レスポンス時間・スループット

### 2. ログ管理

- **構造化ログ**: JSON形式での出力
- **ログレベル**: Debug、Info、Warn、Error
- **ログローテーション**: サイズ・期間ベース

### 3. メトリクス

- **処理統計**: 成功率、処理時間、エラー率
- **リソース使用量**: CPU、メモリ、ディスク
- **ビジネスメトリクス**: Issue処理数、フェーズ完了率

## 今後の拡張計画

### Phase 2: スケーラビリティ向上

- **分散処理**: 複数インスタンス並行実行
- **マイクロサービス化**: 機能別サービス分割
- **イベント駆動アーキテクチャ**: 非同期処理基盤

### Phase 3: AI機能強化

- **コンテキスト学習**: プロジェクト固有の学習
- **予測機能**: Issue複雑度・工数予測
- **最適化機能**: 処理順序・リソース配分最適化

### Phase 4: エコシステム構築

- **マーケットプレイス**: サードパーティプラグイン
- **SaaS化**: クラウドサービス提供
- **エンタープライズ機能**: 権限管理・監査ログ強化

## まとめ

osobaのアーキテクチャは、モジュラー設計により高い拡張性と保守性を実現しています。各レイヤーの責務分離により、個別のコンポーネント変更が他への影響を最小限に抑制し、継続的な機能拡張を可能にしています。

また、エラーハンドリング、セキュリティ、パフォーマンスの各観点から設計されており、本番環境での安定運用を重視した構成となっています。