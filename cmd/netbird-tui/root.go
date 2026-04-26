package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:           "NetBird-TUI",
	Short:         "NetBird VPN 管理 TUI",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return ui.Run(cmd.Context())
	},
}
