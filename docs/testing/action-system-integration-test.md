# ラベルベースアクション実行システム 動作確認手順

このドキュメントでは、Issue #13で実装したラベルベースアクション実行システムの動作確認方法を説明します。

## 1. 単体機能テスト

### テストプログラムの作成

以下の内容で`test_action_system.go`を作成：

```go
package main

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v50/github"
)

func main() {
	// 1. tmuxウィンドウ操作のテスト
	fmt.Println("=== tmuxウィンドウ操作のテスト ===")
	sessionName := "osoba-test"
	issueNumber := 99

	// ウィンドウ名の生成
	windowName := tmux.GetWindowName(issueNumber)
	fmt.Printf("Issue #%d -> Window name: %s\n", issueNumber, windowName)

	// 2. ActionExecutor のテスト
	fmt.Println("\n=== ActionExecutor のテスト ===")
	
	// テスト用のIssue作成
	testIssue := &github.Issue{
		Number: github.Int(13),
		Title:  github.String("Test Issue"),
		Labels: []*github.Label{
			{Name: github.String("status:needs-plan")},
		},
	}

	// ActionManagerの作成
	stateManager := watcher.NewIssueStateManager()
	actionManager := watcher.NewActionManager(sessionName)

	// アクションの実行
	ctx := context.Background()
	err := actionManager.ExecuteAction(ctx, testIssue)
	if err != nil {
		fmt.Printf("Error executing action: %v\n", err)
	} else {
		fmt.Println("✓ Action executed successfully (or no action needed)")
	}

	// 3. ラベルベースアクション判定のテスト
	fmt.Println("\n=== ラベルベースアクション判定のテスト ===")
	
	testCases := []struct {
		label    string
		expected string
	}{
		{"status:needs-plan", "plan"},
		{"status:ready", "implementation"},
		{"status:review-requested", "review"},
		{"unknown-label", "none"},
	}

	for _, tc := range testCases {
		issue := &github.Issue{
			Number: github.Int(1),
			Labels: []*github.Label{
				{Name: github.String(tc.label)},
			},
		}
		
		action := watcher.GetActionForIssue(issue)
		if action != nil {
			if typeGetter, ok := action.(interface{ ActionType() string }); ok {
				fmt.Printf("Label '%s' -> Action type: %s\n", tc.label, typeGetter.ActionType())
			}
		} else {
			fmt.Printf("Label '%s' -> No action\n", tc.label)
		}
	}

	// 4. 状態管理のテスト
	fmt.Println("\n=== 状態管理のテスト ===")
	
	// 状態の設定
	stateManager.SetState(100, watcher.IssueStatePlan, watcher.IssueStatusProcessing)
	fmt.Println("✓ Set Issue #100 to Plan/Processing")
	
	// 状態の確認
	if state, exists := stateManager.GetState(100); exists {
		fmt.Printf("✓ Issue #100 state: Phase=%s, Status=%s\n", state.Phase, state.Status)
	}
	
	// 処理完了
	stateManager.MarkAsCompleted(100, watcher.IssueStatePlan)
	if stateManager.HasBeenProcessed(100, watcher.IssueStatePlan) {
		fmt.Println("✓ Issue #100 Plan phase completed")
	}

	fmt.Println("\n動作確認完了！")
}
```

### 実行方法

```bash
# テストプログラムの実行
go run test_action_system.go

# 期待される出力：
# === tmuxウィンドウ操作のテスト ===
# Issue #99 -> Window name: issue-99
# 
# === ActionExecutor のテスト ===
# ✓ Action executed successfully (or no action needed)
# 
# === ラベルベースアクション判定のテスト ===
# Label 'status:needs-plan' -> Action type: plan
# Label 'status:ready' -> Action type: implementation
# Label 'status:review-requested' -> Action type: review
# Label 'unknown-label' -> No action
# 
# === 状態管理のテスト ===
# ✓ Set Issue #100 to Plan/Processing
# ✓ Issue #100 state: Phase=plan, Status=processing
# ✓ Issue #100 Plan phase completed
# 
# 動作確認完了！
```

## 2. ユニットテストの実行

```bash
# watcher パッケージのテスト
go test ./internal/watcher/... -v

# tmux パッケージのテスト
go test ./internal/tmux/... -v

# 全体のテスト
make test
```

