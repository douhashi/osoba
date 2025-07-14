#!/bin/bash

echo "🔍 GitHub Issue監視トラブルシューティング"
echo "========================================="

# 環境変数チェック
echo "1. 環境変数チェック:"
if [ -z "$OSOBA_GITHUB_TOKEN" ] && [ -z "$GITHUB_TOKEN" ]; then
    echo "   ❌ GitHubトークンが設定されていません"
    echo "   解決方法: export OSOBA_GITHUB_TOKEN='your_token'"
else
    echo "   ✅ GitHubトークンが設定されています"
fi

if [ -z "$OSOBA_GITHUB_OWNER" ]; then
    echo "   ⚠️  OSOBA_GITHUB_OWNER が設定されていません (デフォルト: douhashi)"
else
    echo "   ✅ GitHub Owner: $OSOBA_GITHUB_OWNER"
fi

if [ -z "$OSOBA_GITHUB_REPO" ]; then
    echo "   ⚠️  OSOBA_GITHUB_REPO が設定されていません (デフォルト: osoba)"
else
    echo "   ✅ GitHub Repo: $OSOBA_GITHUB_REPO"
fi

echo ""
echo "2. GitHub API接続テスト:"

# curlでのAPI接続テスト
TOKEN=${OSOBA_GITHUB_TOKEN:-$GITHUB_TOKEN}
OWNER=${OSOBA_GITHUB_OWNER:-douhashi}
REPO=${OSOBA_GITHUB_REPO:-osoba}

if [ -n "$TOKEN" ]; then
    echo "   リポジトリ接続テスト中..."
    RESPONSE=$(curl -s -H "Authorization: token $TOKEN" \
        "https://api.github.com/repos/$OWNER/$REPO")
    
    if echo "$RESPONSE" | grep -q '"full_name"'; then
        echo "   ✅ リポジトリ接続成功"
        
        echo "   Issue取得テスト中..."
        ISSUES=$(curl -s -H "Authorization: token $TOKEN" \
            "https://api.github.com/repos/$OWNER/$REPO/issues?state=open&labels=status:needs-plan")
        
        ISSUE_COUNT=$(echo "$ISSUES" | jq '. | length' 2>/dev/null || echo "エラー")
        
        if [ "$ISSUE_COUNT" = "エラー" ]; then
            echo "   ⚠️  JSON解析エラー (jqコマンドが必要)"
        elif [ "$ISSUE_COUNT" -gt 0 ]; then
            echo "   ✅ status:needs-plan ラベルのIssue: ${ISSUE_COUNT}件"
        else
            echo "   ⚠️  status:needs-plan ラベルのIssueが見つかりません"
        fi
    else
        echo "   ❌ リポジトリ接続失敗"
        echo "   レスポンス: $RESPONSE"
    fi
else
    echo "   ❌ トークンが設定されていないため、API接続テストをスキップ"
fi

echo ""
echo "3. 推奨する解決手順:"
echo "   1. GitHubトークンを設定: export OSOBA_GITHUB_TOKEN='your_token'"
echo "   2. デバッグテスト実行: go run debug-test.go"
echo "   3. 詳細ログで監視: ./osoba start --watch --verbose"