package main

import (
	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "登出並備份目前 config",
		RunE: func(cmd *cobra.Command, args []string) error {
			activePath := config.DetectActive()
			return config.CoreLogout(activePath)
		},
	}
}
