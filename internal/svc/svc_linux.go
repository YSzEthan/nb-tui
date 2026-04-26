//go:build linux

// Package svc provides service start/stop/status operations.
// Linux implementation uses systemctl; macOS uses launchctl (svc_darwin.go).
package svc

import (
	"fmt"
	"os/exec"
)

func IsActive() bool {
	return exec.Command("systemctl", "is-active", "--quiet", "netbird").Run() == nil
}

func Start() error {
	if out, err := exec.Command("sudo", "systemctl", "start", "netbird").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl start netbird: %w\n%s", err, out)
	}
	return nil
}

func Stop() error {
	if out, err := exec.Command("sudo", "systemctl", "stop", "netbird").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl stop netbird: %w\n%s", err, out)
	}
	return nil
}
