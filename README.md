```
                     _           
   ___  ___  ___   | |__    __ _ 
  / _ \/ __|/ _ \  | '_ \  / _` |
 | (_) \__ \ (_) | | |_) || (_| |
  \___/|___/\___/  |_.__/  \__,_|
                                 
```

# osoba - 自律的ソフトウェア開発支援ツール

[![CI](https://github.com/douhashi/osoba/actions/workflows/ci.yml/badge.svg)](https://github.com/douhashi/osoba/actions/workflows/ci.yml)
[![Release](https://github.com/douhashi/osoba/actions/workflows/release.yml/badge.svg)](https://github.com/douhashi/osoba/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/douhashi/osoba)](https://goreportcard.com/report/github.com/douhashi/osoba)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 概要

osobaは、tmux + git worktree + Claude を統合した自律的なソフトウェア開発支援CLIツールです。GitHub Issueをトリガーとして、AIが計画・実装・レビューの各フェーズを自律的に実行し、開発プロセスを大幅に効率化します。

### 主な特徴

- 🤖 **自律的な開発フロー**: GitHub Issueのラベルに基づいた自動的なタスク実行
- 🖥️ **tmuxセッション管理**: リポジトリ・Issue単位での独立した開発環境
- 🌳 **git worktree統合**: Issueごとの独立したブランチとワークツリー
- 🧠 **Claude AI統合**: フェーズごとに最適化されたプロンプト実行
- 🔄 **継続的な監視**: Issueを監視し、自動的にアクションを実行

## 必要な環境

- tmux 3.0以上
- git 2.x以上
- GitHub CLI（gh）
- Claude CLI（claude）

### GitHub認証

osobaはデフォルトでGitHub CLI（gh）を使用してGitHubにアクセスします。事前にghでログインしてください：

```bash
gh auth login
```

## インストール

### リリース版のインストール

最新のリリース版は[GitHub Releases](https://github.com/douhashi/osoba/releases)からダウンロードできます。

```bash
# Linux (amd64)
curl -L https://github.com/douhashi/osoba/releases/latest/download/osoba_Linux_x86_64.tar.gz | tar xz
sudo mv osoba /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/douhashi/osoba/releases/latest/download/osoba_Darwin_arm64.tar.gz | tar xz
sudo mv osoba /usr/local/bin/
```

### ソースからのビルド

```bash
# リポジトリのクローン
git clone https://github.com/douhashi/osoba.git
cd osoba

# ビルドとインストール
make install
# または
go install
```

## クイックスタート

### 1. 初期設定

```bash
# GitHubにログイン（未ログインの場合）
gh auth login

# osobaの初期設定を実行
osoba init

※ .claude/commands 以下に osoba 用のコマンドが生成されます
```

### 2. 基本的な使い方

```bash
# リポジトリでosobaを開始
cd /path/to/your/repo
osoba start

# 別のターミナルを開き、セッションに接続
osoba open
```

### 3. ワークフロー例

1. GitHub Issueを作成し、`status:needs-plan`ラベルを付与
2. osobaが自動的にIssueを検知し、計画フェーズを実行
3. 計画完了後、`status:ready`ラベルに更新
4. 実装フェーズが自動的に開始
5. `osoba open`でセッションに接続して進捗を確認

### 4. リソースのクリーンアップ

```bash
# 特定のIssueに関連するリソースを削除
osoba clean 83

# 全てのIssue関連リソースを削除（確認プロンプトあり）
osoba clean --all
```

## 動作イメージ

### ラベル遷移と自動実行フロー

```mermaid
flowchart LR
    A[GitHub Issue作成] --> B[status:needs-plan]
    B --> C{osoba監視}
    C -->|検知| D[計画フェーズ]
    D --> E[実行計画投稿]
    E --> F[status:ready]
    
    F --> G{osoba監視}
    G -->|検知| H[実装フェーズ]
    H --> I[PR作成]
    I --> J[status:review-requested]
    
    J --> K{osoba監視}
    K -->|検知| L[レビューフェーズ]
    L --> M[コードレビュー完了]
    
    subgraph plan [計画フェーズ]
        D1[tmuxウィンドウ作成]
        D2[git worktree作成]
        D3[Claude実行]
        D1 --> D2 --> D3
    end
    
    subgraph implement [実装フェーズ]
        H1[コード実装]
        H2[テスト実行]
        H3[PR作成]
        H1 --> H2 --> H3
    end
    
    subgraph review [レビューフェーズ]
        L1[コードレビュー]
        L1
    end
    
    D -.-> plan
    H -.-> implement
    L -.-> review
```

### 各フェーズの詳細

#### 計画フェーズ（Plan）
- **トリガー**: `status:needs-plan`ラベル
- **実行内容**:
  - Issue内容の解析
  - 実装計画の策定
  - 技術選定とアーキテクチャ設計
  - タスクの分解と優先度設定
- **アウトプット**: Issue本文への実行計画追記、`status:ready`ラベル更新

#### 実装フェーズ（Implementation）
- **トリガー**: `status:ready`ラベル
- **実行内容**:
  - 計画に基づいたコード実装
  - ユニットテストの作成
  - 統合テストの実行
  - コードスタイルの確認
- **アウトプット**: PR作成、`status:review-requested`ラベル更新

#### レビューフェーズ（Review）
- **トリガー**: `status:review-requested`ラベル
- **実行内容**:
  - コードレビューの実施
  - 品質チェック
  - 改善点の指摘とフィードバック
- **アウトプット**: レビュー完了（手動でのマージが必要）

### 内部動作の詳細

#### tmuxセッション管理
- **セッション作成**: `osoba-{repository-name}`形式
- **ウィンドウ管理**: Issue番号ごとに独立したウィンドウ
- **ウィンドウ命名**: `{issue-number}-{phase}`（例: `53-plan`, `53-implement`）
- **ペイン分割**: Claude実行用、ログ監視用、コード編集用

#### git worktree統合
- **worktree作成**: `.git/osoba/worktrees/{issue-number}`
- **ブランチ管理**: `osoba/#{issue-number}-{description}`形式
- **同期処理**: mainブランチとの自動同期
- **クリーンアップ**: フェーズ完了後の自動worktree削除

#### Claude AI実行
- **プロンプト管理**: フェーズごとに最適化されたプロンプト
- **コンテキスト管理**: Issue情報、コードベース、プロジェクト情報を統合
- **実行制御**: 非同期実行、進捗監視、エラーハンドリング
- **結果反映**: Issue更新、コードコミット、ラベル更新

## 詳細な設定

### 設定ファイルの構造

```yaml
# ~/.config/osoba/osoba.yml
github:
  # ghコマンドを使用する（デフォルト: true）
  use_gh_command: true
  # GitHub APIを直接使用する場合のみ必要
  # token: "${GITHUB_TOKEN}"
  poll_interval: 10s

tmux:
  session_prefix: "osoba-"

claude:
  phases:
    plan:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:plan {{issue-number}}"
    implement:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:implement {{issue-number}}"
    review:
      args: ["--dangerously-skip-permissions"]
      prompt: "/osoba:review {{issue-number}}"
```

### 環境変数

| 環境変数 | 説明 | デフォルト値 |
|----------|------|-------------|
| `GITHUB_TOKEN` | GitHub Personal Access Token（gh auth tokenで自動取得可） | - |

### GitHubアクセス方法

#### ghコマンドを使用（デフォルト）

デフォルトでは、osobaはGitHub CLI（gh）を使用してGitHubにアクセスします。これにより：
- GitHub Personal Access Tokenの管理が不要
- ghコマンドの認証情報を再利用
- より安全なトークン管理

前提条件：
```bash
# ghコマンドでログイン
$ gh auth login
```

#### GitHub APIを直接使用する場合

設定ファイルで`use_gh_command: false`を設定し、トークンを設定します。

トークン取得の優先順位：
1. **環境変数 `GITHUB_TOKEN`** - 最優先
2. **GitHub CLI (`gh auth token`)** - ghでログイン済みの場合
3. **設定ファイル** - osoba.yml内の設定

## セキュリティ上の注意事項

⚠️ **重要**: osobaは自律性を最大化するため、Claude実行時に`--dangerously-skip-permissions`オプションを使用します。セキュリティリスクがあることを理解した上で使用してください。

devcontainerや隔離された環境で実行するなど、可能な限りのセキュリティ対策を行ったうえで使用してください。


### 設計の背景

この設計選択は、開発プロセスの完全自律化を実現するために行われました。一般的な権限制限では、ファイル作成・編集、テスト実行、Git操作などの開発に必要な操作が制限されるため、`--dangerously-skip-permissions`オプションを採用しています。

### 代替案

より安全な使用を希望する場合は、`$HOME/.config/osoba/osoba.yml` に以下の設定変更を検討してください：

```yaml
claude:
  phases:
    plan:
      args: []  # remove --dangerously-skip-permissions
    implement:
      args: []
    review:
      args: []
```

## セットアップ

### 開発環境のセットアップ

1. Go 1.24.5以上をインストール
2. 開発ツールをインストール:
   ```bash
   make install-tools
   # または手動で:
   go install golang.org/x/tools/cmd/goimports@latest
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. Git hooksを有効化:
   ```bash
   git config core.hooksPath .githooks
   ```

### ビルド

```bash
go build
./osoba
```

### テスト

```bash
go test ./...
```

### Lint

```bash
make lint
# または
go vet ./...
```

## 開発

### コミット前のチェック

Git pre-commit hookが自動的に以下をチェックします:
- `go fmt` - コードフォーマット
- `go vet` - 静的解析
- `go mod tidy` - 依存関係の整理

### プロジェクト構造

```
osoba/
├── cmd/         # CLIコマンド
├── internal/    # 内部パッケージ
├── pkg/         # 公開パッケージ
├── .githooks/   # Git hooks
└── Makefile      # ビルドタスク
```

## 開発者向け情報

## ライセンス

このプロジェクトは[MITライセンス](LICENSE)の下で公開されています。

