package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/svc"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "停止 netbird service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return svc.Stop()
		},
	}
}
