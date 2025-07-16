package tmux

import "os"

// init はパッケージ初期化時に実行される
func init() {
	// テスト環境ではモックマネージャーを使用
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		// テスト時は何もしない（テストコードで個別に設定）
		return
	}

	// 本番環境ではデフォルトマネージャーを使用
	// globalManager は global_manager.go で定義済み
}
