package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "刪除 profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.RemoveProfile(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已刪除 %q\n", args[0])
			return nil
		},
	}
}
