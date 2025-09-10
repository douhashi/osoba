package cmd

import (
	"fmt"
	"os"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile  string
	verbose  bool
	logLevel string
	rootCmd  *cobra.Command
	appLog   logger.Logger
)

func init() {
	rootCmd = newRootCmd()

	// サブコマンドの追加
	addCommands()
}

func addCommands() {
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newOpenCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newCleanCmd())
	rootCmd.AddCommand(newResizeCmd())
}

// NewRootCmd creates a new root command with all subcommands
func NewRootCmd() *cobra.Command {
	cmd := newRootCmd()
	// サブコマンドを追加
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newOpenCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newCleanCmd())
	cmd.AddCommand(newResizeCmd())
	return cmd
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osoba",
		Short: "自律的ソフトウェア開発支援ツール",
		Long: `osobaは、tmux + git worktree + claudeを組み合わせた
自律的なソフトウェア開発を支援するCLIツールです。`,
		Version: version.Get().Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 設定ファイルを先に読み込む
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			// ロガーの初期化
			var err error
			appLog, err = initLogger()
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "設定ファイルのパス")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細出力")
	cmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "ログレベル (debug, info, warn, error)")

	viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("log.level", cmd.PersistentFlags().Lookup("log-level"))

	return cmd
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initConfig() error {
	if cfgFile != "" {
		// ファイルの存在を確認
		if _, err := os.Stat(cfgFile); err != nil {
			// ファイルが存在しない場合は、エラーを返さずにデフォルト値を使用
			if os.IsNotExist(err) {
				// 設定ファイルが見つからない場合はデフォルト値を使用
				viper.SetDefault("github.poll_interval", "5s")
				viper.SetDefault("github.labels.plan", "status:needs-plan")
				viper.SetDefault("github.labels.ready", "status:ready")
				viper.SetDefault("github.labels.review", "status:review-requested")
				viper.SetDefault("tmux.session_prefix", "osoba-")
				return nil
			}
			return fmt.Errorf("failed to access config file: %w", err)
		}
		viper.SetConfigFile(cfgFile)
	} else {
		// カレントディレクトリから設定ファイルを検索
		viper.AddConfigPath(".")
		viper.SetConfigName(".osoba")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("OSOBA")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return nil
}

// initLogger はロガーを初期化する
func initLogger() (logger.Logger, error) {
	// 設定を読み込む
	cfg := config.NewConfig()
	if cfgFile != "" {
		if err := cfg.Load(cfgFile); err != nil {
			// 設定ファイルの読み込みに失敗した場合でも、デフォルト値を使って続行
			cfg = config.NewConfig()
		}
	} else {
		cfg.LoadOrDefault("")
	}

	// コマンドラインオプションで上書き
	if logLevel != "" {
		cfg.Log.Level = logLevel
	}

	// verboseフラグが設定されている場合はdebugレベルに設定
	if verbose {
		cfg.Log.Level = "debug"
	}

	return cfg.CreateLogger()
}
