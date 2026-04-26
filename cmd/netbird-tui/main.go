package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

var version = "dev" // injected via -ldflags at build time

func main() {
	// Register all subcommands explicitly — no init() side effects
	rootCmd.AddCommand(
		newStatusCmd(),
		newLsCmd(),
		newCurrentCmd(),
		newSaveCmd(),
		newUseCmd(),
		newRenameCmd(),
		newRmCmd(),
		newUpCmd(),
		newDownCmd(),
		newStartCmd(),
		newStopCmd(),
		newLogoutCmd(),
		newRestoreCmd(),
		newVersionCmd(),
	)
	rootCmd.InitDefaultCompletionCmd()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		os.Exit(1)
	}
}
