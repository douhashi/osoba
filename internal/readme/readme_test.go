package readme

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestREADMEStructure(t *testing.T) {
	readmePath := filepath.Join("..", "..", "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	sections := extractSections(lines)

	t.Run("必要なセクションの順序が正しい", func(t *testing.T) {
		expectedOrder := []string{
			"概要",
			"セキュリティ上の注意事項",
			"必要な環境",
			"インストール",
			"クイックスタート",
			"動作イメージ",
		}

		actualOrder := getSectionOrder(sections, expectedOrder)

		for i, expected := range expectedOrder {
			if i >= len(actualOrder) {
				t.Errorf("Section '%s' not found", expected)
				continue
			}
			if actualOrder[i] != expected {
				t.Errorf("Section order incorrect at position %d: expected '%s', got '%s'", i, expected, actualOrder[i])
			}
		}
	})

	t.Run("削除すべきセクションが存在しない", func(t *testing.T) {
		sectionsToDelete := []string{
			"3. ワークフロー例",
			"内部動作の詳細",
			"GitHubアクセス方法",
		}

		for _, section := range sectionsToDelete {
			if _, exists := sections[section]; exists {
				t.Errorf("Section '%s' should be deleted but still exists", section)
			}
		}
	})

	t.Run("基本的な使い方セクションに削除すべき記述が存在しない", func(t *testing.T) {
		// サブセクションとして確認
		basicUsageContent, exists := sections["クイックスタート > 2. 基本的な使い方"]
		if !exists {
			// 全体のクイックスタートセクションから確認
			quickStartContent, exists := sections["クイックスタート"]
			if !exists {
				t.Fatal("クイックスタート section not found")
			}
			basicUsageContent = quickStartContent
		}

		prohibitedText := "# 別のターミナルを開き、セッションに接続"
		if strings.Contains(basicUsageContent, prohibitedText) {
			t.Errorf("基本的な使い方 section contains prohibited text: '%s'", prohibitedText)
		}
	})

	t.Run("セキュリティ上の注意事項が必要な環境の前に配置されている", func(t *testing.T) {
		securityIndex := findSectionIndex(sections, "セキュリティ上の注意事項")
		environmentIndex := findSectionIndex(sections, "必要な環境")

		if securityIndex == -1 {
			t.Error("セキュリティ上の注意事項 section not found")
		}
		if environmentIndex == -1 {
			t.Error("必要な環境 section not found")
		}
		if securityIndex >= environmentIndex {
			t.Error("セキュリティ上の注意事項 should come before 必要な環境")
		}
	})
}

func extractSections(lines []string) map[string]string {
	sections := make(map[string]string)
	currentSection := ""
	var currentContent strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentSection != "" {
				sections[currentSection] = currentContent.String()
			}
			currentSection = strings.TrimPrefix(line, "## ")
			currentContent.Reset()
		} else if strings.HasPrefix(line, "### ") && currentSection != "" {
			subsectionName := currentSection + " > " + strings.TrimPrefix(line, "### ")
			if currentSection != "" {
				sections[currentSection] = currentContent.String()
			}
			currentSection = subsectionName
			currentContent.Reset()
		} else if currentSection != "" {
			currentContent.WriteString(line + "\n")
		}
	}

	if currentSection != "" {
		sections[currentSection] = currentContent.String()
	}

	return sections
}

func getSectionOrder(sections map[string]string, expectedSections []string) []string {
	var order []string
	for _, expected := range expectedSections {
		if _, exists := sections[expected]; exists {
			order = append(order, expected)
		}
	}
	return order
}

func findSectionIndex(sections map[string]string, sectionName string) int {
	file, err := os.Open(filepath.Join("..", "..", "README.md"))
	if err != nil {
		return -1
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	index := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if strings.TrimPrefix(line, "## ") == sectionName {
				return index
			}
			index++
		}
	}
	return -1
}
