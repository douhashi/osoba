package version

var (
	// Version はビルド時に設定されるバージョン情報
	Version = "dev"
	// Commit はビルド時に設定されるGitコミットハッシュ
	Commit = "none"
	// Date はビルド時に設定されるビルド日時
	Date = "unknown"
)

// Info はバージョン情報を保持する構造体
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Get は現在のバージョン情報を返す
func Get() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}
