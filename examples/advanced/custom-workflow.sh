#!/bin/bash
# カスタムワークフローの例：特定のラベルで異なるアクションを実行

set -e

echo "=== osoba Advanced Example: Custom Workflow ==="
echo

# カスタム設定の作成
create_custom_config() {
    local config_file="/tmp/osoba-custom-workflow.yml"
    
    cat > "$config_file" <<'EOF'
github:
  token: "${GITHUB_TOKEN}"
  poll_interval: 2m
  
  # カスタムラベル設定
  labels:
    # 機能開発用
    feature_plan: "type:feature,status:needs-plan"
    feature_ready: "type:feature,status:ready"
    
    # バグ修正用
    bug_plan: "type:bug,status:needs-plan"
    bug_ready: "type:bug,status:ready"
    
    # ドキュメント用
    doc_plan: "type:documentation,status:needs-plan"
    doc_ready: "type:documentation,status:ready"

tmux:
  session_prefix: "osoba-custom-"
  
  # Issueタイプごとに異なるレイアウト
  window_layout: "main-vertical"

claude:
  model: "claude-3-opus-20240229"
  
  # タイプごとに異なる設定
  feature:
    max_tokens: 8192
    temperature: 0.8
  
  bug:
    max_tokens: 4096
    temperature: 0.5
  
  documentation:
    max_tokens: 4096
    temperature: 0.7

log:
  level: "debug"
  file: "~/.osoba/custom-workflow.log"

# カスタムフック設定
hooks:
  # Issue処理前に実行
  pre_process:
    - "echo 'Processing issue: {{.Issue.Number}}'"
    - "git fetch origin"
  
  # Issue処理後に実行
  post_process:
    - "echo 'Completed issue: {{.Issue.Number}}'"
    - "git push origin HEAD"
EOF
    
    echo "$config_file"
}

# メイン処理
main() {
    # 環境変数チェック
    if [ -z "$GITHUB_TOKEN" ]; then
        echo "Error: GITHUB_TOKEN is required"
        exit 1
    fi
    
    # カスタム設定ファイルの作成
    CONFIG_FILE=$(create_custom_config)
    echo "Custom configuration created at: $CONFIG_FILE"
    echo
    
    # カスタムプロンプトディレクトリの作成（オプション）
    PROMPT_DIR="$HOME/.osoba/prompts"
    mkdir -p "$PROMPT_DIR"
    
    # 機能開発用プロンプトの作成
    cat > "$PROMPT_DIR/feature-plan.md" <<'EOF'
# Feature Planning Prompt

You are planning a new feature. Please consider:
1. User experience and interface design
2. Technical architecture and implementation approach
3. Testing strategy
4. Performance implications
5. Security considerations
EOF
    
    # バグ修正用プロンプトの作成
    cat > "$PROMPT_DIR/bug-plan.md" <<'EOF'
# Bug Fix Planning Prompt

You are planning a bug fix. Please consider:
1. Root cause analysis
2. Impact assessment
3. Fix approach with minimal side effects
4. Regression test cases
5. Verification steps
EOF
    
    echo "Custom prompts created in: $PROMPT_DIR"
    echo
    
    # osoba起動
    echo "Starting osoba with custom workflow..."
    osoba watch --config "$CONFIG_FILE" &
    
    OSOBA_PID=$!
    
    # クリーンアップ
    trap "echo 'Stopping...'; kill $OSOBA_PID 2>/dev/null || true; rm -f $CONFIG_FILE" EXIT
    
    echo
    echo "Custom workflow is active with:"
    echo "  - Feature issues: type:feature label"
    echo "  - Bug issues: type:bug label"
    echo "  - Documentation issues: type:documentation label"
    echo
    echo "Each type will be processed with different Claude parameters"
    echo
    echo "Press Ctrl+C to stop"
    
    wait $OSOBA_PID
}

# 実行
main "$@"