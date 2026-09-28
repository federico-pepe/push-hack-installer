// Package pushdiscover checks whether a Push 3 is reachable on the network,
// mirroring push-hack's scripts/discover.sh — a read-only probe, no SSH
// auth attempted here. SSH key setup and auth come later in the install
// flow, once a host has been confirmed reachable.
package pushdiscover

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// sshPort is the port push-hack's SSH server listens on (see
// scripts/install.sh's `ssh ... "${PUSH_USER}@${PUSH_HOST}"`, default
// port 22 — push-hack does not change it).
const sshPort = "22"

// CheckTimeout bounds how long a single reachability check may block the
// UI. Kept short since this runs on every keystroke-driven recheck from the
// Connect screen, not just once.
const CheckTimeout = 3 * time.Second

// CheckHost reports whether host is reachable on the SSH port. host may be
// a hostname (the default is "push.local", relying on mDNS) or a bare IP,
// matching push-hack's `--host` flag. It returns a short, user-facing
// reason on failure instead of a wrapped Go error — this result goes
// straight to the UI.
func CheckHost(host string) (reachable bool, reason string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return false, "Enter a hostname or IP address."
	}

	// "tcp4", not "tcp": see the matching comment in internal/pushinstall's
	// dial() — mDNS can hand back an IPv6 link-local address that silently
	// times out while IPv4 works fine.
	conn, err := net.DialTimeout("tcp4", net.JoinHostPort(host, sshPort), CheckTimeout)
	if err != nil {
		return false, fmt.Sprintf("Can't reach %s. Check that Push is turned on and on the same network.", host)
	}
	_ = conn.Close()
	return true, ""
}
