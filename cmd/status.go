package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "現在の状態を表示",
		Long:  `実行中の開発セッションとその状態を表示します。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "現在のステータス:")
			fmt.Fprintln(cmd.OutOrStdout(), "  実行中のセッションはありません")
			return nil
		},
	}
	return cmd
}
