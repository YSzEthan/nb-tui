// Package netbird wraps the netbird CLI — runs status/up/down/ssh and helpers
// for managing sudo credentials. All calls shell out to the netbird binary.
package netbird

import (
	"encoding/json"
	"fmt"
)

// ManagementURL handles both string and {Scheme,Host,Path} shapes from NetBird 0.27+.
type ManagementURL struct {
	Value string
}

func (m *ManagementURL) UnmarshalJSON(data []byte) error {
	// Try string first (legacy)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		m.Value = s
		return nil
	}
	// Try object shape: {Scheme, Host, Path}
	var obj struct {
		Scheme string `json:"Scheme"`
		Host   string `json:"Host"`
		Path   string `json:"Path"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("managementURL: unexpected shape: %w", err)
	}
	m.Value = obj.Scheme + "://" + obj.Host + obj.Path
	return nil
}

func (m ManagementURL) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Value)
}

func (m ManagementURL) String() string { return m.Value }

// Status is the parsed output of `netbird status --json`.
type Status struct {
	DaemonStatus  string        `json:"daemonStatus"`
	ManagementURL ManagementURL `json:"managementState"`
	IP            string        `json:"netbirdIp"`
	PrivateKey    string        `json:"privateKey"`
	SetupKey      string        `json:"setupKey"`
	Peers         []Peer        `json:"peers"`
}

// Peer represents a single peer from the status JSON.
type Peer struct {
	Name      string `json:"fqdn"`
	IP        string `json:"netbirdIp"`
	Connected bool   `json:"connected"`
}

// IsLoggedIn returns true when the daemon is reachable and not in NeedsLogin/LoginFailed state.
func (s *Status) IsLoggedIn() bool {
	switch s.DaemonStatus {
	case "":
		// Daemon unreachable — fallback to PrivateKey presence
		return len(s.PrivateKey) > 0
	case "NeedsLogin", "LoginFailed":
		return false
	default:
		return true
	}
}

// IsConnected returns true when the daemon reports an active IP.
func (s *Status) IsConnected() bool {
	return s.IP != ""
}
