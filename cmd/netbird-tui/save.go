package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
)

func newSaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "save <name>",
		Short: "將目前 config 儲存為 profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			activePath := config.DetectActive()
			if err := config.SaveProfile(name, activePath); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已儲存為 %q\n", name)
			return nil
		},
	}
}
