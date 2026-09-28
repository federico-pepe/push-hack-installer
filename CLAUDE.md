# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> **Doc sync rule:** Keep this file, all `docs/` files, and `README.md` in sync with every code change. If a change affects behaviour, APIs, architecture, or known issues — update the relevant docs in the same commit. Save all implementation plans to `/plans/` with filename format `YYYY-MM-DD-title-of-the-plan.md`. Update [CHANGELOG.md](CHANGELOG.md)'s `## [Unreleased]` section in the same commit as any change worth noting to a future reader — new behavior, a fix, a changed API. Skip internal refactors and trivial edits.

> **Writing-style rule:** When writing or editing code comments, use the `caveman` skill. When writing or editing technical documentation (`docs/`, README files, hack-level READMEs, `CHANGELOG.md`), use the `simple-english` skill.

## Project

`push-hack-installer` — cross-platform (Windows/Mac/Linux) desktop app that
installs [push-hack](https://github.com/federico-pepe/ableton-push-hack)
onto an Ableton Push 3 over SSH, without the user ever opening a terminal.
Built with [Wails v3](https://v3alpha.wails.io/) (Go backend + webview
frontend), following the same stack and CI recipe already proven by the
sibling project `push-tethered-app` (`cmd/pushapp-ui`).

Ports the install/uninstall/discover logic from push-hack's
`scripts/install.sh`, `scripts/uninstall.sh`, `scripts/discover.sh`, and
`scripts/lib/common.sh` into Go, replacing terminal prompts with a guided
GUI flow. See [plans/2026-09-27-gui-installer.md](plans/2026-09-27-gui-installer.md)
for the full design.

**This repo does not implement push-hack itself** — it only automates the
deployment of push-hack's already-built hack binaries (push-manager,
push-catalog, push-display) onto a Push 3 device.

## Reference Docs

- `docs/architecture.md` — app structure, Wails frontend/backend split, how
  the install/uninstall/discover logic is ported from push-hack's bash
- `docs/build.md` — cross-platform build/CI notes (webkitgtk on Linux,
  Windows CGO override, packaging)

## Releases

Pre-1.0. Tag `vMAJOR.MINOR.PATCH[-alpha|-beta|-rc.N]`. Update
[CHANGELOG.md](CHANGELOG.md) in the same commit as the tagged code.
