package netbird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// GetStatus runs `netbird status --json` and returns a parsed Status.
func GetStatus(ctx context.Context) (*Status, error) {
	tctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(tctx, "netbird", "status", "--json").Output()
	if err != nil {
		// Daemon unreachable — return empty status so callers can handle gracefully
		return &Status{}, nil
	}
	var s Status
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, fmt.Errorf("netbird status parse: %w", err)
	}
	return &s, nil
}

// Up runs `sudo netbird up`.
func Up() error {
	return RunSudo("netbird", "up")
}

// Down runs `sudo netbird down`.
func Down() error {
	return RunSudo("netbird", "down")
}

// SSHCmd returns the exec.Cmd for `sudo netbird ssh <peer>` — caller runs it interactively.
func SSHCmd(peer string) *exec.Cmd {
	return exec.Command("sudo", "netbird", "ssh", peer)
}

// --- sudo helpers ---

// RunSudo runs a command with sudo, using the current credential cache (non-interactive).
func RunSudo(name string, args ...string) error {
	cmd := exec.Command("sudo", append([]string{name}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w\n%s", name, err, stderr.String())
	}
	return nil
}

// RefreshSudo refreshes the sudo credential timestamp (sudo -v).
func RefreshSudo() error {
	return exec.Command("sudo", "-v").Run()
}

// NeedsSudoPrompt returns true if the credential cache has expired (sudo -n true fails).
func NeedsSudoPrompt() bool {
	return exec.Command("sudo", "-n", "true").Run() != nil
}

// StartKeepalive starts a goroutine that refreshes sudo credentials every interval.
func StartKeepalive(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = exec.Command("sudo", "-n", "-v").Run()
			}
		}
	}()
}

// sudoWarmOnce ensures sudo -v is called at most once per process lifetime.
var sudoWarmOnce sync.Once

// WarmSudo runs sudo -v once if not already done. Returns error if sudo prompt needed interactively.
func WarmSudo() error {
	var err error
	sudoWarmOnce.Do(func() {
		err = RefreshSudo()
	})
	return err
}
