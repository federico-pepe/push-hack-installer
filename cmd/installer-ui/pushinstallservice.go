package main

import (
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/openurl"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/pushinstall"
)

// PushInstallService is bound to the frontend for the final screen: check
// whether push-hack is already on the device, then install or uninstall
// it. Always all three core hacks together — there's no hack-selection
// screen, by design (the user chose not to build one; push-hack's own
// install.sh treats these three as the non-optional framework core).
type PushInstallService struct{}

// IsInstalled reports whether push-hack is already deployed to host.
func (s *PushInstallService) IsInstalled(host string) (bool, string) {
	installed, err := pushinstall.IsInstalled(host)
	if err != nil {
		return false, err.Error()
	}
	return installed, ""
}

// Install deploys all three core hacks to host. Blocking — see
// SSHKeyService.WaitForKeyAccepted for why that's fine with Wails' async
// call model. Installing (specifically, starting push-display's service)
// restarts Push3 and briefly interrupts Live; the UI warns about this
// before calling Install at all.
func (s *PushInstallService) Install(host string) ([]string, string) {
	summary, err := pushinstall.InstallAll(host)
	if err != nil {
		return summary, err.Error()
	}
	return summary, ""
}

// OpenPushManager opens Push Manager's web UI (port 7701, fixed — see
// push-hack's CLAUDE.md, this port is never reassigned) in the system
// browser.
func (s *PushInstallService) OpenPushManager(host string) string {
	if err := openurl.Open("http://" + host + ":7701"); err != nil {
		return err.Error()
	}
	return ""
}

// OpenPushCatalog opens Push Hack Catalog's web UI (port 7702, fixed — same
// as above) in the system browser.
func (s *PushInstallService) OpenPushCatalog(host string) string {
	if err := openurl.Open("http://" + host + ":7702"); err != nil {
		return err.Error()
	}
	return ""
}

// Uninstall removes all three core hacks from host, leaving the top-level
// push-hack data directory and logs in place (matches uninstall.sh's
// default, non `--purge` behavior). Also restarts Push3 briefly, for the same
// push-display reason as Install.
func (s *PushInstallService) Uninstall(host string) ([]string, string) {
	summary, err := pushinstall.UninstallAll(host)
	if err != nil {
		return summary, err.Error()
	}
	return summary, ""
}
