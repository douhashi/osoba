# Go コーディング規約

このドキュメントは、Go CLIツール開発におけるコーディング規約を定めています。[Effective Go](https://golang.org/doc/effective_go)と[Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)に準拠し、読みやすく保守しやすいコードを目指します。

## 基本原則

### Go Proverbs

Goの設計哲学を表す格言：

- **Don't communicate by sharing memory, share memory by communicating**
- **Concurrency is not parallelism**
- **The bigger the interface, the weaker the abstraction**
- **Make the zero value useful**
- **A little copying is better than a little dependency**
- **Clear is better than clever**
- **Errors are values**
- **Don't just check errors, handle them gracefully**

## コードレイアウト

### パッケージ名

- 小文字のみ使用、アンダースコアやキャメルケースは使わない
- 短く、簡潔で、発音可能な名前
- 複数形は避ける（`util` not `utils`）

```go
// 良い例
package user
package auth
package config

// 悪い例
package usersManager
package authentication_service
```

### ファイル名

- 小文字とアンダースコアを使用
- テストファイルは `_test.go` で終わる
- パッケージ名と重複する接頭辞は避ける

```
user.go         // not user_user.go
user_test.go
user_service.go
```

### インポート

```go
import (
    // 標準ライブラリ
    "context"
    "fmt"
    "io"
    
    // 空行で区切る
    
    // 外部ライブラリ
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    
    // 空行で区切る
    
    // 内部パッケージ
    "github.com/douhashi/osoba/internal/config"
    "github.com/douhashi/osoba/internal/tmux"
)
```

## 命名規則

### 基本的な命名スタイル

| 種類 | スタイル | 例 |
|------|----------|-----|
| パッケージ | lowercase | `user`, `config` |
| エクスポートされる名前 | PascalCase | `UserService`, `NewClient` |
| エクスポートされない名前 | camelCase | `userID`, `processData` |
| 定数 | PascalCase または UPPER_SNAKE_CASE | `MaxRetries`, `DEFAULT_TIMEOUT` |
| インターフェース | 名詞＋er形 | `Reader`, `Writer`, `Notifier` |

### 変数名

```go
// 短いスコープでは短い名前
for i := 0; i < 10; i++ {
    fmt.Println(i)
}

// 長いスコープでは説明的な名前
var configFilePath string
var userAuthenticationToken string

// 頭字語は一貫した大文字・小文字を使用
var xmlHTTPRequest  // not xmlHttpRequest
var userID         // not userId
var urlPath        // not urlPath
```

### レシーバー名

- 1〜2文字の短い名前
- 型名の最初の文字を小文字にしたもの
- `self`、`this`、`me` は使わない

```go
type Client struct {
    // ...
}

// 良い例
func (c *Client) Connect() error {
    // ...
}

// 悪い例
func (self *Client) Connect() error {
    // ...
}
```

## 型定義

### 構造体

```go
// フィールドはエクスポートの可否でグループ化
type Server struct {
    // エクスポートされるフィールド
    Host     string
    Port     int
    Timeout  time.Duration
    
    // エクスポートされないフィールド
    mu       sync.Mutex
    clients  map[string]*Client
    shutdown chan struct{}
}

// ゼロ値が有用になるように設計
type Config struct {
    Host    string        // デフォルト: ""
    Port    int          // デフォルト: 0
    Timeout time.Duration // デフォルト: 0
}
```

### インターフェース

```go
// 小さなインターフェースを推奨
type Reader interface {
    Read([]byte) (int, error)
}

// 複数のメソッドを持つ場合は、関連性の高いもののみ
type ReadWriter interface {
    Reader
    Writer
}

// インターフェースの実装チェック
var _ Reader = (*FileReader)(nil)
```

## 関数

### 関数設計

```go
// エラーは最後の戻り値
func LoadConfig(path string) (*Config, error) {
    // ...
}

// コンテキストは最初の引数
func ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
    // ...
}

// オプション引数は関数型で
type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) {
        s.timeout = d
    }
}

func NewServer(opts ...Option) *Server {
    s := &Server{
        timeout: 30 * time.Second, // デフォルト値
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

### defer の使用

```go
func ReadFile(filename string) ([]byte, error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer f.Close() // openの直後に記述
    
    return io.ReadAll(f)
}
```

## エラーハンドリング

### エラーの定義

```go
// センチネルエラー
var (
    ErrNotFound   = errors.New("not found")
    ErrInvalidArg = errors.New("invalid argument")
)

// カスタムエラー型
type ValidationError struct {
    Field string
    Err   error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed on field %s: %v", e.Field, e.Err)
}

func (e *ValidationError) Unwrap() error {
    return e.Err
}
```

### エラーのラップ

```go
func LoadUserData(id string) (*User, error) {
    data, err := db.Query(id)
    if err != nil {
        return nil, fmt.Errorf("load user data: %w", err)
    }
    
    var user User
    if err := json.Unmarshal(data, &user); err != nil {
        return nil, fmt.Errorf("unmarshal user data: %w", err)
    }
    
    return &user, nil
}
```

### エラーチェック

```go
// エラーは即座にチェック
data, err := LoadData()
if err != nil {
    return fmt.Errorf("failed to load data: %w", err)
}

// エラーの型アサーション
var valErr *ValidationError
if errors.As(err, &valErr) {
    log.Printf("Validation error on field: %s", valErr.Field)
}

// センチネルエラーの比較
if errors.Is(err, ErrNotFound) {
    // 404を返す
}
```

## 並行処理

### Goroutine

```go
// goroutineの起動時は必ずcontext対応
func Worker(ctx context.Context, jobs <-chan Job) {
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-jobs:
            if !ok {
                return
            }
            process(job)
        }
    }
}

