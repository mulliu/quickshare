package netutil

import (
	"net"
	"sort"
)

// FindLANIP detects the most suitable LAN IP address for sharing.
// Prefers WiFi interfaces, then Ethernet. Skips loopback, Docker, and virtual adapters.
func FindLANIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	type candidate struct {
		ip  string
		pri int // lower = preferred
	}
	var candidates []candidate

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// Skip common virtual/Docker interfaces
		if isVirtual(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			if isPrivate(ip) {
				pri := ifacePriority(iface.Name)
				candidates = append(candidates, candidate{ip.String(), pri})
			}
		}
	}

	if len(candidates) == 0 {
		return "", ErrNoLANIP
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].pri < candidates[j].pri
	})
	return candidates[0].ip, nil
}

// isPrivate checks if an IP is in RFC 1918 private ranges.
func isPrivate(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
	}
	return false
}

// isVirtual detects virtual/Docker interface names.
func isVirtual(name string) bool {
	virtualPrefixes := []string{"docker", "veth", "br-", "vmnet", "vnic", "vbox", "tailscale", "zerotier"}
	for _, p := range virtualPrefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}
	// Windows virtual adapters (Hyper-V/WSL, Wi-Fi Direct/hosted-network/ICS hotspot).
	// e.g. "vEthernet (WSL (Hyper-V firewall))", "本地连接* 10", "Local Area Connection* 3".
	virtualSubstrings := []string{"vethernet", "hyper-v", "wi-fi direct", "本地连接*", "local area connection*"}
	for _, sub := range virtualSubstrings {
		if containsInsensitive(name, sub) {
			return true
		}
	}
	return false
}

// ifacePriority returns a priority score for interface types (lower = better).
func ifacePriority(name string) int {
	// WiFi preferred over Ethernet for phone connectivity
	lower := name
	if len(lower) > 3 {
		lower = name[:3]
	}
	switch {
	case name == "en0" || name == "en1":
		return 20 // macOS Ethernet
	case name == "eth0" || name == "eth1":
		return 20
	case name == "wlan0" || name == "wlp0" || name == "wlp1" || name == "Wi-Fi":
		return 10 // WiFi
	case len(name) >= 2 && name[:2] == "wl":
		return 10 // Linux WiFi (wlan0, wlp2s0, etc.)
	case len(name) >= 2 && (name[:2] == "en" || name[:2] == "et"):
		return 20 // Linux Ethernet
	}
	// Check for common patterns in longer names (incl. Chinese Windows adapter names)
	if containsInsensitive(name, "wi-fi") || containsInsensitive(name, "wireless") || containsInsensitive(name, "wlan") || contains(name, "无线") {
		return 10
	}
	if containsInsensitive(name, "eth") || containsInsensitive(name, "ethernet") || contains(name, "以太网") {
		return 20
	}
	return 30
}

func containsInsensitive(s, substr string) bool {
	sLower := toLower(s)
	subLower := toLower(substr)
	return len(sLower) >= len(subLower) && contains(sLower, subLower)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
