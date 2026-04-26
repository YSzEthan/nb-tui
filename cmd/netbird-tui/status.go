package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/netbird"
	"netbird-tui/internal/svc"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "印出目前 NetBird 狀態",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := netbird.GetStatus(cmd.Context())
			if err != nil {
				return err
			}
			svcStatus := "○ 未開啟"
			if svc.IsActive() {
				svcStatus = "● 開啟"
			}
			login := "○ 未登入"
			if s.IsLoggedIn() {
				login = "● 已登入"
			}
			conn := "○ 未連線"
			if s.IsConnected() {
				conn = "● " + s.IP
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Service:   %s\nLogin:     %s\nConnected: %s\nMgmt:      %s\n",
				svcStatus, login, conn, s.ManagementURL)
			return nil
		},
	}
}
