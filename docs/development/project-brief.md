# osoba プロジェクト概要

## プロジェクト名
osoba（オソバ）- 自律的ソフトウェア開発支援ツール

## ビジョン
開発者の創造的な作業に集中できる環境を提供し、反復的なタスクを自動化することで、ソフトウェア開発の生産性を劇的に向上させる。

## 概要
osobaは、tmux + git worktree + Claude を統合した自律的なソフトウェア開発支援CLIツールです。GitHub Issueをトリガーとして、AIが計画・実装・レビューの各フェーズを自律的に実行し、開発プロセスを大幅に効率化します。

## 主要機能

### 1. 自律的な開発フロー
- GitHub Issueのラベルに基づいた自動的なタスク実行
- 計画（Plan）→ 実装（Implementation）→ レビュー（Review）の3フェーズ管理
- 各フェーズでの適切なAIプロンプトの実行

### 2. tmuxセッション管理
- リポジトリごとの独立したtmuxセッション
- Issueごとのウィンドウ作成と管理
- 開発環境の自動セットアップ

### 3. git worktree統合
- Issueごとの独立したworktree作成
- ブランチの自動管理
- mainブランチとの同期

### 4. Claude AI統合
- フェーズごとに最適化されたプロンプト実行
- コンテキストを考慮した開発支援
- TDD（テスト駆動開発）のサポート

## アーキテクチャ

### コンポーネント構成
```
osoba/
├── cmd/           # CLIコマンド（watch, open）
├── internal/      # 内部パッケージ
│   ├── config/    # 設定管理
│   ├── github/    # GitHub API統合
│   ├── tmux/      # tmuxセッション管理
│   ├── git/       # git worktree操作
│   ├── claude/    # Claude実行管理
│   └── watcher/   # Issue監視ロジック
└── pkg/           # 公開パッケージ
    └── models/    # 共通データモデル
```

### 主要コマンド
- `osoba watch`: バックグラウンドでIssueを監視し、自律的に開発を実行
- `osoba open`: 実行中のtmuxセッションに接続

## ワークフロー

### 1. Issue作成とラベリング
開発者がGitHub Issueを作成し、適切なステータスラベルを付与：
- `status:needs-plan`: 計画が必要
- `status:ready`: 実装準備完了
- `status:review-requested`: レビュー依頼

### 2. 自動検知と実行
osobaがラベルを検知し、対応するアクションを実行：
```
[Issue検知] → [tmuxウィンドウ作成] → [git worktree作成] → [Claude実行]
```

### 3. フェーズ遷移
各フェーズ完了後、自動的に次のフェーズへ：
```
計画フェーズ → 実装フェーズ → レビューフェーズ
```

## 技術スタック
- **言語**: Go 1.21+
- **CLI Framework**: Cobra
- **GitHub API**: go-github
- **設定管理**: Viper
- **外部ツール**: tmux, git, claude

## 開発方針

### 1. シンプルさの追求
- 最小限の設定で動作
- 直感的なコマンドインターフェース
- 明確なエラーメッセージ

### 2. 拡張性
- プラグイン可能なアーキテクチャ
- カスタマイズ可能なプロンプト
- 複数のAIプロバイダーへの対応準備

### 3. 信頼性
- 堅牢なエラーハンドリング
- 状態の永続化
- 障害からの自動復旧

## ユースケース

### 1. 個人開発者
- サイドプロジェクトの効率的な進行
- アイデアから実装までの高速化
- コードレビューの自動化

### 2. チーム開発
- 定型的なタスクの自動化
- コーディング規約の自動適用
- ドキュメント生成の自動化

### 3. オープンソースプロジェクト
- Issue対応の効率化
- コントリビューターへの支援
- プルリクエストの品質向上

## 今後の展望

### Phase 1（現在）
- 基本的なwatch/open機能
- GitHub Issue連携
- Claude統合

### Phase 2
- 複数リポジトリの同時管理
- より高度なAIプロンプト
- 実行履歴とメトリクス

### Phase 3
- WebUIダッシュボード
- チーム機能
- プラグインシステム

## 制約事項
- tmuxが必要（Linux/macOS）
- GitHub APIトークンが必要
- Claude APIアクセスが必要
- git 2.x以上が必要

## セキュリティ考慮事項
- APIトークンの安全な管理
- ローカル設定ファイルの暗号化
- 機密情報の自動検出とマスキング

## 参考リンク
- [GitHub Repository](https://github.com/douhashi/osoba)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [tmux Documentation](https://github.com/tmux/tmux/wiki)