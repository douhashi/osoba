package log

import (
	"fmt"
	"os"
	"strings"
)

// ANSIカラーコード
const (
	// リセット
	Reset = "\x1b[0m"

	// 色
	Gray   = "\x1b[37m" // グレー (DEBUG)
	Blue   = "\x1b[34m" // 青 (INFO)
	Yellow = "\x1b[33m" // 黄 (WARN)
	Red    = "\x1b[31m" // 赤 (ERROR)

	// 太字
	Bold = "\x1b[1m"
)

// colorizer は色分け表示を管理する
type colorizer struct {
	enabled bool
}

// newColorizer は新しいカラライザーを作成する
func newColorizer(enabled bool) *colorizer {
	// 環境変数でカラー表示を無効化できる
	if os.Getenv("NO_COLOR") != "" {
		enabled = false
	}

	// 出力先がターミナルでない場合は色分けを無効化
	if !isTerminal() {
		enabled = false
	}

	return &colorizer{enabled: enabled}
}

// colorize は指定された文字列を色分けする
func (c *colorizer) colorize(text string, color string) string {
	if !c.enabled {
		return text
	}
	return color + text + Reset
}

// colorizeLevel はログレベルを色分けする
func (c *colorizer) colorizeLevel(level Level) string {
	levelStr := level.String()

	if !c.enabled {
		return levelStr
	}

	var color string
	switch level {
	case DebugLevel:
		color = Gray
	case InfoLevel:
		color = Blue
	case WarnLevel:
		color = Yellow
	case ErrorLevel:
		color = Red
	default:
		color = ""
	}

	if color != "" {
		return fmt.Sprintf("%s%s%s", color, levelStr, Reset)
	}

	return levelStr
}

// colorizeComponent はコンポーネント名を色分けする
func (c *colorizer) colorizeComponent(component string) string {
	if !c.enabled {
		return component
	}

	// コンポーネント名は太字で表示
	return fmt.Sprintf("%s%s%s", Bold, component, Reset)
}

// isTerminal は出力先がターミナルかどうかを判定する
func isTerminal() bool {
	// 簡単なターミナル判定
	// より正確な判定が必要な場合は、github.com/mattn/go-isatty などを使用
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}

// stripColors は文字列からANSIカラーコードを除去する
func stripColors(text string) string {
	// ANSIエスケープシーケンスを除去
	// 簡単な実装なので、より複雑なエスケープシーケンスには対応していない
	result := text

	// よく使われるANSIカラーコードを除去
	colorCodes := []string{
		Reset, Gray, Blue, Yellow, Red, Bold,
		"\x1b[0m", "\x1b[1m", "\x1b[31m", "\x1b[32m", "\x1b[33m", "\x1b[34m", "\x1b[35m", "\x1b[36m", "\x1b[37m",
	}

	for _, code := range colorCodes {
		result = strings.ReplaceAll(result, code, "")
	}

	return result
}
