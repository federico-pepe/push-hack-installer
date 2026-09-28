# Architecture

This app does not exist yet. This file will describe the app structure once
the Wails v3 skeleton is built.

Planned shape, from [plans/2026-09-27-gui-installer.md](../plans/2026-09-27-gui-installer.md):

- `cmd/installer-ui/` — the Wails v3 app. Go backend, webview frontend.
- `internal/sshinstall/` — install and uninstall logic, ported from
  push-hack's `scripts/install.sh` and `scripts/lib/common.sh`.
- `internal/pushdiscover/` — device discovery, ported from push-hack's
  `scripts/discover.sh`.
- `vendor-bin/` — pinned copies of push-hack's pre-built hack binaries
  (push-manager, push-catalog, push_hook.so).

Update this file once each piece exists, with the real file layout.
