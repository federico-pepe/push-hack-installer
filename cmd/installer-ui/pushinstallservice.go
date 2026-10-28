package main

import (
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/applog"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/openurl"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/pushinstall"
)

// PushInstallService is bound to the frontend for the final screen: check
// whether push-hack is already on the device, then install or uninstall
// it. Always all three core hacks together — there's no hack-selection
// screen, by design (the user chose not to build one; push-hack's own
// install.sh treats these three as the non-optional framework core).
type PushInstallService struct{}

// Status reports how much of push-hack is on host: "installed", "none", or
// "needs-script" (the three hacks are there but not the PushHack Remote Script,
// as after an older installer; Install adds it).
func (s *PushInstallService) Status(host string) (string, string) {
	state, err := pushinstall.Status(host)
	if err != nil {
		applog.Printf("Status(%s) failed: %v", host, err)
		return pushinstall.StateNone, err.Error()
	}
	applog.Printf("Status(%s): %s", host, state)
	return state, ""
}

// Install deploys all three core hacks and the PushHack Remote Script to host. Blocking — see
// SSHKeyService.WaitForKeyAccepted for why that's fine with Wails' async
// call model. Installing (specifically, starting push-display's service)
// restarts Push3 and briefly interrupts Live; the UI warns about this
// before calling Install at all.
func (s *PushInstallService) Install(host string) ([]string, string) {
	applog.Printf("Install(%s): starting", host)
	summary, err := pushinstall.InstallAll(host)
	for _, line := range summary {
		applog.Printf("Install(%s): %s", host, line)
	}
	if err != nil {
		applog.Printf("Install(%s) failed: %v", host, err)
		return summary, err.Error()
	}
	applog.Printf("Install(%s): done", host)
	return summary, ""
}

// OpenPushManager opens Push Manager's web UI (port 7701, fixed — see
// push-hack's CLAUDE.md, this port is never reassigned) in the system
// browser.
func (s *PushInstallService) OpenPushManager(host string) string {
	applog.Printf("OpenPushManager(%s)", host)
	if err := openurl.Open("http://" + host + ":7701"); err != nil {
		applog.Printf("OpenPushManager(%s) failed: %v", host, err)
		return err.Error()
	}
	return ""
}

// OpenPushCatalog opens Push Hack Catalog's web UI (port 7702, fixed — same
// as above) in the system browser.
func (s *PushInstallService) OpenPushCatalog(host string) string {
	applog.Printf("OpenPushCatalog(%s)", host)
	if err := openurl.Open("http://" + host + ":7702"); err != nil {
		applog.Printf("OpenPushCatalog(%s) failed: %v", host, err)
		return err.Error()
	}
	return ""
}

// Uninstall removes all three core hacks and the PushHack Remote Script from host, leaving the top-level
// push-hack data directory and logs in place (matches uninstall.sh's
// default, non `--purge` behavior). Also restarts Push3 briefly, for the same
// push-display reason as Install.
func (s *PushInstallService) Uninstall(host string) ([]string, string) {
	applog.Printf("Uninstall(%s): starting", host)
	summary, err := pushinstall.UninstallAll(host)
	for _, line := range summary {
		applog.Printf("Uninstall(%s): %s", host, line)
	}
	if err != nil {
		applog.Printf("Uninstall(%s) failed: %v", host, err)
		return summary, err.Error()
	}
	applog.Printf("Uninstall(%s): done", host)
	return summary, ""
}
