# Build

## Requirements

- Go 1.26+ (matches `cmd/installer-ui/go.mod`'s `go` directive)
- Node 20+ (frontend build)
- `wails3` CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.9`

## Build and run locally

```bash
cd cmd/installer-ui
wails3 task build   # builds to cmd/installer-ui/bin/Push Hack Installer
./bin/Push\ Hack\ Installer
```

`wails3 task dev` runs it in dev mode with hot reload instead.

## Packaging (per OS)

```bash
cd cmd/installer-ui
wails3 task darwin:package:dmg     # macOS: .app bundle, ad-hoc signed, wrapped in a .dmg
wails3 task package                # Windows: NSIS installer .exe (needs makensis on PATH)
wails3 task linux:create:appimage  # Linux: self-contained .AppImage
```

No external runtime dependency (like push-tethered-app's libusb) needs
bundling here — this app is pure Go + a webview, nothing to link against.

## CI

`.github/workflows/build.yml`, adapted from the sibling project
`push-tethered-app`'s own `build.yml` (same Wails v3 stack, same
GTK4/WebKitGTK-on-Linux and wails3-CLI-caching approach), but simpler:
this app has no cgo dependency of its own (no libusb, no core/ sibling
checkout), so there's no MSYS2/mingw toolchain step on Windows and no
Docker cross-compile path to worry about — each OS just builds natively
with its own Taskfile defaults. No iOS/Android steps either (this repo
has no such targets).

Runs on `workflow_dispatch` (manual) and on `v*` tag pushes — not on every
PR/push to main, to avoid duplicating a full three-OS matrix run against
a tree GitHub Actions already tested pre-merge. A tag push also runs the
`release` job, which downloads the three artifacts the build matrix
produced and publishes them to a GitHub Release named after the tag.
