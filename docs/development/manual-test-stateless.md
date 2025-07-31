# ステートレス動作の手動テスト手順書

本ドキュメントでは、osobaのステートレス動作を手動で検証するための手順を説明します。

## 前提条件

- GitHubのテストリポジトリへのアクセス権限
- osobaが正常に起動している状態
- GitHub CLIがインストールされていること

## テストシナリオ

### 1. ラベル手動変更時の再処理確認

#### 1.1 status:implementing → status:ready への手動変更

**目的**: 開発者が実装を一時中断し、再度実装を開始する場合の動作確認

**手順**:
1. 対象Issueが `status:implementing` ラベルを持つ状態を確認
   ```bash
   gh issue view <issue-number>
   ```

2. osobaのログを監視開始
   ```bash
   # 別ターミナルでログを確認
   tail -f osoba.log | grep "issue <issue-number>"
   ```

3. ラベルを手動で変更
   ```bash
   gh issue edit <issue-number> \
     --remove-label "status:implementing" \
     --add-label "status:ready"
   ```

4. osobaが次のポーリングサイクル（通常30秒以内）で以下を実行することを確認：
   - `status:ready` ラベルを検出
   - `status:ready` を削除し、`status:implementing` を追加
   - 実装アクションを再実行

**期待される結果**:
- osobaが自動的にラベルを `status:implementing` に戻す
- 実装アクションが再実行される
- ログに「Trigger label 'status:ready' found without corresponding execution label」が出力される

#### 1.2 status:reviewing → status:review-requested への手動変更

**目的**: レビュアーがレビューを中断し、再度レビューを開始する場合の動作確認

**手順**:
1. 対象Issueが `status:reviewing` ラベルを持つ状態を確認
2. osobaのログを監視開始
3. ラベルを手動で変更
   ```bash
   gh issue edit <issue-number> \
     --remove-label "status:reviewing" \
     --add-label "status:review-requested"
   ```
4. osobaの動作を確認

**期待される結果**:
- osobaが自動的にラベルを `status:reviewing` に戻す
- レビューアクションが再実行される

#### 1.3 status:planning → status:needs-plan への手動変更

**目的**: 計画を見直すために一時的に戻す場合の動作確認

**手順**:
1. 対象Issueが `status:planning` ラベルを持つ状態を確認
2. osobaのログを監視開始
3. ラベルを手動で変更
   ```bash
   gh issue edit <issue-number> \
     --remove-label "status:planning" \
     --add-label "status:needs-plan"
   ```
4. osobaの動作を確認

**期待される結果**:
- osobaが自動的にラベルを `status:planning` に戻す
- 計画アクションが再実行される

### 2. エラーリカバリーの確認

#### 2.1 ネットワークエラー時の自動リトライ

**目的**: 一時的なネットワークエラーからの自動回復を確認

**手順**:
1. ネットワークを一時的に切断（またはGitHub APIのレート制限に達する）
2. osobaのログでエラーとリトライを確認
   ```bash
   tail -f osoba.log | grep -E "(error|retry|rate limit)"
   ```
3. ネットワークを復旧（またはレート制限解除を待つ）

**期待される結果**:
- エラー発生時に自動的にリトライが実行される
- 最大3回のリトライが行われる
- ネットワーク復旧後、正常に処理が再開される

### 3. 複数インスタンスの動作確認

**注意**: 現在の実装では完全な多重実行防止は提供されていません。

**手順**:
1. 2つ以上のosobaインスタンスを異なるセッションIDで起動
2. 同じIssueに対してトリガーラベルを設定
3. 各インスタンスのログを確認

**期待される結果**:
- 複数のインスタンスが同じIssueを処理する可能性がある
- ラベル遷移は冪等性により最終的に正しい状態になる

## トラブルシューティング

### ラベルが期待通りに遷移しない場合

1. **ログを確認**
   ```bash
   grep "ShouldProcessIssue" osoba.log
   ```

2. **現在のラベル状態を確認**
   ```bash
   gh issue view <issue-number> --json labels
   ```

3. **手動でラベルをリセット**
   ```bash
   # すべての status: ラベルを削除
   gh issue edit <issue-number> \
     --remove-label "status:needs-plan" \
     --remove-label "status:planning" \
     --remove-label "status:ready" \
     --remove-label "status:implementing" \
     --remove-label "status:review-requested" \
     --remove-label "status:reviewing"
   
   # 初期ラベルを追加
   gh issue edit <issue-number> --add-label "status:needs-plan"
   ```

### ログレベルの変更

詳細なデバッグ情報が必要な場合：

```bash
# 環境変数でログレベルを設定
export LOG_LEVEL=debug
./osoba
```

## 検証チェックリスト

- [ ] ラベル手動変更時の再処理が正常に動作する
- [ ] エラー発生時の自動リトライが機能する
- [ ] ログに適切な情報が出力される
- [ ] 最終的にIssueが正しい状態に収束する

## 注意事項

- ポーリング間隔により、ラベル変更から処理開始まで最大30秒程度かかる場合があります
- テスト環境では本番環境と異なる設定（ポーリング間隔など）を使用することを推奨します
- GitHub APIのレート制限に注意してください