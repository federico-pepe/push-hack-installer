# Build

## Requirements

- Go 1.26+ (matches `cmd/installer-ui/go.mod`'s `go` directive)
- Node 20+ (frontend build)
- `wails3` CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.9`

## Build and run locally

```bash
cd cmd/installer-ui
wails3 task build   # builds to cmd/installer-ui/bin/installer-ui
./bin/installer-ui
```

`wails3 task dev` runs it in dev mode with hot reload instead.

## CI (not set up yet)

Planned approach: reuse the CI recipe from the sibling project
`push-tethered-app` (`cmd/pushapp-ui`, `.github/workflows/build.yml`), which
already solved, for the same Wails v3 stack:

- the correct webkitgtk package version to install on Linux
- the `CGO_ENABLED=0` override needed on Windows builds
- AppImage packaging on Linux
- caching the `wails3` CLI across CI runs

This repo has no iOS/Android targets, so the Android/iOS steps in that
workflow do not apply here — only the Linux/macOS/Windows desktop jobs.

Update this file with the real workflow file once it is copied over and
adjusted.
