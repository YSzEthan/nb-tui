package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"netbird-tui/internal/config"
	"netbird-tui/internal/netbird"
	"netbird-tui/internal/svc"
)

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [name]",
		Short: "切換到指定 profile（無參數時印出可用清單）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				names, err := config.ListProfiles()
				if err != nil {
					return err
				}
				for _, n := range names {
					fmt.Fprintln(cmd.OutOrStdout(), n)
				}
				return nil
			}
			name := args[0]
			activePath := config.DetectActive()
			src := config.ProfilePath(name)

			if err := netbird.WarmSudo(); err != nil {
				return fmt.Errorf("sudo refresh: %w", err)
			}
			if err := svc.Stop(); err != nil {
				return fmt.Errorf("stop service: %w", err)
			}
			if err := config.WriteActive(src, activePath); err != nil {
				return err
			}
			if err := svc.Start(); err != nil {
				return fmt.Errorf("start service: %w", err)
			}
			if err := netbird.Up(); err != nil {
				return fmt.Errorf("netbird up: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已切換至 %q\n", name)
			return nil
		},
	}
}
