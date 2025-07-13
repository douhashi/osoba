# osoba

自律的ソフトウェア開発補助ツール

## 概要

osobaは、tmux + git worktree + claude を組み合わせた自律的なソフトウェア開発を支援するCLIツールです。

## セットアップ

### 開発環境のセットアップ

1. Go 1.21以上をインストール
2. golangci-lintをインストール:
   ```bash
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.61.0
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