package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
	"netbird-tui/internal/netbird"
	"netbird-tui/internal/svc"
)

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore-last",
		Short: "還原最後一次登出的 config",
		RunE: func(cmd *cobra.Command, args []string) error {
			activePath := config.DetectActive()
			if err := svc.Stop(); err != nil {
				return err
			}
			if err := config.RestoreLast(activePath); err != nil {
				return err
			}
			if err := svc.Start(); err != nil {
				return err
			}
			return netbird.Up()
		},
	}
}
