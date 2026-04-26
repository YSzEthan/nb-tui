package netbird

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

// PeersFromStatus extracts peers already parsed from Status JSON — preferred path.
// Falls back to parsePeersText if the slice is empty (older netbird versions).
func PeersFromStatus(s *Status) []Peer {
	if len(s.Peers) > 0 {
		return s.Peers
	}
	return parsePeersText()
}

// parsePeersText parses `netbird status -d` text output (fallback for older versions).
// Format expected:
//
//	Peers:
//	  <Name>:
//	    NetBird IP: <ip>
//	    Connection Status: Connected|Disconnected
func parsePeersText() []Peer {
	out, err := exec.Command("netbird", "status", "-d").Output()
	if err != nil {
		return []Peer{}
	}
	peers := make([]Peer, 0)
	curIdx := -1
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// Peer name line ends with ':'
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && strings.HasSuffix(trimmed, ":") {
			name := strings.TrimSuffix(trimmed, ":")
			name = strings.TrimSuffix(name, ".netbird.cloud")
			peers = append(peers, Peer{Name: name})
			curIdx = len(peers) - 1
			continue
		}
		if curIdx < 0 {
			continue
		}
		if after, ok := strings.CutPrefix(trimmed, "NetBird IP: "); ok {
			peers[curIdx].IP = after
		}
		if after, ok := strings.CutPrefix(trimmed, "Connection Status: "); ok {
			peers[curIdx].Connected = strings.EqualFold(after, "connected")
		}
	}
	return peers
}
