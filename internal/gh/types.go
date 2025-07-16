package gh

import "time"

// ghRepository はgh repo viewコマンドの出力を表す構造体
type ghRepository struct {
	Name             string      `json:"name"`
	Owner            ghOwner     `json:"owner"`
	Description      string      `json:"description"`
	DefaultBranchRef ghBranchRef `json:"defaultBranchRef"`
	IsPrivate        bool        `json:"isPrivate"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	URL              string      `json:"url"`
	SSHURL           string      `json:"sshUrl"`
	IsArchived       bool        `json:"isArchived"`
	IsFork           bool        `json:"isFork"`
}

// ghOwner はリポジトリの所有者を表す
type ghOwner struct {
	Login string `json:"login"`
}

// ghBranchRef はブランチ参照を表す
type ghBranchRef struct {
	Name string `json:"name"`
}

// ghIssue はgh issue listコマンドの出力を表す構造体
type ghIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    ghAuthor  `json:"author"`
	Labels    []ghLabel `json:"labels"`
}

// ghAuthor はIssueの作成者を表す
type ghAuthor struct {
	Login string `json:"login"`
}

// ghLabel はIssueのラベルを表す
type ghLabel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// ghRateLimit はAPIレート制限情報を表す
type ghRateLimit struct {
	Resources ghRateLimitResources `json:"resources"`
}

// ghRateLimitResources はレート制限のリソース別情報
type ghRateLimitResources struct {
	Core    ghRateLimitResource `json:"core"`
	Search  ghRateLimitResource `json:"search"`
	GraphQL ghRateLimitResource `json:"graphql"`
}

// ghRateLimitResource は個別のレート制限情報
type ghRateLimitResource struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}
