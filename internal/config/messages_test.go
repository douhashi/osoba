package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhaseMessageConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   PhaseMessageConfig
		expected map[string]string
	}{
		{
			name: "デフォルトメッセージ",
			config: PhaseMessageConfig{
				Plan:      "osoba: 計画を作成します",
				Implement: "osoba: 実装を開始します",
				Review:    "osoba: レビューを開始します",
			},
			expected: map[string]string{
				"plan":      "osoba: 計画を作成します",
				"implement": "osoba: 実装を開始します",
				"review":    "osoba: レビューを開始します",
			},
		},
		{
			name: "カスタムメッセージ",
			config: PhaseMessageConfig{
				Plan:      "🤖 計画フェーズを開始します...",
				Implement: "🤖 実装フェーズを開始します...",
				Review:    "🤖 レビューフェーズを開始します...",
			},
			expected: map[string]string{
				"plan":      "🤖 計画フェーズを開始します...",
				"implement": "🤖 実装フェーズを開始します...",
				"review":    "🤖 レビューフェーズを開始します...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 各フェーズのメッセージを確認
			assert.Equal(t, tt.expected["plan"], tt.config.Plan)
			assert.Equal(t, tt.expected["implement"], tt.config.Implement)
			assert.Equal(t, tt.expected["review"], tt.config.Review)
		})
	}
}

func TestConfig_GetPhaseMessage(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		phase     string
		wantMsg   string
		wantFound bool
	}{
		{
			name: "計画フェーズのメッセージ取得",
			config: &Config{
				GitHub: GitHubConfig{
					Messages: PhaseMessageConfig{
						Plan:      "osoba: 計画を作成します",
						Implement: "osoba: 実装を開始します",
						Review:    "osoba: レビューを開始します",
					},
				},
			},
			phase:     "plan",
			wantMsg:   "osoba: 計画を作成します",
			wantFound: true,
		},
		{
			name: "実装フェーズのメッセージ取得",
			config: &Config{
				GitHub: GitHubConfig{
					Messages: PhaseMessageConfig{
						Plan:      "osoba: 計画を作成します",
						Implement: "osoba: 実装を開始します",
						Review:    "osoba: レビューを開始します",
					},
				},
			},
			phase:     "implement",
			wantMsg:   "osoba: 実装を開始します",
			wantFound: true,
		},
		{
			name: "レビューフェーズのメッセージ取得",
			config: &Config{
				GitHub: GitHubConfig{
					Messages: PhaseMessageConfig{
						Plan:      "osoba: 計画を作成します",
						Implement: "osoba: 実装を開始します",
						Review:    "osoba: レビューを開始します",
					},
				},
			},
			phase:     "review",
			wantMsg:   "osoba: レビューを開始します",
			wantFound: true,
		},
		{
			name: "存在しないフェーズ",
			config: &Config{
				GitHub: GitHubConfig{
					Messages: PhaseMessageConfig{
						Plan:      "osoba: 計画を作成します",
						Implement: "osoba: 実装を開始します",
						Review:    "osoba: レビューを開始します",
					},
				},
			},
			phase:     "unknown",
			wantMsg:   "",
			wantFound: false,
		},
		{
			name: "空のメッセージ設定",
			config: &Config{
				GitHub: GitHubConfig{
					Messages: PhaseMessageConfig{},
				},
			},
			phase:     "plan",
			wantMsg:   "",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, found := tt.config.GetPhaseMessage(tt.phase)
			assert.Equal(t, tt.wantMsg, msg)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}

func TestNewDefaultPhaseMessageConfig(t *testing.T) {
	config := NewDefaultPhaseMessageConfig()

	assert.Equal(t, "osoba: 計画を作成します", config.Plan)
	assert.Equal(t, "osoba: 実装を開始します", config.Implement)
	assert.Equal(t, "osoba: レビューを開始します", config.Review)
}
