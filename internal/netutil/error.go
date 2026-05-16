package netutil

import "errors"

var (
	ErrNoLANIP         = errors.New("no suitable LAN IP found")
	ErrNoPortAvailable = errors.New("no port available")
)
