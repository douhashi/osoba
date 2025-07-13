#!/bin/bash
# 複数リポジトリを監視する高度な使用例

set -e

echo "=== osoba Advanced Example: Multi-Repository Watch ==="
echo

# 環境変数の確認
if [ -z "$GITHUB_TOKEN" ]; then
    echo "Error: GITHUB_TOKEN environment variable is not set"
    exit 1
fi

# 監視するリポジトリのリスト
REPOS=(
    "douhashi/osoba"
    "owner/repo1"
    "owner/repo2"
)

# 設定ファイルの作成
CONFIG_FILE="/tmp/osoba-multi-repo.yml"
cat > "$CONFIG_FILE" <<EOF
github:
  token: "\${GITHUB_TOKEN}"
  poll_interval: 3m
  repos:
EOF

# リポジトリをリストに追加
for repo in "${REPOS[@]}"; do
    echo "    - $repo" >> "$CONFIG_FILE"
done

# 残りの設定を追加
cat >> "$CONFIG_FILE" <<EOF

tmux:
  session_prefix: "osoba-multi-"
  window_layout: "tiled"

claude:
  model: "claude-3-opus-20240229"
  max_tokens: 4096

log:
  level: "info"
  file: "~/.osoba/multi-repo.log"
  console: true
EOF

echo "Configuration file created at: $CONFIG_FILE"
echo "Monitoring repositories:"
for repo in "${REPOS[@]}"; do
    echo "  - $repo"
done
echo

# osoba watchの実行
echo "Starting osoba with multi-repository configuration..."
osoba watch --config "$CONFIG_FILE" &

OSOBA_PID=$!

# クリーンアップ
trap "echo 'Cleaning up...'; kill $OSOBA_PID 2>/dev/null || true; rm -f $CONFIG_FILE" EXIT

echo
echo "osoba is now monitoring multiple repositories"
echo "To see all tmux sessions: tmux ls | grep osoba-multi"
echo "To attach to a session: osoba open"
echo
echo "Press Ctrl+C to stop"

wait $OSOBA_PID