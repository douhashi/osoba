package main

import (
	"os"
	"testing"
)

func TestProjectStructure(t *testing.T) {
	t.Run("go.modファイルが存在する", func(t *testing.T) {
		if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
			t.Error("go.mod file does not exist")
		}
	})

	t.Run("必要なディレクトリが存在する", func(t *testing.T) {
		dirs := []string{"cmd", "internal", "pkg"}
		for _, dir := range dirs {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				t.Errorf("directory %s does not exist", dir)
			}
		}
	})

	t.Run(".gitignoreファイルが存在する", func(t *testing.T) {
		if _, err := os.Stat(".gitignore"); os.IsNotExist(err) {
			t.Error(".gitignore file does not exist")
		}
	})

	t.Run("main.goファイルが存在する", func(t *testing.T) {
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			t.Error("main.go file does not exist")
		}
	})
}

func TestGoModContent(t *testing.T) {
	t.Run("go.modにモジュール名が含まれている", func(t *testing.T) {
		content, err := os.ReadFile("go.mod")
		if err != nil {
			t.Fatalf("failed to read go.mod: %v", err)
		}

		expectedModule := "module github.com/douhashi/osoba"
		if !contains(string(content), expectedModule) {
			t.Errorf("go.mod does not contain expected module name: %s", expectedModule)
		}
	})
}

func TestBuildable(t *testing.T) {
	t.Run("go buildが成功する", func(t *testing.T) {
		// This test will be checked by running go test itself
		// If the project builds, the test passes
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr, 0)
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
