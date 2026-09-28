// Package openurl opens a URL in the system's default browser. Wails v3
// does not expose this itself, so this shells out per OS — the same three
// commands any cross-platform Go CLI uses for the job.
package openurl

import (
	"os/exec"
	"runtime"
)

// Open launches url in the default browser.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		// "" is the (required) empty window title argument to `start`.
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default: // linux and other unix-likes
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
