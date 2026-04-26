package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/netbird"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "連線 (sudo netbird up)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return netbird.Up()
		},
	}
}
