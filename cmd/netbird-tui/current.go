package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
)

func newCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "印出目前使用的 profile 名稱",
		RunE: func(cmd *cobra.Command, args []string) error {
			activePath := config.DetectActive()
			name, err := config.CurrentProfileName(activePath)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), "(unsaved)")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), name)
			return nil
		},
	}
}
