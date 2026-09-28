# push-hack-installer

A desktop app that installs [push-hack](https://github.com/federico-pepe/ableton-push-hack) on an Ableton Push 3.

No terminal needed. Download the app, run it, follow the on-screen steps.

## Status

This project is in early planning. See [plans/2026-09-27-gui-installer.md](plans/2026-09-27-gui-installer.md) for the design.

## What it does

The app connects to your Push 3 over SSH and installs the core push-hack
hacks: Push Manager, Push Display, and Push Hack Catalog. It does the same
job as push-hack's `scripts/install.sh`, but with a guided window instead of
a command line.

## Platforms

Windows, macOS, and Linux. Built with [Wails v3](https://v3alpha.wails.io/).

## Building from source

Build instructions will be added once the app skeleton exists. See
[docs/build.md](docs/build.md).

## License

TBD.
