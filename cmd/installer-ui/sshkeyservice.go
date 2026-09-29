package main

import (
	"time"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/applog"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/openurl"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/sshsetup"
)

// keyAcceptTimeout bounds how long WaitForKeyAccepted blocks: long enough
// for a human to switch to the browser, paste a key, and click save, short
// enough that the frontend's spinner doesn't spin forever if they give up.
const keyAcceptTimeout = 5 * time.Minute

// SSHKeyService is bound to the frontend for the SSH key setup screen.
type SSHKeyService struct{}

// AlreadyAuthorized reports whether a key from a previous run already
// exists and Push already accepts it for host — lets the Connect screen
// skip SSH key setup entirely when it's not needed.
func (s *SSHKeyService) AlreadyAuthorized(host string) bool {
	authorized := sshsetup.AlreadyAuthorized(host)
	applog.Printf("AlreadyAuthorized(%s): %v", host, authorized)
	return authorized
}

// EnsureKey generates a keypair if one doesn't exist yet, and returns the
// public key line (to display/copy) and its fingerprint.
func (s *SSHKeyService) EnsureKey() (pubKeyLine string, fingerprint string, err string) {
	line, fp, e := sshsetup.EnsureKey()
	if e != nil {
		applog.Printf("EnsureKey failed: %v", e)
		return "", "", e.Error()
	}
	applog.Printf("EnsureKey: using key fingerprint %s", fp)
	return line, fp, ""
}

// OpenSSHPage opens Push's "add an SSH key" web page in the system browser.
func (s *SSHKeyService) OpenSSHPage(host string) string {
	applog.Printf("OpenSSHPage(%s)", host)
	if err := openurl.Open("http://" + host + "/ssh"); err != nil {
		applog.Printf("OpenSSHPage(%s) failed: %v", host, err)
		return err.Error()
	}
	return ""
}

// WaitForKeyAccepted blocks (see keyAcceptTimeout) until Push accepts the
// local key over SSH, or the timeout elapses.
func (s *SSHKeyService) WaitForKeyAccepted(host string) (bool, string) {
	applog.Printf("WaitForKeyAccepted(%s): waiting up to %s", host, keyAcceptTimeout)
	accepted, reason := sshsetup.WaitForKeyAccepted(host, keyAcceptTimeout)
	applog.Printf("WaitForKeyAccepted(%s): accepted=%v reason=%q", host, accepted, reason)
	return accepted, reason
}
