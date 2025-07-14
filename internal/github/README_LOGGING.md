# GitHub API ログ機能

## 概要

GitHub APIクライアントにHTTPリクエスト/レスポンスのロギング機能を実装しました。
この機能により、API呼び出しの詳細な情報をログに記録し、デバッグやトラブルシューティングが容易になります。

## 実装内容

### 1. loggingRoundTripper

HTTPラウンドトリッパーの実装により、以下の情報をログに記録します：

**リクエスト情報:**
- HTTPメソッド
- URL（クエリパラメータを含む）
- Authorizationヘッダー（マスキング済み）
- User-Agentヘッダー

**レスポンス情報:**
- ステータスコード
- レスポンス時間（ミリ秒）
- レート制限情報（X-RateLimit-Remaining, X-RateLimit-Reset）
- レスポンスボディのプレビュー（最大200文字）

### 2. セキュリティ機能

- Authorizationヘッダーの値は自動的にマスキングされます
  - `Bearer ghp_xxx` → `Bearer [REDACTED]`
  - `token xxx` → `token [REDACTED]`
  - `Basic xxx` → `Basic [REDACTED]`

### 3. エラーハンドリング

- HTTPリクエストエラーも詳細にログ記録
- エラー時の経過時間も記録

## 使用方法

```go
// ロガーを作成
zapLogger, _ := zap.NewDevelopment()
logger := logger.NewZapLogger(zapLogger)

// ログ機能付きクライアントを作成
client, err := github.NewClientWithLogger(token, logger)
if err != nil {
    log.Fatal(err)
}

// 通常通りAPIを呼び出す
repo, err := client.GetRepository(ctx, "owner", "repo")
```

## テスト

### ユニットテスト

```bash
go test ./internal/github/
```

### 統合テスト

実際のGitHub APIを使用したテストを実行：

```bash
export GITHUB_TOKEN="your-github-token"
go test -tags=integration ./internal/github/
```

## 既知の制限事項

### OAuth2トランスポートとの統合

現在の実装では、oauth2パッケージがRoundTripメソッド内でAuthorizationヘッダーを追加するため、
loggingRoundTripperではAuthorizationヘッダーをキャプチャできません。

これは、oauth2.Transportが以下のような動作をするためです：
1. loggingRoundTripperがリクエストを受け取る
2. oauth2.TransportのRoundTripが呼ばれる
3. oauth2.Transport内部でAuthorizationヘッダーが追加される
4. 実際のHTTPリクエストが送信される

この制限により、リクエストログにはAuthorizationヘッダーが含まれませんが、
実際のAPIリクエストでは正しく認証が行われます。

## 今後の改善案

1. **カスタムOAuth2トランスポート**: oauth2の動作を拡張してヘッダーをキャプチャ
2. **レスポンスボディの完全キャプチャ**: 現在は200文字のプレビューのみ
3. **構造化ログの拡張**: リクエスト/レスポンスの詳細な構造化
4. **メトリクス収集**: API呼び出しの統計情報収集