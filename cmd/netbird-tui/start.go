package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/svc"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "啟動 netbird service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return svc.Start()
		},
	}
}
