package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/netbird"
)

func newDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "斷線 (sudo netbird down)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return netbird.Down()
		},
	}
}
