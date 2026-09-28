# push-hack-installer

A desktop app that installs [push-hack](https://github.com/federico-pepe/ableton-push-hack) on an Ableton Push 3.

No terminal needed. Download the app, run it, follow the on-screen steps.

## Status

Early skeleton: the app opens to a Welcome screen. The SSH connect/install
flow is not built yet. See [plans/2026-09-27-gui-installer.md](plans/2026-09-27-gui-installer.md) for the design.

## What it does

The app connects to your Push 3 over SSH and installs the core push-hack
hacks: Push Manager, Push Display, and Push Hack Catalog. It does the same
job as push-hack's `scripts/install.sh`, but with a guided window instead of
a command line.

## Platforms

Windows, macOS, and Linux. Built with [Wails v3](https://v3alpha.wails.io/).

## Building from source

```bash
cd cmd/installer-ui
wails3 task build
./bin/installer-ui
```

See [docs/build.md](docs/build.md) for requirements and CI plans.

## License

TBD.
