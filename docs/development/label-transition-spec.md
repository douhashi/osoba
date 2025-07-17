# ラベル遷移仕様書

このドキュメントは、osoba プロジェクトにおける GitHub Issue のラベル遷移仕様を定義します。

## 概要

osoba は GitHub Issue のラベルを使用して、開発フローの各フェーズを管理します。各フェーズの開始時に適切なラベル遷移を実行することで、Issue の状態を正確に追跡します。

## ラベル遷移の実装方式

### 一元管理による実装

ラベル遷移は Issue Check Cycle の最後に一元的に実行されます。これにより以下のメリットがあります：

1. **確実性**: アクションの実行結果に関わらず、必ずラベル遷移が実行される
2. **保守性**: ラベル遷移ロジックが一箇所に集約される
3. **デグレード防止**: 各アクションの実装変更がラベル遷移に影響しない

### 実装箇所

- ファイル: `internal/watcher/watcher.go`
- メソッド: `StartWithActions` → `executeLabelTransition`

```go
// StartWithActions内でアクション実行後に必ず実行
if err := w.executeLabelTransition(ctx, issue); err != nil {
    log.Printf("Failed to execute label transition for issue #%d: %v", *issue.Number, err)
}
```

## ラベル遷移パターン

### 1. 計画フェーズ (Plan Phase)

| 遷移元ラベル | 遷移先ラベル | トリガー条件 |
|-------------|-------------|------------|
| `status:needs-plan` | `status:planning` | Issue に `status:needs-plan` ラベルが存在する |

### 2. 実装フェーズ (Implementation Phase)

| 遷移元ラベル | 遷移先ラベル | トリガー条件 |
|-------------|-------------|------------|
| `status:ready` | `status:implementing` | Issue に `status:ready` ラベルが存在する |

### 3. レビューフェーズ (Review Phase)

| 遷移元ラベル | 遷移先ラベル | トリガー条件 |
|-------------|-------------|------------|
| `status:review-requested` | `status:reviewing` | Issue に `status:review-requested` ラベルが存在する |

## ラベル遷移のタイミング

1. **Issue 検知時**: watcher が Issue を検知
2. **アクション実行**: 対応するアクションを実行（成功/失敗/スキップに関わらず）
3. **ラベル遷移**: `executeLabelTransition` メソッドでラベル遷移を実行

## エラーハンドリング

### ラベル遷移失敗時の動作

- ラベル遷移に失敗してもプロセスは継続される
- エラーはログに記録される
- 次の Issue Check Cycle で再度遷移を試みる

### GitHub API エラー

- ラベル削除失敗: `failed to remove label %s: %w`
- ラベル追加失敗: `failed to add label %s: %w`

## テスト戦略

### 単体テスト

- ファイル: `internal/watcher/label_transition_test.go`
- テスト項目:
  - 各フェーズのラベル遷移
  - エラーケース（nil issue、API エラー）
  - 遷移不要なケース

### 統合テスト

- ファイル: `internal/watcher/integration_test.go`
- テスト項目:
  - 処理済み Issue でもラベル遷移が実行されること
  - Issue Check Cycle 全体でのラベル遷移動作

## デグレード防止のガイドライン

### 1. 実装時の注意事項

- **禁止事項**: 各アクション内でラベル遷移を実装しない
- **必須事項**: ラベル遷移は `executeLabelTransition` メソッドでのみ実装

### 2. コードレビューのチェックポイント

- [ ] アクション内にラベル遷移処理が含まれていないか
- [ ] 新しいフェーズを追加する場合、`executeLabelTransition` に遷移パターンを追加しているか
- [ ] テストケースが追加されているか

### 3. 新しいフェーズ追加時の手順

1. `executeLabelTransition` メソッドの `transitions` 配列に新しい遷移パターンを追加
2. `label_transition_test.go` に単体テストケースを追加
3. 必要に応じて統合テストを更新

## 今後の拡張性

### 複雑な遷移パターン

現在の実装は単純な 1:1 のラベル遷移のみサポートしていますが、将来的には以下の拡張が可能です：

- 条件付き遷移（複数の条件を満たした場合のみ遷移）
- 複数ラベルの同時遷移
- カスタム遷移ロジックのプラグイン化

### 設定ファイルによる管理

将来的にラベル遷移パターンを設定ファイルで管理することも検討できます：

```yaml
label_transitions:
  - from: "status:needs-plan"
    to: "status:planning"
    phase: "plan"
  - from: "status:ready"
    to: "status:implementing"
    phase: "implementation"
```

## 関連ドキュメント

- [プロジェクト概要](project-brief.md)
- [Git/GitHub 運用ルール](git-instructions.md)
- [Go コーディング規約](go-coding-standards.md)