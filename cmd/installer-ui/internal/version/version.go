// Package version holds the build-time version string of the installer.
//
// Version is overridden at build time via:
//
//	go build -ldflags "-X github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/version.Version=v0.1.4-beta"
//
// Local `go build`/`go run` without that flag reports "dev".
package version

var Version = "dev"
