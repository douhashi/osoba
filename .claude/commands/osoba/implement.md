---
allowed-tools: Bash, Read, Write, Edit, MultiEdit, Grep, Glob, LS
description: "TDDによる実装作業"
---

# 実装

開発者として、引数で指定されたIssueをTDD（テスト駆動開発）で実装する

## ワークフロー

1. **Issue確認**
   - 引数で指定されたIssue番号をghコマンドで確認
   - `GH_PAGER= gh issue view <issue番号>` でIssue内容を取得
   - 実装内容と受け入れ条件を把握

2. **TDDによる実装**
   - テストを先に作成
   - テストが失敗することを確認
   - 最小限の実装でテストをパス
   - リファクタリング

3. **テスト実行**
   - 実装部分のテストを実行
   - システム全体のフルテストを実行

4. **PR作成**
   - 全テストがパスしたらPRを作成

5. **完了報告**
   - Issueにコメントで完了を報告
   - `GH_PAGER= gh issue comment <issue番号> --body "実装が完了しました。PRを作成しました: #<PR番号>"`
   - `GH_PAGER= gh issue edit <issue番号>  --remove-label "status:implementing" --add-label "status:review-requested"` でラベルを付与/削除

## 基本ルール

- テストファーストで実装を進める
- 既存のコーディング規約に従う
- 全てのテストがパスすることを確認
- セキュリティを考慮した実装を行う
