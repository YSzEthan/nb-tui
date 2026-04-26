// Package netbird wraps the netbird CLI — runs status/up/down/ssh and helpers
// for managing sudo credentials. All calls shell out to the netbird binary.
package netbird

import "encoding/json"

// management mirrors the "management" object in `netbird status --json`.
type management struct {
	URL string `json:"url"`
}

// peersList mirrors the "peers" object: {total, connected, details:[...]}.
type peersList struct {
	Details []Peer `json:"details"`
}

// sshServer mirrors the "sshServer" object in `netbird status --json`.
type sshServer struct {
	Enabled bool `json:"enabled"`
}

// Status is the parsed output of `netbird status --json`.
type Status struct {
	DaemonStatus string     `json:"daemonStatus"`
	Management   management `json:"management"`
	IP           string     `json:"netbirdIp"`
	PublicKey    string     `json:"publicKey"`
	PeersData    peersList  `json:"peers"`
	SSHServer    sshServer  `json:"sshServer"`
}

// ManagementURL returns the management server URL for display.
func (s *Status) ManagementURL() string { return s.Management.URL }

// Peers returns the list of peers from the parsed status JSON.
func (s *Status) Peers() []Peer { return s.PeersData.Details }

// SSHEnabled returns true when the local NetBird SSH server is running.
func (s *Status) SSHEnabled() bool { return s.SSHServer.Enabled }

// Peer represents a single peer from the status JSON.
// Connected is derived from the "status" string field ("Connected" → true).
type Peer struct {
	Name      string
	IP        string
	Connected bool
}

func (p *Peer) UnmarshalJSON(data []byte) error {
	var raw struct {
		FQDN   string `json:"fqdn"`
		IP     string `json:"netbirdIp"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Name = raw.FQDN
	p.IP = raw.IP
	p.Connected = raw.Status == "Connected"
	return nil
}

// IsLoggedIn returns true when the daemon is reachable and not in NeedsLogin/LoginFailed state.
func (s *Status) IsLoggedIn() bool {
	switch s.DaemonStatus {
	case "":
		return len(s.PublicKey) > 0
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
