# internal/gh 未実装機能詳細

## 概要
internal/ghパッケージに未実装の機能と、internal/githubから移植が必要な実装の詳細リスト。

## 未実装メソッド一覧

### 1. Pull Request関連機能（優先度: 高）

#### GetPullRequestForIssue
- **機能**: IssueからPRを検索する
- **internal/github実装**: 複数の検索戦略を使用（GraphQL、検索API、ブランチパターン）
- **必要なghコマンド**: 
  ```bash
  gh pr list --search "linked:issue-NUMBER"
  gh api graphql -f query='...'
  ```

#### MergePullRequest
- **機能**: PRをマージする
- **internal/github実装**: マージ可能性チェック、自動マージ
- **必要なghコマンド**:
  ```bash
  gh pr merge NUMBER --merge
  ```

#### GetPullRequestStatus
- **機能**: PRのステータスを取得
- **internal/github実装**: マージ状態、レビュー状態、CI状態の確認
- **必要なghコマンド**:
  ```bash
  gh pr view NUMBER --json state,mergeable,reviews,statusCheckRollup
  ```

#### ListPullRequestsByLabels
- **機能**: ラベルでPRをフィルタリング
- **internal/github実装**: GraphQLでの効率的な取得
- **必要なghコマンド**:
  ```bash
  gh pr list --label "label1,label2" --json number,title,state
  ```

#### GetClosingIssueNumber
- **機能**: PRが閉じるIssue番号を取得
- **internal/github実装**: PR本文解析、リンク解析
- **必要なghコマンド**:
  ```bash
  gh pr view NUMBER --json body,closingIssuesReferences
  ```

### 2. ラベル一括操作機能（優先度: 中）

#### TransitionLabels
- **機能**: 原子的なラベル遷移（削除と追加を同時実行）
- **internal/github実装**: トランザクション的な処理
- **必要なghコマンド**:
  ```bash
  # 現在は個別実行が必要
  gh issue edit NUMBER --remove-label "old" 
  gh issue edit NUMBER --add-label "new"
  ```
- **課題**: ghコマンドでは原子性が保証されない

## 実装の複雑度分析

### 複雑度: 高
1. **GetPullRequestForIssue**
   - 複数の検索戦略が必要
   - GraphQL実装が複雑
   - フォールバック処理

2. **GetClosingIssueNumber**
   - PR本文のパース処理
   - 複数の記法への対応

### 複雑度: 中
1. **ListPullRequestsByLabels**
   - ページネーション処理
   - 複数ラベルのAND/OR検索

2. **GetPullRequestStatus**
   - 複数ステータスの統合
   - CI状態の解析

### 複雑度: 低
1. **MergePullRequest**
   - 単純なコマンド実行
   - エラーハンドリングのみ

2. **TransitionLabels**
   - 2つのコマンドの連続実行
   - エラー時のロールバック

## 内部ヘルパー機能の移植

### 1. エラーハンドリング
- **internal/github/errors.go**: カスタムエラー型
- **internal/github/error_parser.go**: GitHub APIエラー解析
- **移植の必要性**: ghコマンドのエラー出力解析

### 2. リトライ機構
- **internal/github/retry_strategy.go**: 指数バックオフ
- **internal/github/label_manager.go**: ラベル操作のリトライ
- **移植の必要性**: API制限対応

### 3. ロギング
- **internal/github/logging_transport.go**: HTTP通信ログ
- **移植の必要性**: ghコマンド実行ログ

## テストカバレッジギャップ

### 現状
- internal/github: 76個のテスト関数
- internal/gh: 16個のテスト関数
- **ギャップ**: 60個のテストが不足

### 移植が必要なテスト
1. PR関連テスト（約20個）
2. エラーハンドリングテスト（約10個）
3. リトライ機構テスト（約8個）
4. 統合テスト（約15個）

## 実装優先順位

### Phase 1（必須）
1. GetPullRequestForIssue
2. MergePullRequest
3. GetPullRequestStatus
4. GetClosingIssueNumber

### Phase 2（重要）
1. ListPullRequestsByLabels
2. TransitionLabels
3. エラーハンドリング強化

### Phase 3（改善）
1. リトライ機構
2. 詳細ロギング
3. パフォーマンス最適化