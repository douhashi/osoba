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
- 🔄 **継続的な監視**: バックグラウンドでIssueを監視し、自動的にアクションを実行

## 必要な環境

- Go 1.21以上
- tmux 3.0以上
- git 2.x以上
- GitHub CLI（gh）
- Claude CLI（claude）

## インストール

### 推奨: Homebrewを使用（macOS/Linux）

```bash
# 近日公開予定
brew install douhashi/tap/osoba
```

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
# 設定ファイルの作成
mkdir -p ~/.config/osoba
cat > ~/.config/osoba/config.yml << EOF
github:
  token: "${GITHUB_TOKEN}"
  poll_interval: 5m
tmux:
  session_prefix: "osoba-"
claude:
  model: "claude-3-opus-20240229"
EOF
```

### 2. 基本的な使い方

```bash
# リポジトリでosobaを開始
cd /path/to/your/repo
osoba watch

# 別のターミナルでセッションに接続
osoba open
```

### 3. ワークフロー例

1. GitHub Issueを作成し、`status:needs-plan`ラベルを付与
2. osobaが自動的にIssueを検知し、計画フェーズを実行
3. 計画完了後、`status:ready`ラベルに更新
4. 実装フェーズが自動的に開始
5. `osoba open`でセッションに接続して進捗を確認

## 詳細な設定

### 設定ファイルの構造

```yaml
# ~/.config/osoba/config.yml
github:
  token: "ghp_xxxxxxxxxxxx"  # GitHub Personal Access Token
  poll_interval: 5m           # Issue監視間隔
  repos:                      # 監視するリポジトリ（省略時は現在のリポジトリ）
    - owner/repo1
    - owner/repo2

tmux:
  session_prefix: "osoba-"    # tmuxセッション名のプレフィックス
  window_layout: "tiled"      # ウィンドウレイアウト

claude:
  model: "claude-3-opus-20240229"
  max_tokens: 4096
  temperature: 0.7

log:
  level: "info"               # ログレベル: debug, info, warn, error
  file: "~/.osoba/osoba.log" # ログファイルのパス
```

### 環境変数

| 環境変数 | 説明 | デフォルト値 |
|----------|------|-------------|
| `OSOBA_CONFIG` | 設定ファイルのパス | `~/.config/osoba/config.yml` |
| `OSOBA_LOG_LEVEL` | ログレベル | `info` |
| `GITHUB_TOKEN` | GitHub Personal Access Token | - |

## セットアップ

### 開発環境のセットアップ

1. Go 1.24.5以上をインストール
2. 開発ツールをインストール:
   ```bash
   make install-tools
   # または手動で:
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest
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
golangci-lint run
```

## 開発

### コミット前のチェック

Git pre-commit hookが自動的に以下をチェックします:
- `go fmt` - コードフォーマット
- `go vet` - 静的解析
- `golangci-lint` - 統合リンター

### プロジェクト構造

```
osoba/
├── cmd/         # CLIコマンド
├── internal/    # 内部パッケージ
├── pkg/         # 公開パッケージ
├── .githooks/   # Git hooks
└── .golangci.yml # golangci-lint設定
```

## CI/CD

このプロジェクトでは、GitHub Actionsを使用してCI/CDパイプラインを構築しています。

### CI ワークフロー

プルリクエストとmainブランチへのプッシュで以下が実行されます：

- **テスト**: `go test -v -race ./...`
- **ビルド**: 各プラットフォーム向けのクロスコンパイル
- **Lint**: `go vet` と `go fmt` による静的解析
- **コードカバレッジ**: Codecovへのレポート送信

### リリースワークフロー

タグプッシュ時に自動的にリリースが作成されます：

```bash
# バージョンタグを作成してプッシュ
git tag v0.1.0
git push origin v0.1.0
```

GoReleaserが以下のプラットフォーム向けバイナリを生成：
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## 開発者向け情報

### Makefileタスク

```bash
make help        # 利用可能なタスクを表示
make build       # バイナリをビルド
make install     # バイナリをインストール
make test        # テストを実行
make lint        # lintを実行
make fmt         # コードフォーマット
make clean       # ビルド成果物をクリーン
make run         # アプリケーションを実行
```

## 貢献方法

プロジェクトへの貢献を歓迎します！詳細は[CONTRIBUTING.md](CONTRIBUTING.md)をご覧ください。

## ライセンス

このプロジェクトは[MITライセンス](LICENSE)の下で公開されています。

## 関連ドキュメント

- [プロジェクト概要](docs/development/project-brief.md)
- [Goコーディング規約](docs/development/go-coding-standards.md)
- [Git運用ルール](docs/development/git-instructions.md)
- [ghコマンドガイド](docs/development/gh-instructions.md)