package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhaseConfig(t *testing.T) {
	t.Run("PhaseConfigの基本的な構造", func(t *testing.T) {
		config := &PhaseConfig{
			Args:   []string{"--dangerously-skip-permissions"},
			Prompt: "/osoba:plan {{issue-number}}",
		}

		assert.Equal(t, []string{"--dangerously-skip-permissions"}, config.Args)
		assert.Equal(t, "/osoba:plan {{issue-number}}", config.Prompt)
	})
}

func TestClaudeConfig(t *testing.T) {
	t.Run("ClaudeConfigの基本的な構造", func(t *testing.T) {
		config := &ClaudeConfig{
			Phases: map[string]*PhaseConfig{
				"plan": {
					Args:   []string{"--dangerously-skip-permissions"},
					Prompt: "/osoba:plan {{issue-number}}",
				},
				"implement": {
					Args:   []string{},
					Prompt: "/osoba:implement {{issue-number}}",
				},
				"review": {
					Args:   []string{"--read-only"},
					Prompt: "/osoba:review {{issue-number}}",
				},
			},
		}

		assert.NotNil(t, config.Phases["plan"])
		assert.NotNil(t, config.Phases["implement"])
		assert.NotNil(t, config.Phases["review"])
		assert.Equal(t, []string{"--dangerously-skip-permissions"}, config.Phases["plan"].Args)
		assert.Equal(t, "/osoba:implement {{issue-number}}", config.Phases["implement"].Prompt)
		assert.Equal(t, []string{"--read-only"}, config.Phases["review"].Args)
	})
}

func TestNewDefaultClaudeConfig(t *testing.T) {
	t.Run("デフォルト設定の生成", func(t *testing.T) {
		config := NewDefaultClaudeConfig()

		assert.NotNil(t, config)
		assert.NotNil(t, config.Phases)
		assert.Contains(t, config.Phases, "plan")
		assert.Contains(t, config.Phases, "implement")
		assert.Contains(t, config.Phases, "review")

		// Plan phase
		assert.Equal(t, []string{"--dangerously-skip-permissions"}, config.Phases["plan"].Args)
		assert.Equal(t, "/osoba:plan {{issue-number}}", config.Phases["plan"].Prompt)

		// Implement phase
		assert.Empty(t, config.Phases["implement"].Args)
		assert.Equal(t, "/osoba:implement {{issue-number}}", config.Phases["implement"].Prompt)

		// Review phase
		assert.Equal(t, []string{"--read-only"}, config.Phases["review"].Args)
		assert.Equal(t, "/osoba:review {{issue-number}}", config.Phases["review"].Prompt)
	})
}

func TestClaudeConfig_GetPhase(t *testing.T) {
	t.Run("存在するフェーズの取得", func(t *testing.T) {
		config := NewDefaultClaudeConfig()

		phaseConfig, exists := config.GetPhase("plan")
		assert.True(t, exists)
		assert.NotNil(t, phaseConfig)
		assert.Equal(t, []string{"--dangerously-skip-permissions"}, phaseConfig.Args)
	})

	t.Run("存在しないフェーズの取得", func(t *testing.T) {
		config := NewDefaultClaudeConfig()

		phaseConfig, exists := config.GetPhase("unknown")
		assert.False(t, exists)
		assert.Nil(t, phaseConfig)
	})
}
