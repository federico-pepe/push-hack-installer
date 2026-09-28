package sshsetup

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshUser is push-hack's fixed on-device runtime user for normal (non-root)
// operations — see lib/common.sh's PUSH_USER default.
const sshUser = "ableton"

// sshPort matches pushdiscover's reachability check.
const sshPort = "22"

// pollInterval matches install.sh's own ConnectTimeout=10 op-by-op pacing
// closely enough while staying responsive to a UI that's polling, not
// blocking on a single slow attempt.
const pollInterval = 2 * time.Second

// dialTimeout is applied per attempt, not to the whole poll.
const dialTimeout = 5 * time.Second

// loadSigner reads and parses the local keypair. Returns a user-facing
// reason string on failure, same shape as the rest of this package's public
// functions, since callers forward it straight to the UI.
func loadSigner() (ssh.Signer, string) {
	privPath, err := keyPath()
	if err != nil {
		return nil, err.Error()
	}
	keyBytes, err := os.ReadFile(privPath)
	if err != nil {
		return nil, "No SSH key found."
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, "The saved SSH key is invalid."
	}
	return signer, ""
}

// tryAuth makes a single SSH auth attempt against host using signer. It
// never checks the host key (mirrors install.sh's `StrictHostKeyChecking=no`
// — see lib/common.sh's ssh_opts) since Push doesn't publish a host key for
// users to pin, and BatchMode is implicit: this never falls back to a
// password prompt.
func tryAuth(host string, signer ssh.Signer) bool {
	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         dialTimeout,
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(host, sshPort), config)
	if err != nil {
		return false
	}
	_ = client.Close()
	return true
}

// Signer loads the installer's keypair for packages that need to open their
// own SSH sessions (see internal/pushinstall) — a user-facing reason string
// on failure, same shape as the rest of this package's public functions.
func Signer() (ssh.Signer, string) {
	return loadSigner()
}

// AlreadyAuthorized reports whether a local key already exists and Push
// already accepts it — a single attempt, no retrying. Used right after the
// Connect screen to skip SSH key setup entirely when a previous run already
// completed it against this host.
func AlreadyAuthorized(host string) bool {
	signer, reason := loadSigner()
	if reason != "" {
		return false
	}
	return tryAuth(host, signer)
}

// WaitForKeyAccepted retries an SSH auth handshake against host using the
// local keypair (generated/loaded by EnsureKey) until it succeeds or
// timeout elapses.
//
// This is a long-running blocking call by design, not a background
// goroutine with an event callback — Wails calls are already async
// promises, so the frontend awaits this while showing a spinner, and the
// user has that whole window to open the browser and paste the key in.
func WaitForKeyAccepted(host string, timeout time.Duration) (accepted bool, reason string) {
	signer, reason := loadSigner()
	if reason != "" {
		return false, reason + " Restart this step."
	}

	deadline := time.Now().Add(timeout)
	for {
		if tryAuth(host, signer) {
			return true, ""
		}
		if time.Now().After(deadline) {
			return false, fmt.Sprintf("Push hasn't accepted the key yet. Make sure you pasted it at http://%s/ssh and saved.", host)
		}
		time.Sleep(pollInterval)
	}
}
