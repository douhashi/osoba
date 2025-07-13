package cmd

import (
	"fmt"
	"os"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	rootCmd *cobra.Command
)

func init() {
	rootCmd = newRootCmd()

	// サブコマンドの追加
	addCommands()
}

func addCommands() {
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newStatusCmd())
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "osoba",
		Short: "自律的ソフトウェア開発支援ツール",
		Long: `osobaは、tmux + git worktree + claudeを組み合わせた
自律的なソフトウェア開発を支援するCLIツールです。`,
		Version: version.Get().Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// ロギング設定
			logger.SetVerbose(verbose)

			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "設定ファイルのパス")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細出力")

	viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))

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
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		viper.AddConfigPath(home + "/.config/osoba")
		viper.AddConfigPath(home)
		viper.SetConfigName("osoba")
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
