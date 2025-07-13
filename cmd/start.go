package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "開発セッションを開始",
		Long:  `特定のGitHub Issueに対する開発セッションを開始します。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "開発セッションを開始しました")
			return nil
		},
	}
	return cmd
}
