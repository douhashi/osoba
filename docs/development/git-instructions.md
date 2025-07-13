# Git/Github: ブランチ運用とコミットルール

このドキュメントは、プロジェクトにおけるGitの運用ルールとコミット規約をまとめたものです。

## Git Worktree環境

現在のディレクトリが `.git/worktree` 以下にある場合は、git worktree機能を使った作業環境です。
現在のディレクトリ (例: `.git/worktree/issue-12`) が、コードベースなので、現在のディレクトリより上位のファイルを書き換えないようにしてください。

## ブランチ運用

### ブランチ命名規則

ブランチ名は以下の形式に従ってください：

```
<prefix>/#<issue番号>-<簡潔な説明>
```

#### 接頭辞（prefix）の種類

| 接頭辞 | 用途 | 例 |
|--------|------|-----|
| `feat` | 新機能追加 | `feat/#123-user-authentication` |
| `fix` | バグ修正 | `fix/#456-login-error` |
| `docs` | ドキュメント更新 | `docs/#789-api-documentation` |
| `style` | スタイル調整（ロジックに影響しない変更） | `style/#234-button-design` |
| `refactor` | リファクタリング | `refactor/#567-payment-service` |
| `test` | テスト追加・修正 | `test/#890-user-model-specs` |
| `chore` | 雑務的な変更（ビルド、設定など） | `chore/#345-update-dependencies` |

### ブランチ作成コマンド

```bash
# 新機能追加
git checkout -b feat/#<issue番号>-<機能名>

# バグ修正
git checkout -b fix/#<issue番号>-<バグ名>

# その他の例
git checkout -b docs/#<issue番号>-<ドキュメント名>
git checkout -b style/#<issue番号>-<対象名>
git checkout -b refactor/#<issue番号>-<対象名>
git checkout -b test/#<issue番号>-<テスト名>
git checkout -b chore/#<issue番号>-<作業名>
```

## コミットルール

### コミットは意味のある最小の単位で行う

コミットタイミングの例:

1. 新しい機能やモジュールの実装が完了したとき
2. テストが完了して、テストがパスしたとき
3. ユーティリティ関数やヘルパー機能の実装が完了したとき
4. 設定ファイルの作成・更新が完了したとき
5. ドキュメントの更新が完了したとき

### コミット前の確認事項

```bash
# コミット前にコードスタイルを整える（Pythonの場合）
# black . (フォーマッター)
# flake8 . (リンター)

# テスト実行
# pytest

# コミット
git add .
git commit -m "<type>: コミットメッセージ"
```

### コミットメッセージ形式

コミットメッセージは以下の形式に従ってください：

```
<type>: <subject>

[optional body]

[optional footer]
```

### コミットタイプ

| タイプ | 説明 | 絵文字 |
|--------|------|--------|
| `feat` | 新機能追加 | 🚀 |
| `fix` | バグ修正 | 🐛 |
| `docs` | ドキュメント更新 | 📚 |
| `style` | スタイル調整（コードの意味に影響しない変更） | 💅 |
| `refactor` | リファクタリング（機能追加やバグ修正を含まない） | ♻️ |
| `test` | テスト追加・修正 | 🧪 |
| `chore` | 雑務的な変更（ビルド、ツール、ライブラリなど） | 🔧 |

### コミット例

```bash
# 新機能追加
git commit -m "feat: ユーザー認証機能を追加"

# バグ修正
git commit -m "fix: CLIコマンドの引数解析エラーを修正"

# ドキュメント更新
git commit -m "docs: READMEにセットアップ手順を追加"

# リファクタリング
git commit -m "refactor: CLIコマンドをクラスベースに再設計"

# テスト追加
git commit -m "test: CLIコマンドの統合テストを追加"

# 設定ファイル更新
git commit -m "chore: 設定ファイルに新しいオプションを追加"

# 依存関係の更新
git commit -m "chore: requirements.txtの依存関係を更新"
```

## セキュリティ上の注意事項

### 絶対に操作してはいけないファイル

以下のファイルは機密情報を含むため、絶対に操作しないでください：

- `.env` ファイル（環境変数）
- `.env.local`、`.env.production` などの環境固有ファイル
- `config/credentials.yml.enc`（暗号化された認証情報）
- `config/master.key`（復号化キー）
- `*.pem`、`*.key` ファイル（証明書、秘密鍵）
- `vendor/` ディレクトリ（外部ライブラリ）

### コミット時の注意

- APIキー、アクセストークンなどの機密情報をハードコーディングしない
- 機密情報が含まれていないか必ず確認してからコミットする
- 誤って機密情報をコミットした場合は、即座に指示者に報告する


## トラブルシューティング

### コンフリクトの解決

```bash
# 最新のmainブランチを取得
git fetch origin main

# mainブランチの変更を現在のブランチにマージ
git merge origin/main

# コンフリクトを解決後
git add .
git commit -m "fix: mainブランチとのコンフリクトを解決"
```

## 参考資料

- [Conventional Commits](https://www.conventionalcommits.org/)
- [GitHub Flow](https://docs.github.com/en/get-started/quickstart/github-flow)
