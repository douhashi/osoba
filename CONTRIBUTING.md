# Contributing to osoba

osobaプロジェクトへの貢献を検討いただき、ありがとうございます！このドキュメントでは、プロジェクトへの貢献方法について説明します。

## はじめに

osobaは、開発者の生産性向上を目指すオープンソースプロジェクトです。皆様からの貢献により、より良いツールへと成長していきます。

## 貢献の方法

### 1. Issue の報告

バグの発見や機能の提案がある場合は、[GitHub Issues](https://github.com/douhashi/osoba/issues)で報告してください。

#### バグ報告のテンプレート

```markdown
## 概要
[バグの簡潔な説明]

## 再現手順
1. [ステップ1]
2. [ステップ2]
3. [ステップ3]

## 期待される動作
[正常な場合の動作]

## 実際の動作
[バグによる動作]

## 環境
- OS: [例: macOS 14.0]
- Go version: [例: 1.21.5]
- osoba version: [例: v0.1.0]
```

### 2. プルリクエストの作成

#### 開発環境のセットアップ

```bash
# リポジトリのフォーク
# GitHubでフォークボタンをクリック

# クローン
git clone https://github.com/YOUR_USERNAME/osoba.git
cd osoba

# 開発ツールのインストール
make install-tools

# Git hooksの有効化
git config core.hooksPath .githooks

# ブランチの作成
git checkout -b feat/#<issue番号>-<機能名>
```

#### 開発フロー

1. **Issueの確認**: 作業を始める前に、関連するIssueを確認または作成
2. **ブランチの作成**: Git運用ルールに従ってブランチを作成
3. **テスト駆動開発**: テストを先に書いてから実装
4. **コミット**: 意味のある単位でコミット
5. **プルリクエスト**: mainブランチへのPRを作成

#### プルリクエストのチェックリスト

- [ ] 関連するIssueがリンクされている
- [ ] テストが追加/更新されている
- [ ] すべてのテストがパスしている (`go test ./...`)
- [ ] コードが正しくフォーマットされている (`go fmt ./...`)
- [ ] 静的解析がパスしている (`go vet ./...`)
- [ ] ドキュメントが更新されている（必要な場合）

## コーディング規約

### Go言語のスタイル

[Goコーディング規約](docs/development/go-coding-standards.md)に従ってください。主なポイント：

- **命名規則**: GoのイディオムとConventionに従う
- **エラーハンドリング**: 適切にエラーをラップして返す
- **テスト**: テーブル駆動テストを推奨
- **ドキュメント**: 公開APIには必ずコメントを付ける

### コミットメッセージ

[Git運用ルール](docs/development/git-instructions.md)に従ってください：

```
<type>: <subject>

[optional body]

[optional footer]
```

例：
```
feat: GitHub Issue監視機能を追加

- ポーリング間隔を設定可能に
- 複数リポジトリの同時監視をサポート

Closes #123
```

## テスト

### ユニットテストの実行

```bash
# すべてのテストを実行
go test ./...

# カバレッジ付きで実行
go test -cover ./...

# 特定のパッケージのテスト
go test ./internal/watcher
```

### テストの書き方

```go
func TestWatcher_Start(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {
            name: "正常系: 有効な設定で起動",
            config: Config{
                PollInterval: 5 * time.Minute,
            },
            wantErr: false,
        },
        {
            name: "異常系: 無効な設定",
            config: Config{
                PollInterval: 0,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := NewWatcher(tt.config)
            err := w.Start(context.Background())
            if (err != nil) != tt.wantErr {
                t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## ドキュメント

### ドキュメントの更新

新しい機能を追加した場合は、以下のドキュメントの更新を検討してください：

- `README.md`: 主要な機能変更の場合
- `docs/`: 技術的な詳細ドキュメント
- コード内コメント: 公開APIや複雑なロジック

### ドキュメントのスタイル

- 明確で簡潔な日本語を使用
- コード例を積極的に含める
- 見出しは階層的に構成

## リリースプロセス

1. **バージョンタグ**: セマンティックバージョニングに従う
2. **CHANGELOG**: 重要な変更を記録
3. **リリースノート**: ユーザー向けの変更内容を記載

## コミュニティ

### 行動規範

- 建設的で礼儀正しいコミュニケーション
- 多様性を尊重し、インクルーシブな環境を維持
- 技術的な議論に集中

### サポート

質問や議論は以下で行えます：

- GitHub Issues: バグ報告や機能提案
- GitHub Discussions: 一般的な質問や議論

## ライセンス

貢献していただいたコードは、プロジェクトと同じ[MITライセンス](LICENSE)の下で公開されます。

## 謝辞

osobaプロジェクトに貢献してくださるすべての方々に感謝します！皆様の協力により、より良い開発体験を提供できます。

---

質問がある場合は、お気軽にIssueを作成してください。Happy coding! 🚀