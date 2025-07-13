# osoba Examples

このディレクトリには、osobaの使用例が含まれています。

## ディレクトリ構成

- `basic/` - 基本的な使用例
- `config/` - 設定ファイルのサンプル
- `advanced/` - 高度な使用例

## 基本的な使い方

### 1. シンプルな監視開始

```bash
# examples/basic/simple-watch.sh を参照
cd /path/to/your/repo
osoba watch
```

### 2. 設定ファイルを使った起動

```bash
# examples/config/sample-config.yml を参照
osoba watch --config examples/config/sample-config.yml
```

### 3. 複数リポジトリの監視

```bash
# examples/advanced/multi-repo.sh を参照
osoba watch --repos owner/repo1,owner/repo2
```

## 詳細な例

各ディレクトリ内のファイルを参照してください：

- 基本的な使用例は `basic/` ディレクトリ
- カスタマイズされた設定は `config/` ディレクトリ
- 複雑なワークフローは `advanced/` ディレクトリ