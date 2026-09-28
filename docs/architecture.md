# Architecture

## `cmd/installer-ui/`

The Wails v3 app: a Go backend and a plain TypeScript webview frontend
(no framework — same "vanilla" template as push-tethered-app's
`cmd/pushapp-ui`). Desktop only (macOS, Windows, Linux) — no iOS/Android
targets.

- `main.go` — creates the window and binds `ConnectService`, `SSHKeyService`,
  and `PushInstallService` to the frontend.
- `connectservice.go` — thin Wails-bound wrapper over `internal/pushdiscover`.
- `internal/pushdiscover/reachability.go` — `CheckHost(host)`: a TCP dial to
  the host's SSH port (22) with a short timeout. This is a reachability
  probe only, ported in spirit from push-hack's `scripts/discover.sh` — it
  does **not** attempt SSH auth.
- `sshkeyservice.go` — thin Wails-bound wrapper over `internal/sshsetup` and
  `internal/openurl`.
- `internal/sshsetup/key.go` — `EnsureKey()`: generates an RSA-3072 keypair
  at `~/.ssh/push_hack_id_rsa` if one doesn't exist yet, or loads the
  existing one. A dedicated filename, not `~/.ssh/id_rsa` (what
  push-hack's `scripts/install.sh` `ssh_wizard` uses) — this app must never
  silently overwrite or shadow a key the user already has for something
  else. Returns the authorized_keys-format public key line and its SHA256
  fingerprint.
- `internal/sshsetup/poll.go` — `tryAuth`/`loadSigner` are the shared
  single-attempt building blocks:
  - `AlreadyAuthorized(host)`: one attempt, no retrying. Used right after
    the Connect screen to skip SSH key setup entirely when a previous run
    already got this exact key (`push_hack_id_rsa`) accepted for this host.
    Note it's specific to that one key file — a host that already trusts a
    *different* key (e.g. the user's own, or an older `~/.ssh/id_rsa` from
    testing before this app pinned its own filename) still reports
    unauthorized here, correctly, since push-hack's own key genuinely isn't
    on that host yet.
  - `WaitForKeyAccepted(host, timeout)`: retries the same handshake (as user
    `ableton`, host key checking disabled — mirrors `lib/common.sh`'s
    `ssh_opts`, since Push has no host key for users to pin) every 2s until
    it succeeds or the timeout (5 minutes) elapses. Long blocking call by
    design: Wails calls are async promises, so the frontend awaits it while
    showing a spinner instead of needing a separate event/callback
    mechanism.
- `internal/openurl/openurl.go` — opens a URL in the system's default
  browser by shelling out per OS (`open`/`start`/`xdg-open`); Wails v3 has no
  built-in equivalent. Used for Push's `/ssh` page and (from
  `pushinstallservice.go`) Push Manager/Push Hack Catalog's own web UIs.
- `pushinstallservice.go` — thin Wails-bound wrapper over
  `internal/pushinstall`.
