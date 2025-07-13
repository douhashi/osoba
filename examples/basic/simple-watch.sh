#!/bin/bash
# 基本的なosoba watchの使用例

set -e

echo "=== osoba Basic Example: Simple Watch ==="
echo

# 環境変数の確認
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is not set"
    echo "Please set it with: export GITHUB_TOKEN=your_token"
    exit 1
fi

# 現在のディレクトリがgitリポジトリか確認
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Current directory is not a git repository"
    exit 1
fi

# リポジトリ情報の取得
REPO_URL=$(git config --get remote.origin.url)
echo "Repository: $REPO_URL"
echo

# osobaの起動
echo "Starting osoba watch..."
echo "Press Ctrl+C to stop"
echo

# バックグラウンドでosoba watchを実行
osoba watch &

# プロセスIDを保存
OSOBA_PID=$!

# 終了時のクリーンアップ
trap "echo 'Stopping osoba...'; kill $OSOBA_PID 2>/dev/null || true" EXIT

# 簡単な使用方法の表示
echo "osoba is now watching for GitHub Issues with labels:"
echo "  - status:needs-plan"
echo "  - status:ready"
echo "  - status:review-requested"
echo
echo "To connect to the tmux session, run:"
echo "  osoba open"
echo

# プロセスが終了するまで待機
wait $OSOBA_PID