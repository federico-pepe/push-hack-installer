package pushinstall

import (
	"fmt"
	"strings"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/sshsetup"
)

// UninstallAll removes every push-hack service from host — not just the
// three core hacks this installer deploys, but also any community hack
// installed later via Push Hack Catalog. Matches uninstall.sh's default
// (non---purge) behavior exactly: it never hardcodes a hack list either,
// it discovers every `push-hack-*` init.d script on the device and removes
// each one, leaving the top-level push-hack directory and its logs in
// place. Stopping push-display's service here is what removes the
// LD_PRELOAD patch from /etc/init.d/push3 and restarts Push3 — the UI
// warns about this before calling this at all.
func UninstallAll(host string) (summary []string, err error) {
	signer, reason := sshsetup.Signer()
	if reason != "" {
		return nil, fmt.Errorf("%s", reason)
	}

	abletonClient, err := dial(host, ableton, signer)
	if err != nil {
		return nil, fmt.Errorf("connect to Push: %w", err)
	}
	defer abletonClient.Close()

	rootClient, err := dial(host, root, signer)
	if err != nil {
		return nil, fmt.Errorf("connect to Push as root: %w", err)
	}
	defer rootClient.Close()

	userDataDir := detectUserDataDir(abletonClient)
	pushHackDir := userDataDir + "/push-hack"

	// Discover every installed push-hack service — this installer's own
	// three core hacks and anything installed later via the Catalog, same
	// as uninstall.sh's `ls /etc/init.d/ | grep '^push-hack-'` scan. Retried
	// on a blank result — see runRetryNonEmpty's doc comment — since this
	// single command decides whether anything below happens at all; a
	// silently-empty listing here would otherwise make Uninstall a
	// no-op that still reports success.
	listing, _ := rootClient.runRetryNonEmpty("ls /etc/init.d/ 2>/dev/null | grep '^push-hack-' || true", 4)
	for _, svcName := range strings.Fields(listing) {
		hackID := strings.TrimPrefix(svcName, "push-hack-")
		removeOneHack(abletonClient, rootClient, svcName, hackID, pushHackDir)
		summary = append(summary, hackID+": removed")
	}

	// Legacy backup left behind by push-display's old standalone deploy.sh
	// path (not something this installer ever creates itself, but a device
	// tested with that script beforehand could still have it) — cleaned up
	// unconditionally, matching uninstall.sh.
	rootClient.runOK("rm -f /etc/init.d/push3.push-hack-bak")

	if len(summary) == 0 {
		summary = append(summary, "No push-hack services found on Push.")
	}
	return summary, nil
}

// removeOneHack stops and removes one hack's service and hack directory.
// Every step is best-effort (mirrors uninstall.sh's `|| true`-guarded
// commands) since a partially-installed or already-half-removed hack
// shouldn't block removing everything else.
func removeOneHack(abletonClient, rootClient *client, svcName, hackID, pushHackDir string) {
	initdPath := "/etc/init.d/" + svcName

	// Stop calls the service's own stop action — for push-display this is
	// what strips the LD_PRELOAD line and restarts Push3 cleanly.
	rootClient.runOK(initdPath + " stop")

	rootClient.runOK(fmt.Sprintf(`
		if command -v update-rc.d >/dev/null 2>&1; then
			update-rc.d -f %[1]s remove 2>/dev/null || true
		else
			rm -f /etc/rc*.d/*%[1]s 2>/dev/null || true
		fi
	`, svcName))
	rootClient.runOK("rm -f " + shellQuote(initdPath))

	remoteHackDir := pushHackDir + "/hacks/" + hackID
	abletonClient.runOK("rm -rf " + shellQuote(remoteHackDir))
}