- `internal/pushinstall/` — ports `scripts/install.sh` and
  `scripts/uninstall.sh` (plus `lib/common.sh`'s shared helpers) into Go,
  for exactly the three core hacks — there is deliberately no
  hack-selection screen. One deliberate scope cut from the bash originals:
  this never detects the init system, because real Push 3 hardware only
  ever runs sysvinit (see push-hack's `docs/push3-internals.md`) — the
  systemd branch of `install.sh` is dead code on real hardware and isn't
  ported.
  - `vendor/` — the three hacks' pre-built binaries, `hack.json`, and
    push-display's `service.initd` template, embedded via `go:embed`
    (`hacks.go`). Copied from an `ableton-push-hack` checkout at the time
    this was written — this is the **offline fallback**, not the primary
    source: `fetch.go`'s `fetchLatest` tries downloading each of those same
    three files fresh from `ableton-push-hack`'s `main` branch
    (`raw.githubusercontent.com`) before every install, all-or-nothing per
    hack (a fetched binary paired with a stale embedded `hack.json`, or
    vice versa, risks a worse mismatch than just picking one consistent
    source). `installOne` only falls back to the embedded copy if the
    fetch fails outright — no network, GitHub down, a non-200. The vendored
    snapshot still goes stale relative to push-hack's own commits, same as
    before, but that only matters when a user installs with no internet
    access at all.
  - `ssh.go` — a `client` per SSH user (`ableton` or `root` — Push has no
    `sudo`, so root operations need a separate login, mirroring
    `lib/common.sh`'s `push_exec` vs `push_exec_root`). File upload speaks
    the classic SCP exec protocol (`scp -qtd <dir>`) by hand, including
    reading the remote's ack bytes to surface errors — not SFTP, since
    that subsystem's availability on Push's SSH server is unverified,
    while `scp` is known to work (push-hack's own scripts use it). Dials
    `tcp4` specifically, not `tcp` — mDNS can hand back an IPv6 link-local
    address for `push.local` that silently times out on some
    networks/interfaces while IPv4 works fine; observed causing real,
    hard-to-diagnose connection failures during testing.
    `runRetryNonEmpty` retries a command up to 4 times if it succeeds but
    returns blank output — Push's SSH server has been observed to
    intermittently do this under back-to-back traffic (root cause
    unconfirmed; a connection-rate throttle on the embedded sshd is the
    leading suspect). `UninstallAll`'s service-discovery listing uses this,
    since a blank reply there used to make Uninstall silently do nothing
    while still reporting success.
  - `hackjson.go` — `rewriteHackJSON`: resolves the `${USER_DATA}`
    placeholder and injects `_push_hack_dir`/`_hack_dir`, as text
    substitution before parsing (matching `install.sh`'s `sed`-based
    approach) so it behaves identically regardless of where the
    placeholder appears.
  - `initd.go` — `genericInitd` is a byte-for-byte port of
    `lib/common.sh`'s `generate_initd_script`, used for push-manager and
    push-catalog (neither ships its own `service.initd`).
    `renderCustomInitd` substitutes `{{...}}` placeholders into
    push-display's own template, which patches `/etc/init.d/push3`'s
    `LD_PRELOAD` line and restarts Push3 on start/stop — this is the one
    hack whose "service" does more than run a plain binary.
  - `install.go` / `uninstall.go` / `status.go` — `InstallAll`,
    `UninstallAll`, `IsInstalled`. `UninstallAll` does **not** hardcode the
    three core hacks: like `uninstall.sh`, it discovers every
    `push-hack-*` init.d script on the device first (`ls /etc/init.d/ |
    grep '^push-hack-'`) and removes each one — this is what also cleans
    up hacks installed later via Push Hack Catalog, not just this
    installer's own three. `IsInstalled` only checks the three core hacks
    (used to decide "Install" vs "Uninstall" on the final screen); nothing
    here purges the top-level `push-hack` directory or logs, matching
    `uninstall.sh`'s non-`--purge` default.
- `frontend/index.html`, `frontend/src/main.ts`, `frontend/public/style.css`
  — four screens, toggled by a `.is-active` class (no router at this size):
  - **Welcome**: title, a yellow warning box (not approved/endorsed by
    Ableton, back up first, must uninstall before a Push OS update, this app
    uses its own dedicated SSH key), a "don't contact Ableton Support, join
    the Discord" note, a required "I understand the risks" checkbox gating
    Continue (mirrors `install.sh`'s typed "Yes" disclaimer gate), and a
    Cancel button that calls Wails' `Application.Quit()`.
  - **Connect**: a hostname field (defaults to `push.local`), which
    re-checks reachability 500ms after each keystroke via `ConnectService`.
    Shows a live status line, enables "Continue" only once reachable, and a
    "Can't connect?" toggle reveals a hint that the same field accepts a
    manual IP address — no separate field, since editing the one input is
    simpler than juggling two. Continue first calls
    `SSHKeyService.AlreadyAuthorized` and skips straight past SSH key setup
    if it's already true.
  - **SSH key setup**: generates/loads the key via `SSHKeyService.EnsureKey`,
    shows its fingerprint, copies the key with Wails' native
    `Clipboard.SetText` (not the browser `navigator.clipboard` API, which
    silently no-ops in this webview without a direct user gesture — a
    "Copy key" button is also there as the reliable manual fallback), opens
    Push's `/ssh` page, then awaits `WaitForKeyAccepted` showing a "waiting"
    status until it resolves.
  - **Install/uninstall**: always all three core hacks together (no
    hack-selection screen — the user's call: push-hack's own `install.sh`
    treats these as the framework's non-optional core). On entry, calls
    `PushInstallService.IsInstalled` and shows a big green checkmark status
    ("Push Hack is already installed on Push") above the button row if so,
    disabling "Install push-hack" and enabling "Uninstall push-hack" (and
    vice versa when not installed — both buttons are always visible, never
    hidden, just disabled as appropriate). Once installed, two extra
    buttons open Push Manager (`:7701`) and Push Hack Catalog (`:7702`) in
    the system browser via `PushInstallService.OpenPushManager`/
    `OpenPushCatalog`.
  - Buttons are flat, square-cornered, and blue (`--accent-blue`) — the
    original Wails template's pink/red gradient and rounded corners are
    gone. `.btn-secondary` (outline, muted) is for non-primary actions like
    Cancel and the two "Open ..." buttons; `.btn-danger` (red) is for
    Uninstall specifically.
- `build/` — per-OS packaging config generated by `wails3 init`
  (`darwin/`, `windows/`, `linux/`), plus `config.yml` (app/company name,
  bundle identifier, description — feeds Info.plist, the NSIS installer,
  etc.). After changing `Taskfile.yml`'s `APP_NAME` or anything in
  `config.yml`, run `wails3 task common:update:build-assets` to
  regenerate the per-platform files from them — skipping this once caused
  a real bug: `darwin/Info.plist`'s `CFBundleExecutable` kept pointing at
  the pre-rename binary name, which is exactly what macOS shows to users
  as "can't be opened because it may be damaged" (a `CFBundleExecutable`/
  actual-binary-name mismatch, not a Gatekeeper/quarantine warning — same
  root cause and fix as an earlier push-tethered-app incident). That
  regen command does not preserve `darwin/dmg-file-icon.icns` or
  `dmg-background.png`, so the `.dmg` packaging task
  (`build/darwin/Taskfile.yml`'s `create:dmg`) only depends on
  `icons.icns`, not those two.
- `Taskfile.yml` — build/dev/package tasks, run via `wails3 task <name>`.

Update this file as things change.
