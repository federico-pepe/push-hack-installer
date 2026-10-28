package pushinstall

import (
	"fmt"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/sshsetup"
)

// Install states, as Status returns them.
const (
	StateNone        = "none"         // not installed (or only part of it)
	StateInstalled   = "installed"    // the three services and the PushHack Remote Script
	StateNeedsScript = "needs-script" // the three services, but no PushHack Remote Script yet
)

// Status reports how much of push-hack is on host. This installer always
// deploys or removes the three core hacks together (no hack-selection screen),
// so a simple all-or-nothing check is enough for them. A Push set up by an
// older installer has the services but not the PushHack Remote Script: that
// is StateNeedsScript, and running Install again adds it. A partial install
// (e.g. interrupted mid-run) shows as StateNone, and re-running Install is
// safe: every step is idempotent.
func Status(host string) (string, error) {
	signer, reason := sshsetup.Signer()
	if reason != "" {
		return StateNone, fmt.Errorf("%s", reason)
	}

	rootClient, err := dial(host, root, signer)
	if err != nil {
		return StateNone, fmt.Errorf("connect to Push as root: %w", err)
	}
	defer rootClient.Close()

	for _, h := range coreHacks() {
		if !rootClient.runOK("test -f /etc/init.d/" + serviceName(h.id)) {
			return StateNone, nil
		}
	}
	if !remoteScriptInstalled(rootClient, detectUserDataDir(rootClient)) {
		return StateNeedsScript, nil
	}
	return StateInstalled, nil
}
