package utils

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *GitHubRepoInfo
		wantErr bool
	}{
		{
			name: "HTTPS URL with .git",
			url:  "https://github.com/douhashi/osoba.git",
			want: &GitHubRepoInfo{
				Owner: "douhashi",
				Repo:  "osoba",
			},
			wantErr: false,
		},
		{
			name: "HTTPS URL without .git",
			url:  "https://github.com/douhashi/osoba",
			want: &GitHubRepoInfo{
				Owner: "douhashi",
				Repo:  "osoba",
			},
			wantErr: false,
		},
		{
			name: "SSH URL with .git",
			url:  "git@github.com:douhashi/osoba.git",
			want: &GitHubRepoInfo{
				Owner: "douhashi",
				Repo:  "osoba",
			},
			wantErr: false,
		},
		{
			name: "SSH URL without .git",
			url:  "git@github.com:douhashi/osoba",
			want: &GitHubRepoInfo{
				Owner: "douhashi",
				Repo:  "osoba",
			},
			wantErr: false,
		},
		{
			name: "SSH URL with ssh:// prefix",
			url:  "ssh://git@github.com/douhashi/osoba.git",
			want: &GitHubRepoInfo{
				Owner: "douhashi",
				Repo:  "osoba",
			},
			wantErr: false,
		},
		{
			name: "SSH URL with ssh:// prefix without .git",
			url:  "ssh://git@github.com/douhashi/osoba",
			want: &GitHubRepoInfo{
				Owner: "douhashi",
				Repo:  "osoba",
			},
			wantErr: false,
		},
		{
			name: "Organization repository",
			url:  "https://github.com/golang/go.git",
			want: &GitHubRepoInfo{
				Owner: "golang",
				Repo:  "go",
			},
			wantErr: false,
		},
		{
			name:    "Invalid URL - not GitHub",
			url:     "https://gitlab.com/owner/repo.git",
			wantErr: true,
		},
		{
			name:    "Invalid URL - missing parts",
			url:     "https://github.com/owner",
			wantErr: true,
		},
		{
			name:    "Invalid URL - empty",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitHubURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGitHubURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Owner != tt.want.Owner || got.Repo != tt.want.Repo {
					t.Errorf("ParseGitHubURL() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
