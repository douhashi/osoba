package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewPathManager(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		expected string
	}{
		{
			name:     "default base directory",
			baseDir:  "",
			expected: filepath.Join(os.Getenv("HOME"), ".local", "share", "osoba"),
		},
		{
			name:     "custom base directory",
			baseDir:  "/custom/path",
			expected: "/custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPathManager(tt.baseDir)
			if pm.DataDir() != tt.expected {
				t.Errorf("DataDir() = %v, want %v", pm.DataDir(), tt.expected)
			}
		})
	}
}

func TestPathManager_RunDir(t *testing.T) {
	pm := NewPathManager("/test/base")
	expected := "/test/base/run"
	if got := pm.RunDir(); got != expected {
		t.Errorf("RunDir() = %v, want %v", got, expected)
	}
}

func TestPathManager_LogDir(t *testing.T) {
	pm := NewPathManager("/test/base")
	tests := []struct {
		name           string
		repoIdentifier string
		expected       string
	}{
		{
			name:           "normal repository identifier",
			repoIdentifier: "owner-repo",
			expected:       "/test/base/logs/owner-repo",
		},
		{
			name:           "repository identifier with special chars",
			repoIdentifier: "owner/repo:branch",
			expected:       "/test/base/logs/owner_repo_branch",
		},
		{
			name:           "repository identifier with dots",
			repoIdentifier: "github.com/owner/repo",
			expected:       "/test/base/logs/github_com_owner_repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pm.LogDir(tt.repoIdentifier); got != tt.expected {
				t.Errorf("LogDir(%q) = %v, want %v", tt.repoIdentifier, got, tt.expected)
			}
		})
	}
}

func TestPathManager_PIDFile(t *testing.T) {
	pm := NewPathManager("/test/base")
	tests := []struct {
		name           string
		repoIdentifier string
		expected       string
	}{
		{
			name:           "normal repository identifier",
			repoIdentifier: "owner-repo",
			expected:       "/test/base/run/owner-repo.pid",
		},
		{
			name:           "repository identifier with special chars",
			repoIdentifier: "owner/repo:branch",
			expected:       "/test/base/run/owner_repo_branch.pid",
		},
		{
			name:           "repository identifier with dots",
			repoIdentifier: "github.com/owner/repo",
			expected:       "/test/base/run/github_com_owner_repo.pid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pm.PIDFile(tt.repoIdentifier); got != tt.expected {
				t.Errorf("PIDFile(%q) = %v, want %v", tt.repoIdentifier, got, tt.expected)
			}
		})
	}
}

func TestPathManager_EnsureDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping directory creation test on Windows")
	}

	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	if err := pm.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	dirs := []string{
		pm.RunDir(),
		filepath.Join(pm.DataDir(), "logs"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "owner-repo",
			expected: "owner-repo",
		},
		{
			name:     "forward slashes",
			input:    "owner/repo",
			expected: "owner_repo",
		},
		{
			name:     "colons",
			input:    "owner:repo",
			expected: "owner_repo",
		},
		{
			name:     "dots",
			input:    "github.com/owner/repo",
			expected: "github_com_owner_repo",
		},
		{
			name:     "multiple special characters",
			input:    "github.com/owner/repo:branch",
			expected: "github_com_owner_repo_branch",
		},
		{
			name:     "backslashes",
			input:    "owner\\repo",
			expected: "owner_repo",
		},
		{
			name:     "spaces",
			input:    "owner repo",
			expected: "owner_repo",
		},
	}

	pm := &pathManager{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pm.sanitizeIdentifier(tt.input); got != tt.expected {
				t.Errorf("sanitizeIdentifier(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPathManager_AllPIDFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping PID file test on Windows")
	}

	tmpDir := t.TempDir()
	pm := NewPathManager(tmpDir)

	if err := pm.EnsureDirectories(); err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	pidFiles := []string{
		"owner1-repo1.pid",
		"owner2-repo2.pid",
		"not-a-pid.txt",
	}

	for _, file := range pidFiles {
		path := filepath.Join(pm.RunDir(), file)
		if err := os.WriteFile(path, []byte("12345"), 0600); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	found, err := pm.AllPIDFiles()
	if err != nil {
		t.Fatalf("AllPIDFiles() error = %v", err)
	}

	if len(found) != 2 {
		t.Errorf("AllPIDFiles() returned %d files, want 2", len(found))
	}

	for _, path := range found {
		if !strings.HasSuffix(path, ".pid") {
			t.Errorf("AllPIDFiles() returned non-pid file: %s", path)
		}
	}
}
