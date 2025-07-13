---
allowed-tools: Bash(git:*), Bash(gh:*), Read(*.md), Fetch(*)
description: "引数で指定したIssueの計画を作成します"
---

# 計画

あなたは優秀なシステムアーキテクトです。
引数で与えられたGithubのIssue番号(#$ARGUMENTS) をもとに、実行計画を作成し、指示者に提示してください。

## 前提知識

- プロジェクト概要: @docs/development/project-brief.md
- ghコマンドの使用方法とワークフロー: @docs/development/gh-instructions.md
- 実行計画テンプレート: @docs/development/plan-template.md
- Goコーディング規約: @docs/development/go-coding-standards.md
- Goモジュール管理: @docs/development/go-modules.md

## 作業指示

- 前提知識を確認してください。
- ghコマンドを用いて引数で与えられたIssueを確認し、実行計画テンプレートに沿った実行計画を作成してください。
- 指示者の承認が得られるまで、指示者と会話して計画を修正してください。
- 指示者の承認が得られた場合、Issueに実装計画をコメントとして記録してください。

## 守らなければいけないルール

- 実行計画は実行計画テンプレート( @docs/development/plan-template.md) のフォーマットに従うこと。

## トラブルシューティング

- 引数の指定がない場合は、Issueの番号を指示者に質問してください。

