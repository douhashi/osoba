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

// RateLimit は個別のレート制限情報（エクスポート用）
type RateLimit struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

// RateLimitResources はレート制限のリソース別情報（エクスポート用）
type RateLimitResources struct {
	Core    RateLimit `json:"core"`
	Search  RateLimit `json:"search"`
	GraphQL RateLimit `json:"graphql"`
}

// RateLimitResponse はAPIレート制限情報（エクスポート用）
type RateLimitResponse struct {
	Resources RateLimitResources `json:"resources"`
}

// Issue はIssue情報（エクスポート用）
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    Author    `json:"author"`
	Labels    []Label   `json:"labels"`
}

// Author はIssueの作成者（エクスポート用）
type Author struct {
	Login string `json:"login"`
}

// Label はIssueのラベル（エクスポート用）
type Label struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// Repository はリポジトリ情報（エクスポート用）
type Repository struct {
	Name             string    `json:"name"`
	Owner            Owner     `json:"owner"`
	Description      string    `json:"description"`
	DefaultBranchRef BranchRef `json:"defaultBranchRef"`
	IsPrivate        bool      `json:"isPrivate"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	URL              string    `json:"url"`
	SSHURL           string    `json:"sshUrl"`
	IsArchived       bool      `json:"isArchived"`
	IsFork           bool      `json:"isFork"`
}

// Owner はリポジトリの所有者（エクスポート用）
type Owner struct {
	Login string `json:"login"`
}

// BranchRef はブランチ参照（エクスポート用）
type BranchRef struct {
	Name string `json:"name"`
}
