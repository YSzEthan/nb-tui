package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
)

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "重新命名 profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.RenameProfile(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%q → %q\n", args[0], args[1])
			return nil
		},
	}
}
