package main

import (
	"time"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/openurl"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/sshsetup"
)

// keyAcceptTimeout bounds how long WaitForKeyAccepted blocks: long enough
// for a human to switch to the browser, paste a key, and click save, short
// enough that the frontend's spinner doesn't spin forever if they give up.
const keyAcceptTimeout = 5 * time.Minute

// SSHKeyService is bound to the frontend for the SSH key setup screen.
type SSHKeyService struct{}

// EnsureKey generates a keypair if one doesn't exist yet, and returns the
// public key line (to display/copy) and its fingerprint.
func (s *SSHKeyService) EnsureKey() (pubKeyLine string, fingerprint string, err string) {
	line, fp, e := sshsetup.EnsureKey()
	if e != nil {
		return "", "", e.Error()
	}
	return line, fp, ""
}

// OpenSSHPage opens Push's "add an SSH key" web page in the system browser.
func (s *SSHKeyService) OpenSSHPage(host string) string {
	if err := openurl.Open("http://" + host + "/ssh"); err != nil {
		return err.Error()
	}
	return ""
}

// WaitForKeyAccepted blocks (see keyAcceptTimeout) until Push accepts the
// local key over SSH, or the timeout elapses.
func (s *SSHKeyService) WaitForKeyAccepted(host string) (bool, string) {
	return sshsetup.WaitForKeyAccepted(host, keyAcceptTimeout)
}
