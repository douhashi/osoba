package log

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LogEntry はログエントリーを表す
type LogEntry struct {
	Time      time.Time              `json:"time"`
	Level     Level                  `json:"level"`
	Component string                 `json:"component,omitempty"`
	Message   string                 `json:"msg"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Formatter はログフォーマッターのインターフェース
type Formatter interface {
	Format(entry *LogEntry, colorizer *colorizer) string
}

// TextFormatter はテキスト形式のフォーマッター
type TextFormatter struct{}

// Format はテキスト形式でログをフォーマットする
func (f *TextFormatter) Format(entry *LogEntry, colorizer *colorizer) string {
	var parts []string

	// タイムスタンプ
	timestamp := entry.Time.Format("2006-01-02T15:04:05.000Z07:00")
	parts = append(parts, timestamp)

	// レベル
	level := colorizer.colorizeLevel(entry.Level)
	parts = append(parts, fmt.Sprintf("[%s]", level))

	// コンポーネント
	if entry.Component != "" {
		component := colorizer.colorizeComponent(entry.Component)
		parts = append(parts, fmt.Sprintf("[%s]", component))
	}

	// メッセージ
	parts = append(parts, entry.Message)

	// フィールド
	if entry.Fields != nil && len(entry.Fields) > 0 {
		var fieldParts []string
		for k, v := range entry.Fields {
			fieldParts = append(fieldParts, fmt.Sprintf("%s=%v", k, v))
		}
		if len(fieldParts) > 0 {
			parts = append(parts, strings.Join(fieldParts, " "))
		}
	}

	return strings.Join(parts, " ")
}

// JSONFormatter はJSON形式のフォーマッター
type JSONFormatter struct{}

// Format はJSON形式でログをフォーマットする
func (f *JSONFormatter) Format(entry *LogEntry, colorizer *colorizer) string {
	// JSON形式では色分けは行わない
	data := map[string]interface{}{
		"time":  entry.Time.Format(time.RFC3339Nano),
		"level": entry.Level.String(),
		"msg":   entry.Message,
	}

	if entry.Component != "" {
		data["component"] = entry.Component
	}

	// フィールドを展開
	if entry.Fields != nil {
		for k, v := range entry.Fields {
			data[k] = v
		}
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		// JSON化に失敗した場合はプレーンテキストで出力
		return fmt.Sprintf("JSON_ERROR: %v - Original: %s", err, entry.Message)
	}

	return string(bytes)
}

// newFormatter は指定された形式のフォーマッターを作成する
func newFormatter(format Format) Formatter {
	switch format {
	case JSONFormat:
		return &JSONFormatter{}
	case TextFormat:
		return &TextFormatter{}
	default:
		return &TextFormatter{}
	}
}