// WaitGroupを使った同期
var wg sync.WaitGroup
for i := 0; i < workers; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        worker(id)
    }(i)
}
wg.Wait()
```

### チャネル

```go
// チャネルの方向を明示
func producer() <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch)
        for i := 0; i < 10; i++ {
            ch <- i
        }
    }()
    return ch
}

func consumer(ch <-chan int) {
    for n := range ch {
        fmt.Println(n)
    }
}
```

## テスト

### テスト関数

```go
func TestUserService_Create(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateUserInput
        want    *User
        wantErr bool
    }{
        {
            name: "valid input",
            input: &CreateUserInput{
                Name:  "John Doe",
                Email: "john@example.com",
            },
            want: &User{
                Name:  "John Doe",
                Email: "john@example.com",
            },
            wantErr: false,
        },
        {
            name: "empty name",
            input: &CreateUserInput{
                Name:  "",
                Email: "john@example.com",
            },
            want:    nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewUserService()
            got, err := svc.Create(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Create() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### テストヘルパー

```go
func setupTestDB(t *testing.T) (*sql.DB, func()) {
    t.Helper()
    
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("failed to open database: %v", err)
    }
    
    cleanup := func() {
        db.Close()
    }
    
    return db, cleanup
}

func TestUserRepository(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // テストロジック
}
```

## CLIツール固有のベストプラクティス

### Cobraを使ったコマンド定義

```go
package cmd

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
    Use:   "osoba",
    Short: "自律的ソフトウェア開発支援ツール",
    Long: `osobaは、tmux + git worktree + claudeを組み合わせた
自律的なソフトウェア開発を支援するCLIツールです。`,
    
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // 設定ファイルの読み込み
        if err := initConfig(); err != nil {
            return fmt.Errorf("failed to initialize config: %w", err)
        }
        return nil
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.PersistentFlags().StringP("config", "c", "", "設定ファイルのパス")
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "詳細出力")
    
    viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}
```

### 設定管理

```go
type Config struct {
    GitHub struct {
        Token        string        `mapstructure:"token"`
        PollInterval time.Duration `mapstructure:"poll_interval"`
    } `mapstructure:"github"`
    
    Tmux struct {
        SessionPrefix string `mapstructure:"session_prefix"`
    } `mapstructure:"tmux"`
}

func LoadConfig(path string) (*Config, error) {
    viper.SetConfigFile(path)
    
    // デフォルト値の設定
    viper.SetDefault("github.poll_interval", 5*time.Minute)
    viper.SetDefault("tmux.session_prefix", "osoba-")
    
    // 環境変数の読み込み
    viper.SetEnvPrefix("OSOBA")
    viper.AutomaticEnv()
    
    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    
    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    
    return &cfg, nil
}
```

## プロジェクト構造

推奨されるGo CLIツールのプロジェクト構造：

```
osoba/
├── cmd/
│   ├── root.go      # ルートコマンド
│   ├── watch.go     # watchサブコマンド
│   └── open.go      # openサブコマンド
├── internal/        # 外部に公開しない内部パッケージ
│   ├── config/
│   ├── github/
│   ├── tmux/
│   └── watcher/
├── pkg/            # 外部に公開可能なパッケージ
│   └── models/
├── main.go
├── go.mod
├── go.sum
├── .gitignore
├── Makefile
└── README.md
```

## ツール

### 必須ツール

- **gofmt/goimports**: コードフォーマッター
- **go vet**: 静的解析
- **go test**: テストランナー

### 標準ツールの活用

標準のGoツールを活用してコード品質を維持します：

```bash
# コードフォーマット
go fmt ./...

# 静的解析
go vet ./...

# インポートの整理
goimports -w .

# テスト実行
go test -v ./...
```

## まとめ

このコーディング規約は、Goの公式ガイドラインに基づいたものです。重要なのは：

1. **シンプルさ** - 複雑さより明快さを選ぶ
2. **一貫性** - プロジェクト内で一貫したスタイルを保つ
3. **慣用的** - Go wayに従う

詳細については以下を参照してください：
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)