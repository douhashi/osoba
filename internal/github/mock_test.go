package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v67/github"
)

// mockGitHubService はテスト用のGitHub APIサービスのモック
type mockGitHubService struct {
	getRepoFunc    func(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
	listIssuesFunc func(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	rateLimitsFunc func(ctx context.Context) (*github.RateLimits, *github.Response, error)
}

func (m *mockGitHubService) Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
	if m.getRepoFunc != nil {
		return m.getRepoFunc(ctx, owner, repo)
	}
	return nil, nil, errors.New("not implemented")
}

func (m *mockGitHubService) ListByRepo(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	if m.listIssuesFunc != nil {
		return m.listIssuesFunc(ctx, owner, repo, opts)
	}
	return nil, nil, errors.New("not implemented")
}

// mockGitHubClient はテスト用のGitHubクライアントのモック
type mockGitHubClient struct {
	repositories *mockGitHubService
	issues       *mockGitHubService
	rateLimits   func(ctx context.Context) (*github.RateLimits, *github.Response, error)
}

func newMockGitHubClient() *mockGitHubClient {
	return &mockGitHubClient{
		repositories: &mockGitHubService{},
		issues:       &mockGitHubService{},
	}
}

// テスト用のヘルパー関数
func newTestRepository(name string) *github.Repository {
	return &github.Repository{
		Name: github.String(name),
		Owner: &github.User{
			Login: github.String("douhashi"),
		},
	}
}

func newTestIssue(number int, title string, labels []string) *github.Issue {
	issue := &github.Issue{
		Number: github.Int(number),
		Title:  github.String(title),
		State:  github.String("open"),
	}

	for _, label := range labels {
		issue.Labels = append(issue.Labels, &github.Label{
			Name: github.String(label),
		})
	}

	return issue
}

func newTestResponse(nextPage int) *github.Response {
	return &github.Response{
		NextPage: nextPage,
	}
}

func newTestRateLimits(limit, remaining int) *github.RateLimits {
	return &github.RateLimits{
		Core: &github.Rate{
			Limit:     limit,
			Remaining: remaining,
		},
	}
}
