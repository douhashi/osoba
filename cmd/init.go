package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "プロジェクトを初期化",
		Long:  `osobaプロジェクトのための初期設定を行います。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "初期化が完了しました")
			return nil
		},
	}
	return cmd
}
