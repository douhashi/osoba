---
allowed-tools: Bash, Read, Grep, Glob, LS
description: "実装計画の作成"
---

# 計画

引数で指定されたIssueの実装計画を作成します。
実行計画とは、 **詳細な実装手順が記載されたGithubのコメントを作成すること** です。

## ワークフロー

1. **Issue確認**
   - 引数で指定されたIssue番号をghコマンドで確認
   - `GH_PAGER= gh issue view <issue番号>` でIssue内容を取得
   - 要件と背景を把握

2. **現状分析**
   - 関連するコードを調査
   - 既存の実装パターンを確認
   - 影響範囲を特定

3. **実装計画作成**
   - 技術的なアプローチを決定
   - 必要なテストケースを洗い出し
   - 作業手順を具体化

4. **計画の記録**
   - Issueにコメントとして実装計画を投稿
   - `GH_PAGER= gh issue comment <issue番号> --body "実装計画: ..."`

5. **ラベル更新**
   - `GH_PAGER= gh issue edit <issue番号> --remove-label "status:planning" --add-label "status:ready"` でラベルを付与

## 基本ルール

- コードベースを書き換えてはいけません。
- TDDに適した計画を立てる
- 既存のアーキテクチャに沿った設計
- テスト可能な実装方針を選択
- 段階的な実装が可能な計画にする