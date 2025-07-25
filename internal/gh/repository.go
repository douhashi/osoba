package gh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/douhashi/osoba/internal/github"
)

// GetRepository はリポジトリ情報を取得する
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	// gh repo view コマンドを実行
	output, err := c.executor.Execute(ctx, "gh", "repo", "view", fmt.Sprintf("%s/%s", owner, repo), "--json", "name,owner,description,defaultBranchRef,isPrivate,createdAt,updatedAt,url,sshUrl,isArchived,isFork")
	if err != nil {
		var execErr *ExecError
		if errors.As(err, &execErr) {
			if strings.Contains(execErr.Stderr, "Could not resolve to a Repository") {
				return nil, errors.New("repository not found")
			}
		}
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// JSONをパース
	var ghRepo ghRepository
	if err := json.Unmarshal([]byte(output), &ghRepo); err != nil {
		return nil, fmt.Errorf("failed to parse repository data: %w", err)
	}

	// Repository型に変換
	return convertToRepository(&ghRepo), nil
}

// convertToRepository はgh用の構造体をgithub.Repository型に変換する
func convertToRepository(ghRepo *ghRepository) *github.Repository {
	repo := &github.Repository{
		Name:        github.String(ghRepo.Name),
		Description: github.String(ghRepo.Description),
		Private:     github.Bool(ghRepo.IsPrivate),
		HTMLURL:     github.String(ghRepo.URL),
		Fork:        github.Bool(ghRepo.IsFork),
		Owner: &github.User{
			Login: github.String(ghRepo.Owner.Login),
		},
	}

	return repo
}
