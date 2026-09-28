package pushinstall

import (
	"fmt"
	"strings"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/sshsetup"
)

// ableton is push-hack's fixed on-device runtime user — see
// lib/common.sh's PUSH_USER default (this installer never overrides it,
// unlike install.sh's --user flag, since there's no scripted/CI use case
// here).
const ableton = "ableton"
const root = "root"

// InstallAll deploys all three core hacks (push-manager, push-catalog,
// push-display) to host, porting install.sh's deploy_hack +
// install_hack_service for exactly this fixed hack set — there's no
// hack-selection screen, so no per-hack enable/disable to honor.
//
// Returns a short per-hack status line for each hack (for a simple summary
// in the UI) and an error if anything fatal happened. Non-fatal steps
// (stopping a service that wasn't running, best-effort boot-enable) mirror
// install.sh's own `|| true`-guarded commands and never fail the install.
func InstallAll(host string) (summary []string, err error) {
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

	// Create the top-level structure. On a freshly factory-reset device the
	// data dir may not yet be writable by ableton — fall back to creating it
	// as root, then hand ownership back, mirroring install.sh's main().
	if !abletonClient.runOK(fmt.Sprintf("mkdir -p %s %s", shellQuote(pushHackDir+"/hacks"), shellQuote(pushHackDir+"/logs"))) {
		if _, err := rootClient.run(fmt.Sprintf("mkdir -p %s %s", shellQuote(pushHackDir+"/hacks"), shellQuote(pushHackDir+"/logs"))); err != nil {
			return nil, fmt.Errorf("create %s on Push: %w", pushHackDir, err)
		}
		if _, err := rootClient.run(fmt.Sprintf("chown -R ableton:users %s", shellQuote(pushHackDir))); err != nil {
			return nil, fmt.Errorf("fix ownership of %s: %w", pushHackDir, err)
		}
	}

	for _, h := range coreHacks() {
		line, err := installOne(abletonClient, rootClient, h, userDataDir, pushHackDir)
		if err != nil {
			return summary, fmt.Errorf("%s: %w", h.id, err)
		}
		summary = append(summary, line)
	}

	return summary, nil
}

func installOne(abletonClient, rootClient *client, h hack, userDataDir, pushHackDir string) (string, error) {
	remoteHackDir := pushHackDir + "/hacks/" + h.id
	svcName := serviceName(h.id)

	if _, err := abletonClient.run("mkdir -p " + shellQuote(remoteHackDir)); err != nil {
		return "", fmt.Errorf("create hack directory: %w", err)
	}
	// Normalise ownership — a prior root-owned copy (e.g. push-display's
	// .so) can leave this root:root, which would then fail the non-root
	// hack.json copy below. See install.sh's deploy_hack comment on this.
	if _, err := rootClient.run(fmt.Sprintf("chown ableton:users %s", shellQuote(remoteHackDir))); err != nil {
		return "", fmt.Errorf("set ownership: %w", err)
	}

	// Stop any existing service first so the running binary isn't locked
	// during copy — best-effort, matches install.sh's `2>/dev/null || true`.
	rootClient.runOK("/etc/init.d/" + svcName + " stop")

	if h.binaryName != "" {
		remoteBinaryPath := remoteHackDir + "/" + h.binaryName
		// .so files are deployed as root, matching install.sh (they're
		// owned by root on the target); everything else as ableton.
		var copyErr error
		if strings.HasSuffix(h.binaryName, ".so") {
			copyErr = rootClient.copyBytes(h.binary, remoteBinaryPath, 0755)
		} else {
			copyErr = abletonClient.copyBytes(h.binary, remoteBinaryPath, 0755)
		}
		if copyErr != nil {
			return "", fmt.Errorf("copy %s: %w", h.binaryName, copyErr)
		}
		if _, err := rootClient.run("chmod +x " + shellQuote(remoteBinaryPath)); err != nil {
			return "", fmt.Errorf("make %s executable: %w", h.binaryName, err)
		}
	}

	rewritten, err := rewriteHackJSON(h.hackJSON, userDataDir, pushHackDir, remoteHackDir)
	if err != nil {
		return "", fmt.Errorf("prepare hack.json: %w", err)
	}
	if err := abletonClient.copyBytes(rewritten, remoteHackDir+"/hack.json", 0644); err != nil {
		return "", fmt.Errorf("copy hack.json: %w", err)
	}

	if err := installService(rootClient, h, remoteHackDir, pushHackDir); err != nil {
		return "", err
	}

	return h.id + ": installed", nil
}

// installService writes the init.d script (custom or generic — see
// hacks.go), enables it at boot, and starts it. Starting push-display's
// service is what actually patches /etc/init.d/push3 and restarts Push3 —
// the UI warns about this before calling InstallAll at all.
func installService(rootClient *client, h hack, remoteHackDir, pushHackDir string) error {
	svcName := serviceName(h.id)
	logDir := pushHackDir + "/logs"

	var initdContent []byte
	if len(h.customInitd) > 0 {
		initdContent = renderCustomInitd(h.customInitd, h.id, remoteHackDir, logDir)
	} else {
		initdContent = []byte(genericInitd(h.id, remoteHackDir+"/"+h.binaryName, remoteHackDir+"/hack.json", logDir+"/"+h.id+".log"))
	}

	initdPath := "/etc/init.d/" + svcName
	if err := rootClient.copyBytes(initdContent, initdPath, 0755); err != nil {
		return fmt.Errorf("install service script: %w", err)
	}
	if _, err := rootClient.run("chmod +x " + shellQuote(initdPath)); err != nil {
		return fmt.Errorf("make service script executable: %w", err)
	}

	// Enable at boot — try update-rc.d, fall back to manual rc symlinks.
	// Best-effort: a device without either mechanism still gets the hack
	// running now, just not surviving a reboot, same as install.sh.
	rootClient.runOK(fmt.Sprintf(`
		if command -v update-rc.d >/dev/null 2>&1; then
			update-rc.d %[1]s defaults 2>/dev/null || true
		else
			for n in 2 3 4 5; do
				ln -sf /etc/init.d/%[1]s /etc/rc${n}.d/S99%[1]s 2>/dev/null || true
			done
		fi
	`, svcName))

	rootClient.runOK(initdPath + " stop")
	if _, err := rootClient.run(initdPath + " start"); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	return nil
}
