package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocumentationExists verifies that all required documentation files exist
func TestDocumentationExists(t *testing.T) {
	requiredDocs := []string{
		"docs/development/architecture.md",
		"docs/development/workflow-specification.md",
		"docs/development/testing-strategy.md",
		"docs/development/troubleshooting-guide.md",
	}

	for _, docPath := range requiredDocs {
		t.Run(docPath, func(t *testing.T) {
			if _, err := os.Stat(docPath); os.IsNotExist(err) {
				t.Errorf("Required documentation file does not exist: %s", docPath)
			}
		})
	}
}

// TestArchitectureDocumentContent verifies the architecture document contains required sections
func TestArchitectureDocumentContent(t *testing.T) {
	content, err := os.ReadFile("docs/development/architecture.md")
	if err != nil {
		t.Fatalf("Failed to read architecture.md: %v", err)
	}

	contentStr := string(content)
	requiredSections := []string{
		"# アーキテクチャ設計書",
		"## システム全体構成",
		"## コンポーネント詳細",
		"### internal/watcher",
		"### internal/github",
		"### internal/tmux",
		"### internal/git",
		"### internal/claude",
		"## データフロー",
		"## 依存関係",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Architecture document missing required section: %s", section)
		}
	}
}

// TestWorkflowSpecificationContent verifies the workflow specification document
func TestWorkflowSpecificationContent(t *testing.T) {
	content, err := os.ReadFile("docs/development/workflow-specification.md")
	if err != nil {
		t.Fatalf("Failed to read workflow-specification.md: %v", err)
	}

	contentStr := string(content)
	requiredSections := []string{
		"# ワークフロー仕様書",
		"## Issue検知メカニズム",
		"## 3フェーズ処理フロー",
		"### 計画フェーズ",
		"### 実装フェーズ",
		"### レビューフェーズ",
		"## ラベル遷移仕様",
		"## エラーハンドリング",
		"## tmux統合",
		"## git worktree管理",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Workflow specification document missing required section: %s", section)
		}
	}
}

// TestTestingStrategyContent verifies the testing strategy document
func TestTestingStrategyContent(t *testing.T) {
	content, err := os.ReadFile("docs/development/testing-strategy.md")
	if err != nil {
		t.Fatalf("Failed to read testing-strategy.md: %v", err)
	}

	contentStr := string(content)
	requiredSections := []string{
		"# テスト戦略",
		"## テスト分類",
		"## ユニットテスト",
		"## 統合テスト",
		"## テスト実行方針",
		"## カバレッジ要件",
		"## テストデータ管理",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Testing strategy document missing required section: %s", section)
		}
	}
}

// TestTroubleshootingGuideContent verifies the troubleshooting guide document
func TestTroubleshootingGuideContent(t *testing.T) {
	content, err := os.ReadFile("docs/development/troubleshooting-guide.md")
	if err != nil {
		t.Fatalf("Failed to read troubleshooting-guide.md: %v", err)
	}

	contentStr := string(content)
	requiredSections := []string{
		"# トラブルシューティングガイド",
		"## 一般的な問題と解決方法",
		"## Issue検知の問題",
		"## tmuxセッションの問題",
		"## git worktreeの問題",
		"## Claude実行の問題",
		"## ラベル遷移の問題",
		"## デバッグ手法",
		"## ログ解析",
	}

	for _, section := range requiredSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Troubleshooting guide document missing required section: %s", section)
		}
	}
}

// TestDocumentationFileStructure verifies that documentation follows the expected structure
func TestDocumentationFileStructure(t *testing.T) {
	docsDir := "docs/development"

	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".md") {
			// Check that markdown files have proper title structure
			content, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("Failed to read %s: %v", path, err)
				return nil
			}

			contentStr := string(content)
			if !strings.HasPrefix(contentStr, "# ") {
				t.Errorf("Documentation file %s should start with a level 1 header", path)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk documentation directory: %v", err)
	}
}

// TestMermaidDiagramsPresent verifies that architecture diagrams are present
func TestMermaidDiagramsPresent(t *testing.T) {
	content, err := os.ReadFile("docs/development/architecture.md")
	if err != nil {
		t.Fatalf("Failed to read architecture.md: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "```mermaid") {
		t.Error("Architecture document should contain Mermaid diagrams")
	}
}
