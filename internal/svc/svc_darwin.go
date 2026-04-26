//go:build darwin

package svc

import (
	"fmt"
	"os/exec"
	"strings"
)

const plist = "/Library/LaunchDaemons/io.netbird.client.plist"

func IsActive() bool {
	out, err := exec.Command("launchctl", "print", "system/io.netbird.client").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "state = running")
}

func Start() error {
	if out, err := exec.Command("sudo", "launchctl", "bootstrap", "system", plist).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap netbird: %w\n%s", err, out)
	}
	return nil
}

func Stop() error {
	if out, err := exec.Command("sudo", "launchctl", "bootout", "system/io.netbird.client").CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootout netbird: %w\n%s", err, out)
	}
	return nil
}
