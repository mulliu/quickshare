package netutil

import (
	"fmt"
	"net"
)

// Listen tries preferred ports in order, falling back to random OS-assigned.
// Returns the listener (still open) and the actual port number.
func Listen(preferred ...int) (net.Listener, int, error) {
	ports := append(preferred, 0)
	for _, port := range ports {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		listener, err := net.ListenTCP("tcp", addr)
		if err != nil {
			continue
		}
		actual := listener.Addr().(*net.TCPAddr).Port
		return listener, actual, nil
	}
	return nil, 0, ErrNoPortAvailable
}
