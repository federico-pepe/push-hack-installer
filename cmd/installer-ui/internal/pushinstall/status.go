package pushinstall

import (
	"fmt"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/sshsetup"
)

// IsInstalled reports whether all three core hacks' service scripts are
// present on host. This installer always deploys or removes all three
// together (no hack-selection screen), so a simple all-or-nothing check is
// enough to decide whether the final screen offers "Install" or
// "Uninstall" — a partial install (e.g. interrupted mid-run) shows as not
// installed, and re-running Install is safe: every step is idempotent.
func IsInstalled(host string) (bool, error) {
	signer, reason := sshsetup.Signer()
	if reason != "" {
		return false, fmt.Errorf("%s", reason)
	}

	rootClient, err := dial(host, root, signer)
	if err != nil {
		return false, fmt.Errorf("connect to Push as root: %w", err)
	}
	defer rootClient.Close()

	for _, h := range coreHacks() {
		if !rootClient.runOK("test -f /etc/init.d/" + serviceName(h.id)) {
			return false, nil
		}
	}
	return true, nil
}
