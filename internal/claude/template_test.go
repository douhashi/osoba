package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateVariables(t *testing.T) {
	t.Run("TemplateVariablesの基本的な構造", func(t *testing.T) {
		vars := &TemplateVariables{
			IssueNumber: 123,
			IssueTitle:  "Test Issue",
			RepoName:    "douhashi/osoba",
		}

		assert.Equal(t, 123, vars.IssueNumber)
		assert.Equal(t, "Test Issue", vars.IssueTitle)
		assert.Equal(t, "douhashi/osoba", vars.RepoName)
	})
}

func TestExpandTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     *TemplateVariables
		want     string
	}{
		{
			name:     "単一の変数置換",
			template: "/osoba:plan {{issue-number}}",
			vars:     &TemplateVariables{IssueNumber: 46},
			want:     "/osoba:plan 46",
		},
		{
			name:     "複数の変数置換",
			template: "Issue #{{issue-number}}: {{issue-title}}",
			vars: &TemplateVariables{
				IssueNumber: 46,
				IssueTitle:  "Claude起動機能",
			},
			want: "Issue #46: Claude起動機能",
		},
		{
			name:     "リポジトリ名の置換",
			template: "Working on {{repo-name}} issue #{{issue-number}}",
			vars: &TemplateVariables{
				IssueNumber: 46,
				RepoName:    "douhashi/osoba",
			},
			want: "Working on douhashi/osoba issue #46",
		},
		{
			name:     "変数なしのテンプレート",
			template: "No variables here",
			vars:     &TemplateVariables{},
			want:     "No variables here",
		},
		{
			name:     "全ての変数を含むテンプレート",
			template: "[{{repo-name}}] #{{issue-number}}: {{issue-title}}",
			vars: &TemplateVariables{
				IssueNumber: 46,
				IssueTitle:  "Claude起動機能と設定管理の実装",
				RepoName:    "douhashi/osoba",
			},
			want: "[douhashi/osoba] #46: Claude起動機能と設定管理の実装",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandTemplate(tt.template, tt.vars)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExpandTemplate_EmptyValues(t *testing.T) {
	t.Run("空の値での置換", func(t *testing.T) {
		template := "Issue #{{issue-number}}: {{issue-title}}"
		vars := &TemplateVariables{
			IssueNumber: 0,
			IssueTitle:  "",
		}

		got := ExpandTemplate(template, vars)
		assert.Equal(t, "Issue #0: ", got)
	})
}

func TestExpandTemplate_SpecialCharacters(t *testing.T) {
	t.Run("特殊文字を含むタイトル", func(t *testing.T) {
		template := "{{issue-title}}"
		vars := &TemplateVariables{
			IssueTitle: "feat: Claude起動機能 & 設定管理",
		}

		got := ExpandTemplate(template, vars)
		assert.Equal(t, "feat: Claude起動機能 & 設定管理", got)
	})
}
