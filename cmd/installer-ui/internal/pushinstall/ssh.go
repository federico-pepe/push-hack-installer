// Package pushinstall ports push-hack's scripts/install.sh and
// scripts/uninstall.sh (plus lib/common.sh's shared helpers) into Go, for
// the three core hacks only: Push Manager, Push Display, Push Hack
// Catalog. Unlike the bash originals, this never detects the init system —
// real Push 3 hardware only ever runs sysvinit (see push-hack's
// docs/push3-internals.md), so the systemd branch of install.sh is dead
// code on real hardware and isn't ported here.
package pushinstall

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshPort matches pushdiscover's reachability check and sshsetup's auth checks.
const sshPort = "22"

// dialTimeout bounds each individual SSH connection attempt.
const dialTimeout = 10 * time.Second

// client wraps an *ssh.Client for one remote user (ableton or root) — the
// two-session split mirrors lib/common.sh's push_exec vs push_exec_root:
// Push has no sudo, so root operations need a separate login.
type client struct {
	sshClient *ssh.Client
}

// dial opens an SSH connection to host as user, using the installer's own
// key (see internal/sshsetup). Host key checking is disabled — mirrors
// install.sh's StrictHostKeyChecking=no (lib/common.sh's ssh_opts) since
// Push doesn't publish a host key for users to pin.
func dial(host, user string, signer ssh.Signer) (*client, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         dialTimeout,
	}
	// "tcp4", not "tcp": mDNS resolution of push.local can return an IPv6
	// link-local address alongside the IPv4 one, and that link-local route
	// has been observed to silently time out on some networks/interfaces
	// while the IPv4 route works fine — forcing IPv4 avoids picking the
	// flaky one.
	c, err := ssh.Dial("tcp4", net.JoinHostPort(host, sshPort), config)
	if err != nil {
		return nil, fmt.Errorf("connect as %s: %w", user, err)
	}
	return &client{sshClient: c}, nil
}

func (c *client) Close() {
	_ = c.sshClient.Close()
}

// run executes cmd and returns combined stdout+stderr. Mirrors push_exec /
// push_exec_root — a single command per SSH session, like the bash
// originals (no persistent shell state between calls).
func (c *client) run(cmd string) (output string, err error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("open session: %w", err)
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	session.Stderr = &buf
	if err := session.Run(cmd); err != nil {
		return buf.String(), fmt.Errorf("%s: %w", cmd, err)
	}
	return buf.String(), nil
}

// runOK is run, but treats a non-zero exit as a non-error false result —
// for the many install.sh commands run as `cmd 2>/dev/null || true` /
// existence checks, where failure is an expected, silent outcome.
func (c *client) runOK(cmd string) bool {
	_, err := c.run(cmd)
	return err == nil
}

// runRetryNonEmpty runs cmd up to attempts times, retrying whenever it
// succeeds but returns blank output. Observed against a real device: Push's
// SSH server intermittently returns an empty result for a successful,
// otherwise-correct command (cause unconfirmed — a connection-rate throttle
// on the embedded sshd is the leading suspect) under back-to-back SSH
// traffic. For a command whose result decides whether anything else
// happens at all — see UninstallAll's use of this for its service
// discovery listing — treating one blank reply as "nothing to do" turns a
// transient hiccup into a silent, total no-op. A short delay between
// attempts gives it a chance to clear.
func (c *client) runRetryNonEmpty(cmd string, attempts int) (output string, err error) {
	for i := 0; i < attempts; i++ {
		output, err = c.run(cmd)
		if err == nil && strings.TrimSpace(output) != "" {
			return output, nil
		}
		if i < attempts-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return output, err
}

// copyBytes uploads in-memory content (embedded binaries, a hack.json
// rewritten with injected fields, a generated init.d script) to remotePath
// on the far side, using the classic SCP exec protocol (`scp -t <dir>`) —
// not SFTP. This exactly matches what install.sh's `scp` invocations do on
// the wire, which is known to work against Push's SSH server; the SFTP
// subsystem's availability there is unverified, so this avoids relying
// on it.
func (c *client) copyBytes(data []byte, remotePath string, mode os.FileMode) error {
	dir, base := splitRemotePath(remotePath)

	session, err := c.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("open stdin: %w", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open stdout: %w", err)
	}

	var stderr bytes.Buffer
	session.Stderr = &stderr

	// -t (sink mode, "to"): the remote end receives files into dir. -d
	// requires dir to already be a directory, which is exactly what every
	// caller here wants — a copy to a path whose parent doesn't exist yet
	// is a bug at the call site, not something to silently paper over.
	if err := session.Start(fmt.Sprintf("scp -qtd %s", shellQuote(dir))); err != nil {
		return fmt.Errorf("start scp: %w", err)
	}

	if err := writeSCPFile(stdin, stdout, base, data, mode); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("scp to %s: %w", remotePath, err)
	}
	_ = stdin.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("scp to %s: %w (%s)", remotePath, err, stderr.String())
	}
	return nil
}

// writeSCPFile speaks the minimal subset of the SCP sink protocol needed to
// send one file: a "C<mode> <size> <name>" header, the file bytes, then a
// single NUL byte — reading the remote's single-byte ack after both the
// header and the data, since the remote (Push's scp -t) uses a non-zero ack
// followed by a message line to report errors like "permission denied",
// which a fire-and-forget write would silently miss. See
// https://github.com/openssh/openssh-portable's scp.c for the protocol.
func writeSCPFile(w io.Writer, r io.Reader, name string, data []byte, mode os.FileMode) error {
	header := fmt.Sprintf("C%04o %d %s\n", mode.Perm(), len(data), name)
	if _, err := io.WriteString(w, header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if err := readSCPAck(r); err != nil {
		return fmt.Errorf("after header: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	if _, err := w.Write([]byte{0}); err != nil {
		return fmt.Errorf("write terminator: %w", err)
	}
	if err := readSCPAck(r); err != nil {
		return fmt.Errorf("after data: %w", err)
	}
	return nil
}

// readSCPAck reads one protocol ack byte: 0 is success, 1 or 2 is an error
// whose message follows as a newline-terminated line.
func readSCPAck(r io.Reader) error {
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("read ack: %w", err)
	}
	if buf[0] == 0 {
		return nil
	}
	msg, _ := bufio.NewReader(r).ReadString('\n')
	return fmt.Errorf("remote error: %s", msg)
}
