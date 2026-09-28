package pushinstall

import (
	"path"
	"strings"
)

// splitRemotePath splits a remote path into its directory and base name
// using forward-slash (POSIX) rules regardless of the host OS this app
// runs on — Push always uses Linux paths, unlike Go's path/filepath which
// would use backslashes when this installer runs on Windows.
func splitRemotePath(remotePath string) (dir, base string) {
	return path.Dir(remotePath), path.Base(remotePath)
}

// shellQuote wraps s in single quotes for safe use in a remote shell
// command, escaping any single quote it contains. Every remote path this
// package builds is one of a small, fixed set of installer-controlled
// values (the pinned remote dir, hack IDs, binary names) — never
// user-typed input — so this only needs to be correct, not hardened
// against adversarial input.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
