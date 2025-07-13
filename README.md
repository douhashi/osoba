# osoba

[![CI](https://github.com/douhashi/osoba/actions/workflows/ci.yml/badge.svg)](https://github.com/douhashi/osoba/actions/workflows/ci.yml)
[![Release](https://github.com/douhashi/osoba/actions/workflows/release.yml/badge.svg)](https://github.com/douhashi/osoba/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/douhashi/osoba)](https://goreportcard.com/report/github.com/douhashi/osoba)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

自律的ソフトウェア開発補助ツール

## 概要

osobaは、tmux + git worktree + claude を組み合わせた自律的なソフトウェア開発を支援するCLIツールです。

## インストール

### リリース版のインストール

最新のリリース版は[GitHub Releases](https://github.com/douhashi/osoba/releases)からダウンロードできます。

### ソースからのビルド

```bash
# リポジトリのクローン
git clone https://github.com/douhashi/osoba.git
cd osoba

# ビルド
go build -o osoba main.go

# インストール（オプション）
go install
```

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
- **Lint**: `golangci-lint` による静的解析
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
make test        # テストを実行
make lint        # lintを実行
make fmt         # コードフォーマット
make clean       # ビルド成果物をクリーン
```