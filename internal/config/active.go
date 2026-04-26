package config

import (
	"fmt"
	"os/exec"
)

// ReadActiveRaw returns the raw bytes of the active config file (via sudo cat).
func ReadActiveRaw(path string) ([]byte, error) {
	out, err := exec.Command("sudo", "cat", path).Output()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return out, nil
}

// WriteActive copies src over the active config path using `sudo install`.
func WriteActive(src, activePath string) error {
	cmd := exec.Command("sudo", "install", "-m", "600", "-o", "root", "-g", "root", src, activePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("write active config: %w\n%s", err, out)
	}
	return nil
}