## 3. 統合動作確認

### 3.1 事前準備

#### tmuxセッションの作成
```bash
# テスト用のtmuxセッションを作成
tmux new-session -d -s osoba-test-repo
```

#### 設定ファイルの準備
`~/.osoba/config.yaml`:
```yaml
github:
  token: "your-github-token"
  poll_interval: 30s

repos:
  - owner: "douhashi"
    name: "osoba"
    session_name: "osoba-test-repo"
```

### 3.2 watchコマンドの実行

```bash
# ビルド
go build -o osoba main.go

# watchコマンドを実行
./osoba watch
```

### 3.3 動作確認シナリオ

#### シナリオ1: 新しいIssueにラベルを付ける
1. GitHubで新しいIssueを作成
2. `status:needs-plan`ラベルを付ける
3. 30秒以内に以下が実行されることを確認：
   - tmuxウィンドウ `issue-XXX` が作成される
   - ログに「Executing action for issue #XXX with label status:needs-plan」が表示される

#### シナリオ2: 既存のIssueのラベルを変更
1. 既存のIssueのラベルを`status:ready`に変更
2. 新しいアクションが実行されることを確認
3. 同じIssueに対して重複実行されないことを確認

#### シナリオ3: tmuxウィンドウの確認
```bash
# tmuxセッション内のウィンドウ一覧を確認
tmux list-windows -t osoba-test-repo

# 期待される出力例：
# 0: main* (1 panes) [80x24]
# 1: issue-13 (1 panes) [80x24]
# 2: issue-14 (1 panes) [80x24]
```

### 3.4 ログの確認

osobaの実行ログで以下のメッセージを確認：
- `Starting to watch repository: douhashi/osoba`
- `Checking for issues with action labels...`
- `Found issue #XXX with label: status:needs-plan`
- `Executing action for issue #XXX`
- `Setting issue #XXX to Plan/Processing state`

## 4. トラブルシューティング

### tmuxセッションが見つからない場合
```bash
# セッションの存在確認
tmux list-sessions

# セッションが存在しない場合は作成
tmux new-session -d -s osoba-test-repo
```

### GitHubトークンエラーの場合
```bash
# 方法1: 環境変数で設定
export GITHUB_TOKEN="your-token"

# 方法2: GitHub CLIでログイン
gh auth login

# 方法3: 設定ファイルで設定
echo "github.token: your-token" >> ~/.osoba/config.yaml
```

### アクションが実行されない場合
1. ラベル名が正確か確認（`status:needs-plan`、`status:ready`、`status:review-requested`）
2. poll_intervalの設定を確認（デフォルト5分）
3. ログでエラーメッセージを確認

## 5. クリーンアップ

```bash
# tmuxセッションの削除
tmux kill-session -t osoba-test-repo

# テストファイルの削除
rm test_action_system.go

# ビルド成果物の削除
rm osoba
```

## 6. 実装の詳細

### 主要コンポーネント

1. **ActionExecutor インターフェース** (`internal/watcher/action.go`)
   - `Execute()`: アクションの実行
   - `CanExecute()`: 実行可能性の判定

2. **ActionManager** (`internal/watcher/action.go`)
   - ラベルに基づくアクションの選択と実行
   - 状態管理との連携

3. **IssueStateManager** (`internal/watcher/state.go`)
   - スレッドセーフな状態管理
   - 重複実行の防止

4. **tmux Window管理** (`internal/tmux/window.go`)
   - Issue番号に基づくウィンドウ作成
   - ウィンドウの存在確認と切り替え

### ラベルとアクションの対応

| ラベル | アクションタイプ | 説明 |
|--------|-----------------|------|
| `status:needs-plan` | PlanAction | 計画フェーズの実行 |
| `status:ready` | ImplementationAction | 実装フェーズの実行 |
| `status:review-requested` | ReviewAction | レビューフェーズの実行 |

### 状態遷移

```
Pending → Processing → Completed
                    ↘ Failed
```

各Issueは以下の状態を持ちます：
- **IssueNumber**: Issue番号
- **Phase**: 現在のフェーズ（plan/implementation/review）
- **Status**: 処理状態（pending/processing/completed/failed）
- **LastAction**: 最後のアクション実行時刻