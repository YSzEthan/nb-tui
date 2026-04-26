package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
)

func newLsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "列出所有 profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			activePath := config.DetectActive()
			current, _ := config.CurrentProfileName(activePath)

			names, err := config.ListProfiles()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(無已儲存的 profile)")
				return nil
			}
			for _, name := range names {
				marker := "  "
				if name == current {
					marker = "★ "
				}
				mgmt, err := config.ReadProfileMgmtURL(name)
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", marker, name)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s%s  %s\n", marker, name, mgmt)
				}
			}
			return nil
		},
	}
}
