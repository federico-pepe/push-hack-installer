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

// WaitForKeyAccepted retries an SSH auth handshake against host using the
// local keypair (generated/loaded by EnsureKey) until it succeeds or
// timeout elapses. It never checks the host key (mirrors install.sh's
// `StrictHostKeyChecking=no` — see lib/common.sh's ssh_opts) since Push
// doesn't publish a host key for users to pin, and BatchMode is implicit:
// this never falls back to a password prompt.
//
// This is a long-running blocking call by design, not a background
// goroutine with an event callback — Wails calls are already async
// promises, so the frontend awaits this while showing a spinner, and the
// user has that whole window to open the browser and paste the key in.
func WaitForKeyAccepted(host string, timeout time.Duration) (accepted bool, reason string) {
	privPath, err := keyPath()
	if err != nil {
		return false, err.Error()
	}
	keyBytes, err := os.ReadFile(privPath)
	if err != nil {
		return false, "No SSH key found. Restart this step."
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return false, "The saved SSH key is invalid. Restart this step."
	}

	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         dialTimeout,
	}
	addr := net.JoinHostPort(host, sshPort)

	deadline := time.Now().Add(timeout)
	for {
		client, err := ssh.Dial("tcp", addr, config)
		if err == nil {
			_ = client.Close()
			return true, ""
		}
		if time.Now().After(deadline) {
			return false, fmt.Sprintf("Push hasn't accepted the key yet. Make sure you pasted it at http://%s/ssh and saved.", host)
		}
		time.Sleep(pollInterval)
	}
}
