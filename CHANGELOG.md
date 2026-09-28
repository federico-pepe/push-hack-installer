# Changelog

All notable changes to this project are documented here.

## [Unreleased]

- Fixed: the Windows installer only worked on amd64 — Windows on ARM
  (e.g. Parallels on Apple Silicon) refused it with "this product can't
  be installed on the current Windows architecture: supports amd64".
  CI now builds both `amd64` and `arm64` and packages them into one
  universal NSIS installer (the template already supported this; CI just
  wasn't building both architectures to feed it).

## [0.1.0-beta] - 2026-09-28

First release. The full guided flow works end to end against a real
Push 3: find it on the network, get its SSH key accepted, install or
uninstall push-hack (Push Manager, Push Display, Push Hack Catalog),
verified with CI builds on macOS, Windows, and Linux.

- Added CI: `.github/workflows/build.yml` builds macOS (`.dmg`), Windows
  (NSIS installer `.exe`), and Linux (`.AppImage`) on `workflow_dispatch`
  and `v*` tags, adapted from push-tethered-app's own workflow but simpler
  (no cgo dependency here, so no MSYS2/mingw toolchain step or Docker
  cross-compile path needed). A tag push also publishes a GitHub Release
  with all three artifacts attached.
- Code audit and cleanup: fixed the root `.gitignore`'s blanket `build/`
  rule, which was silently excluding `cmd/installer-ui/build/` (Wails'
  packaging assets — icons, platform configs, NSIS/AppImage/nfpm scripts)
  from every commit so far; this would have broken CI and fresh-clone
  builds. Removed a dead, never-called `copyFile` function. Bumped
  `typescript` from `^4.9.3` (too old to understand the project's own
  `tsconfig.json`, which needs TS 5+ for `moduleResolution: "bundler"`) to
  `^5.7.0`, and pinned `@wailsio/runtime` to its resolved version instead
  of a floating `latest`. Removed the leftover generic Wails scaffold
  README under `cmd/installer-ui/` (referenced files that don't exist in
  this project). Fixed a couple of doc typos and one stale reference to a
  never-ported `scripts/discover.sh`.
- Fixed macOS `.app` showing "can't be opened because it may be damaged":
  `CFBundleExecutable` in `build/darwin/Info.plist`/`Info.dev.plist` was
  left over from before the app's binary name was corrected to "Push Hack
  Installer", so it pointed at a file that no longer existed in the
  bundle. Regenerating build assets after the rename fixed it (same root
  cause and fix as an earlier push-tethered-app incident).
- Renamed the built app from `installer-ui` to `Push Hack Installer`
  (`Taskfile.yml`'s `APP_NAME`, plus `build/config.yml`'s `companyName`/
  `productName`/`productIdentifier`/`description`/`copyright`, which were
  still 100% unedited wails3 scaffold placeholders — "My Company", "My
  Product", `com.mycompany.myproduct`, etc. — and would have shipped as-is
  in the packaged installer/dmg metadata).
- Removed the `.dmg`'s background image per request (plain background
  now) and stopped depending on `dmg-file-icon.icns`, which doesn't
  survive a rerun of `wails3 task common:update:build-assets`.
- Hack binaries are no longer install-time-frozen to whatever was vendored
  into this installer at its own build time: `InstallAll` now tries
  fetching each hack's binary/`hack.json`/init.d script fresh from
  `ableton-push-hack`'s `main` branch first, falling back to the embedded
  vendored copy only if that fails (offline, GitHub unreachable, etc.).
  The install summary notes which source was used per hack.

- Project scaffolded: repo, docs, and design plan created.
- Added the Wails v3 desktop app skeleton (`cmd/installer-ui`, macOS/Windows/Linux
  only, no iOS/Android): a Welcome screen with the app's purpose, safety
  notes, and a "Continue" button.
- Added a Connect screen: type a hostname (defaults to `push.local`), the
  app checks reachability live and enables "Continue" once found.
- Added an SSH key setup screen: generates a keypair if needed, copies the
  public key, opens Push's `/ssh` page in the browser, then waits (no
  "press Enter") until Push accepts the key.
- The generated key now lives at `~/.ssh/push_hack_id_rsa`, not `~/.ssh/id_rsa`
  — a dedicated filename so this app can never collide with a key the user
  already has for something else.
- The Connect screen now checks whether Push already trusts this computer's
  key and skips SSH key setup entirely when it does, instead of always
  showing it.
- Rewrote the Welcome screen as a real safety disclaimer: not
  approved/endorsed by Ableton, back up first, must uninstall before a Push
  OS update, a note not to contact Ableton Support (join the Discord
  instead), and a required "I understand the risks" checkbox gating
  Continue — mirrors `install.sh`'s typed "Yes" gate. Added a Cancel button
  that quits the app.
- Restyled buttons: flat, square-cornered, blue — replacing the original
  Wails template's pink/red gradient and rounded corners.
- Added the install/uninstall screen — the last piece. Always deploys all
  three core hacks together (no hack-selection screen): Push Manager, Push
  Display, Push Hack Catalog. Checks whether push-hack is already on the
  device and shows "Install push-hack" or "Uninstall push-hack"
  accordingly (both buttons always visible, the inactive one disabled), a
  green checkmark when already installed, and a warning that either action
  briefly restarts Push3 (and Live). Once installed, two more buttons open
  Push Manager and Push Hack Catalog's web UIs in the browser.
- Uninstall discovers and removes *every* `push-hack-*` service on the
  device, not just the three this app installs — matching
  `scripts/uninstall.sh`, this also cleans up hacks installed later via
  Push Hack Catalog. The top-level `push-hack` data directory and logs are
  left in place, matching `uninstall.sh`'s non-`--purge` default.
- The install/uninstall logic is a Go port of `scripts/install.sh` and
  `scripts/uninstall.sh`, scoped to real Push 3 hardware only (it never
  detects the init system — Push always runs sysvinit, so the bash
  originals' systemd branch isn't ported). File transfer speaks the
  classic SCP protocol by hand, not SFTP, since push-hack's own scripts use
  `scp` and SFTP's availability on Push's SSH server is unverified.
- Fixed: Uninstall could silently do nothing while still reporting success.
  Two causes found against a real device: (1) mDNS can hand back an IPv6
  link-local address for `push.local` that silently times out while IPv4
  works fine — every SSH/TCP dial in the app now forces IPv4. (2) Push's
  SSH server has been observed to intermittently return blank output for
  an otherwise-successful command; Uninstall's service-discovery listing
  now retries on a blank result instead of treating it as "nothing
  installed."
- Welcome screen: title is "Push Hack Installer", the disclaimer and
  Discord note now span the full width (matching the warning box below
  them), the warning box is solid vivid yellow with dark text instead of a
  tinted overlay, its copy was rewritten (mentions AI-assisted community
  development, restructured into clearer paragraphs), and both buttons are
  wider.
- Connect screen: subtitle now says to check Push is on and on the same
  Wi-Fi network, replacing the vaguer "we'll look for it."
- Install/uninstall screen: title is "Push Hack Installer", subtitle
  explains it installs the core modules or uninstalls everything including
  Catalog-installed hacks, the "already installed" status is now bigger
  and sits above the button row (the restart warning box was removed
  entirely), and all buttons are bigger.
