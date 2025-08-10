# feat: auto_plan_issue機能の実装

## 概要
Issue #210 の要求に基づき、処理中のIssueがない場合に自動的に次のIssueを`status:needs-plan`状態に移行させるauto_plan_issue機能を実装しました。

## 変更内容

### 📋 新機能
- **auto_plan_issue設定**: 設定ファイルで機能のON/OFF切り替えが可能
- **自動ラベル付与**: status:*ラベルが付いていない最も若い番号のIssueに`status:needs-plan`ラベルを自動付与
- **watcherサイクル統合**: 既存のwatcher処理サイクルの最後に実行

### 🔧 実装詳細

#### 設定関連
- `internal/config/config.go`: `AutoPlanIssue`フィールドを`GitHubConfig`に追加（デフォルト: `false`）
- `cmd/templates/config.yml`: 設定テンプレートにコメント付きで設定項目追加

#### GitHub API操作
- `internal/gh/list_issues.go`: `ListAllOpenIssues`メソッド追加
- `internal/github/client.go`: `ListAllOpenIssues`メソッド追加と`convertMapToIssue`ヘルパー関数実装
- `internal/github/interface.go`: インターフェースに`ListAllOpenIssues`メソッド追加

#### 自動計画ロジック  
- `internal/watcher/auto_plan.go`: 
  - `executeAutoPlanIfNoActiveIssues`: メインロジック
  - `findLowestNumberIssueWithoutStatusLabel`: 最も若い番号のラベルなしIssueを特定
  - `hasStatusLabel`: status:*ラベル判定
  - エラーハンドリングと詳細ログ出力

#### watcher統合
- `internal/watcher/watcher.go`: `checkIssues`メソッドの最後にauto_plan実行を追加

### 🧪 テスト
- `internal/watcher/auto_plan_test.go`: 包括的テストスイート
- `internal/config/config_test.go`: 設定テスト  
- `internal/testutil/mocks/github.go`: モッククライアント拡張

## テスト結果
```
=== RUN   TestExecuteAutoPlanIfNoActiveIssues
=== RUN   TestExecuteAutoPlanIfNoActiveIssues/正常系:_status:*ラベルがない場合、最も若い番号のIssueにラベル付与
=== RUN   TestExecuteAutoPlanIfNoActiveIssues/正常系:_auto_plan_issue設定が無効の場合はスキップ  
=== RUN   TestExecuteAutoPlanIfNoActiveIssues/正常系:_status:*ラベル付きIssueが存在する場合はスキップ
=== RUN   TestExecuteAutoPlanIfNoActiveIssues/正常系:_ラベルなしIssueが存在しない場合はスキップ
=== RUN   TestExecuteAutoPlanIfNoActiveIssues/異常系:_GitHub_API呼び出し失敗
=== RUN   TestExecuteAutoPlanIfNoActiveIssues/異常系:_ラベル付与失敗
--- PASS: TestExecuteAutoPlanIfNoActiveIssues (0.00s)

=== RUN   TestAutoPlanIssueConfig  
--- PASS: TestAutoPlanIssueConfig (0.00s)
```

## 動作例
1. **設定有効時**: watcherが処理中のIssue（status:*ラベル付き）がないことを確認
2. **Issue検索**: 全てのオープンIssueから、status:*ラベルが付いていない最も若い番号のIssueを特定  
3. **ラベル付与**: 対象Issueに`status:needs-plan`ラベルを自動付与
4. **ログ出力**: 処理結果を詳細ログに記録

## 🎯 要求仕様との対応
- ✅ 設定による機能のON/OFF切り替え（`auto_plan_issue`、デフォルト`false`）
- ✅ status:*ラベルが付いていない最も若い番号のIssueへの自動ラベル付与
- ✅ 既存`auto_merge_lgtm`機能と同じパターンでの実装
- ✅ watcherサイクルとの統合

## 🚀 Breaking Changes
なし。デフォルトで機能は無効のため、既存の動作に影響なし。

## 📝 Notes
- TDD（テスト駆動開発）アプローチで実装
- 既存`auto_merge_lgtm`機能のパターンを踏襲
- エラーハンドリングとロバストネスに配慮
- 詳細なテストカバレッジ