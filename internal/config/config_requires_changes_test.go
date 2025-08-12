package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLabelConfig_RequiresChangesField(t *testing.T) {
	t.Run("default value for RequiresChanges field", func(t *testing.T) {
		cfg := NewConfig()

		// デフォルト値が設定されていることを確認
		assert.Equal(t, "status:requires-changes", cfg.GitHub.Labels.RequiresChanges)
	})

	t.Run("GetLabels includes requires-changes label", func(t *testing.T) {
		cfg := &Config{
			GitHub: GitHubConfig{
				Labels: LabelConfig{
					Plan:            "status:needs-plan",
					Ready:           "status:ready",
					Review:          "status:review-requested",
					RequiresChanges: "status:requires-changes",
					Revising:        "status:revising",
				},
			},
		}

		labels := cfg.GetLabels()

		// 5つのラベルが含まれることを確認
		assert.Len(t, labels, 5)
		assert.Contains(t, labels, "status:needs-plan")
		assert.Contains(t, labels, "status:ready")
		assert.Contains(t, labels, "status:review-requested")
		assert.Contains(t, labels, "status:requires-changes")
		assert.Contains(t, labels, "status:revising")
	})

	t.Run("backward compatibility - empty RequiresChanges field", func(t *testing.T) {
		cfg := &Config{
			GitHub: GitHubConfig{
				Labels: LabelConfig{
					Plan:            "status:needs-plan",
					Ready:           "status:ready",
					Review:          "status:review-requested",
					RequiresChanges: "", // 空の場合
				},
			},
		}

		// SetDefaults が空フィールドにデフォルト値を設定することを確認
		cfg.SetDefaults()
		assert.Equal(t, "status:requires-changes", cfg.GitHub.Labels.RequiresChanges)
	})
}
