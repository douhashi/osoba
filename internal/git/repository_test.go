package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRepositoryName(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (string, func())
		want    string
		wantErr bool
	}{
		{
			name: "正常系: リモートoriginから取得",
			setup: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				configDir := filepath.Join(gitDir, "config")

				err := os.MkdirAll(filepath.Dir(configDir), 0755)
				if err != nil {
					t.Fatal(err)
				}

				configContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = https://github.com/douhashi/osoba.git
`
				err = os.WriteFile(configDir, []byte(configContent), 0644)
				if err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			want:    "osoba",
			wantErr: false,
		},
		{
			name: "正常系: SSHリモートから取得",
			setup: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				configDir := filepath.Join(gitDir, "config")

				err := os.MkdirAll(filepath.Dir(configDir), 0755)
				if err != nil {
					t.Fatal(err)
				}

				configContent := `[core]
	repositoryformatversion = 0
[remote "origin"]
	url = git@github.com:douhashi/test-repo.git
`
				err = os.WriteFile(configDir, []byte(configContent), 0644)
				if err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			want:    "test-repo",
			wantErr: false,
		},
		{
			name: "異常系: Gitリポジトリではない",
			setup: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				cleanup := func() {
					os.RemoveAll(tmpDir)
				}
				return tmpDir, cleanup
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "異常系: リモートが設定されていない",
			setup: func(t *testing.T) (string, func()) {
				tmpDir := t.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				configDir := filepath.Join(gitDir, "config")

				err := os.MkdirAll(filepath.Dir(configDir), 0755)
				if err != nil {
					t.Fatal(err)
				}

				configContent := `[core]
	repositoryformatversion = 0
`
				err = os.WriteFile(configDir, []byte(configContent), 0644)
				if err != nil {
					t.Fatal(err)
				}

				cleanup := func() {
					os.RemoveAll(tmpDir)
				}

				return tmpDir, cleanup
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, cleanup := tt.setup(t)
			defer cleanup()

			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			err = os.Chdir(dir)
			if err != nil {
				t.Fatal(err)
			}

			got, err := GetRepositoryName()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRepositoryName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("GetRepositoryName() = %v, want %v", got, tt.want)
			}
		})
	}
}
