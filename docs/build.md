# Build

This app does not exist yet. This file will hold build and CI notes once the
Wails v3 skeleton is built.

Planned approach: reuse the CI recipe from the sibling project
`push-tethered-app` (`cmd/pushapp-ui`, `.github/workflows/build.yml`), which
already solved:

- the correct webkitgtk package version to install on Linux
- the `CGO_ENABLED=0` override needed on Windows builds
- AppImage packaging on Linux
- caching the `wails3` CLI across CI runs

Update this file with the real commands once the workflow is copied over and
adjusted for this repo.
