// Package config manages the NetBird active config path, user-owned profile files,
// and logout/restore operations. Profiles are stored in ~/.config/netbird-profiles/.
package config

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

const (
	pathNew    = "/var/lib/netbird/default.json"
	pathLegacy = "/etc/netbird/config.json"
)

// DetectActive returns the path to the active NetBird config file.
func DetectActive() string {
	if sudoFileExists(pathNew) {
		return pathNew
	}
	if sudoFileExists(pathLegacy) {
		return pathLegacy
	}
	// Linux fallback: parse systemctl cat output
	if runtime.GOOS == "linux" {
		if p := detectViaSystemctl(); p != "" {
			return p
		}
	}
	return pathNew
}

func sudoFileExists(path string) bool {
	return exec.Command("sudo", "test", "-f", path).Run() == nil
}

var configFlagRe = regexp.MustCompile(`--config[= ]([^ "]+)`)

func detectViaSystemctl() string {
	out, err := exec.Command("systemctl", "cat", "netbird").Output()
	if err != nil {
		return ""
	}
	if m := configFlagRe.FindSubmatch(out); len(m) > 1 {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}
