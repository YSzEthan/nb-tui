package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProfileDir returns ~/.config/netbird-profiles/
func ProfileDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "netbird-profiles")
}

func lastLogoutPath() string {
	return filepath.Join(ProfileDir(), ".last-logout.json")
}

// ReadProfileMgmtURL returns the ManagementURL string from a saved profile.
func ReadProfileMgmtURL(name string) (string, error) {
	if err := validateProfileName(name); err != nil {
		return "", err
	}
	raw, err := os.ReadFile(ProfilePath(name))
	if err != nil {
		return "", err
	}
	return extractMgmtURL(raw), nil
}

// parseMgmtURLValue normalizes a ManagementURL JSON value that may be either a string
// (legacy) or an object {Scheme, Host, Path} (NetBird 0.27+).
func parseMgmtURLValue(v json.RawMessage) string {
	var s string
	if err := json.Unmarshal(v, &s); err == nil {
		return s
	}
	var obj struct{ Scheme, Host, Path string }
	if err := json.Unmarshal(v, &obj); err == nil {
		return obj.Scheme + "://" + obj.Host + obj.Path
	}
	return ""
}

func extractMgmtURL(raw []byte) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	v, ok := m["ManagementURL"]
	if !ok {
		return ""
	}
	return parseMgmtURLValue(v)
}

// EnsureProfileDir creates the profile directory if it doesn't exist.
func EnsureProfileDir() error {
	return os.MkdirAll(ProfileDir(), 0700)
}

// ListProfiles returns all saved profiles by scanning *.json files.
func ListProfiles() ([]string, error) {
	entries, err := os.ReadDir(ProfileDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != ".last-logout.json" {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return names, nil
}

// validateProfileName rejects names containing path separators or traversal sequences.
func validateProfileName(name string) error {
	if name == "" || strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid profile name %q: must not contain '/', '\\', or '..'", name)
	}
	return nil
}

// ProfilePath returns the path for a named profile.
func ProfilePath(name string) string {
	return filepath.Join(ProfileDir(), name+".json")
}

// SaveProfile copies the active config to a named profile file (user-owned, 600).
func SaveProfile(name, activePath string) error {
	if err := validateProfileName(name); err != nil {
		return err
	}
	if err := EnsureProfileDir(); err != nil {
		return err
	}
	raw, err := ReadActiveRaw(activePath)
	if err != nil {
		return err
	}
	dst := ProfilePath(name)
	if err := os.WriteFile(dst, raw, 0600); err != nil {
		return fmt.Errorf("save profile %s: %w", name, err)
	}
	return nil
}

// RenameProfile renames a profile file.
func RenameProfile(old, newName string) error {
	if err := validateProfileName(old); err != nil {
		return err
	}
	if err := validateProfileName(newName); err != nil {
		return err
	}
	src := ProfilePath(old)
	dst := ProfilePath(newName)
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("profile %q already exists", newName)
	}
	return os.Rename(src, dst)
}

// RemoveProfile deletes a profile file.
func RemoveProfile(name string) error {
	if err := validateProfileName(name); err != nil {
		return err
	}
	return os.Remove(ProfilePath(name))
}

// Fingerprint computes sha256(mgmtURL|PrivateKey|SetupKey) for identity matching.
func Fingerprint(activePath string) (string, error) {
	raw, err := ReadActiveRaw(activePath)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256([]byte(fingerprintRaw(raw)))
	return fmt.Sprintf("%x", h), nil
}

// CurrentProfileName finds the saved profile name matching the active config fingerprint.
func CurrentProfileName(activePath string) (string, error) {
	activeHash, err := Fingerprint(activePath)
	if err != nil {
		return "", err
	}
	names, err := ListProfiles()
	if err != nil {
		return "", err
	}
	for _, name := range names {
		raw, err := os.ReadFile(ProfilePath(name))
		if err != nil {
			continue
		}
		h := sha256.Sum256([]byte(fingerprintRaw(raw)))
		if fmt.Sprintf("%x", h) == activeHash {
			return name, nil
		}
	}
	return "", nil
}

// fingerprintRaw extracts mgmt|pk|sk from raw JSON bytes for hashing.
func fingerprintRaw(raw []byte) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	mgmt, pk, sk := "", "", ""
	if v, ok := m["ManagementURL"]; ok {
		mgmt = parseMgmtURLValue(v)
	}
	if v, ok := m["PrivateKey"]; ok {
		_ = json.Unmarshal(v, &pk) //nolint:errcheck // best-effort: missing key → empty string
	}
	if v, ok := m["SetupKey"]; ok {
		_ = json.Unmarshal(v, &sk) //nolint:errcheck // best-effort: missing key → empty string
	}
	return mgmt + "|" + pk + "|" + sk
}

// CoreLogout: netbird down (ignore err) → save active to .last-logout.json → sudo rm active.
func CoreLogout(activePath string) error {
	_ = exec.Command("sudo", "netbird", "down").Run()
	raw, err := ReadActiveRaw(activePath)
	if err != nil {
		return err
	}
	if err := EnsureProfileDir(); err != nil {
		return err
	}
	if err := os.WriteFile(lastLogoutPath(), raw, 0600); err != nil {
		return fmt.Errorf("write logout sentinel: %w", err)
	}
	out, err := exec.Command("sudo", "rm", "-f", activePath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove active config: %w\n%s", err, out)
	}
	return nil
}

// RestoreLast restores the last logout config over activePath.
func RestoreLast(activePath string) error {
	src := lastLogoutPath()
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("no last-logout config found")
	}
	if err := WriteActive(src, activePath); err != nil {
		return err
	}
	return os.Remove(src)
}
