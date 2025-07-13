# Go モジュール管理ガイド

Go モジュールは、Go 1.11で導入された依存関係管理システムです。このガイドでは、osobaプロジェクトでのGoモジュールの使用方法について説明します。

## 基本概念

### go.mod ファイル

プロジェクトのルートに配置され、以下を定義します：
- モジュール名
- Go バージョン
- 依存関係

```go
module github.com/douhashi/osoba

go 1.21

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
)
```

### go.sum ファイル

依存関係の暗号学的ハッシュを記録し、再現可能なビルドを保証します。

## 基本的な使い方

### プロジェクトの初期化

```bash
# 新しいモジュールを初期化
go mod init github.com/douhashi/osoba
```

### 依存関係の管理

```bash
# 依存関係を追加（コード内でimportして go build/run で自動追加）
go get github.com/spf13/cobra@latest

# 特定バージョンを指定
go get github.com/spf13/cobra@v1.8.0

# 最新のマイナーバージョンを取得
go get -u github.com/spf13/cobra

# 最新のパッチバージョンを取得
go get -u=patch github.com/spf13/cobra

# 依存関係を削除
go get github.com/spf13/cobra@none
```

### 依存関係の整理

```bash
# 未使用の依存関係を削除
go mod tidy

# go.modとgo.sumの検証
go mod verify

# ベンダーディレクトリの作成
go mod vendor

# 依存関係のダウンロード
go mod download
```

## バージョン管理

### セマンティックバージョニング

Goモジュールはセマンティックバージョニング（semver）に従います：

- `v1.2.3`: 正確なバージョン
- `v1.2.x`: パッチバージョンの最新
- `v1.x.x`: マイナーバージョンの最新
- `latest`: 最新の安定版

### メジャーバージョンの扱い

v2以降のメジャーバージョンは、インポートパスに含める必要があります：

```go
import "github.com/example/module/v2"
```

go.modでの記述：
```go
require github.com/example/module/v2 v2.1.0
```

## replace ディレクティブ

ローカル開発や一時的な修正に使用：

```go
// ローカルパスへの置き換え
replace github.com/example/module => ../local-module

// 特定バージョンへの固定
replace github.com/example/broken v1.2.3 => github.com/example/broken v1.2.2

// フォークの使用
replace github.com/original/repo => github.com/yourfork/repo v1.0.0
```

## プライベートリポジトリ

### 環境変数の設定

```bash
# プライベートリポジトリのアクセス設定
export GOPRIVATE=github.com/yourorg/*
export GONOSUMDB=github.com/yourorg/*

# Git認証の設定
git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
```

### .netrcファイルの使用

```bash
# ~/.netrc
machine github.com
  login your-username
  password your-token
```

## ワークスペースモード

Go 1.18以降で利用可能な、複数モジュールの同時開発機能：

```bash
# ワークスペースの初期化
go work init ./main-module ./sub-module

# モジュールの追加
go work use ./another-module
```

go.work ファイル：
```go
go 1.21

use (
    ./main-module
    ./sub-module
    ./another-module
)
```

## CI/CD での使用

### GitHub Actions の例

```yaml
name: Go Build and Test

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
        cache: true
    
    - name: Download dependencies
      run: go mod download
    
    - name: Verify dependencies
      run: go mod verify
    
    - name: Build
      run: go build -v ./...
    
    - name: Test
      run: go test -v -race ./...
```

### Dockerでの使用

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 依存関係のキャッシュ
COPY go.mod go.sum ./
RUN go mod download

# ソースコードのコピーとビルド
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o osoba main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/osoba .

CMD ["./osoba"]
```

## ベストプラクティス

### 1. 最小バージョン選択

Goは最小バージョン選択（MVS）アルゴリズムを使用します。常に要求される最小バージョンを選択することで、予期しない更新を防ぎます。

### 2. go.sum のコミット

`go.sum`ファイルは必ずバージョン管理にコミットし、ビルドの再現性を保証します。

### 3. 定期的な更新

```bash
# 直接依存関係の更新確認
go list -u -m all

# 脆弱性のチェック
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

### 4. モジュールプロキシの活用

```bash
# デフォルトのプロキシ設定
export GOPROXY=https://proxy.golang.org,direct

# 企業環境での設定例
export GOPROXY=https://corp-proxy.example.com,https://proxy.golang.org,direct
```

### 5. 最小限の公開API

internal/ディレクトリを活用して、公開APIを最小限に保ちます：

```
osoba/
├── internal/    # 外部からインポート不可
│   ├── config/
│   └── watcher/
└── pkg/         # 外部からインポート可能
    └── models/
```

## トラブルシューティング

### 依存関係の問題

```bash
# キャッシュのクリア
go clean -modcache

# 強制的な再ダウンロード
go mod download -x

# 依存関係グラフの確認
go mod graph
```

### バージョン競合の解決

```bash
# 特定パッケージの依存関係を確認
go mod why github.com/example/package

# 利用可能なバージョンの確認
go list -m -versions github.com/example/package
```

### プロキシ/認証の問題

```bash
# プロキシを無効化してデバッグ
GOPROXY=direct go mod download

# 詳細なログ出力
go mod download -x
```

## 参考リンク

- [Go Modules Reference](https://go.dev/ref/mod)
- [Tutorial: Getting started with multi-module workspaces](https://go.dev/doc/tutorial/workspaces)
- [Module version numbering](https://go.dev/doc/modules/version-numbers)