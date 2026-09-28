# Changelog

All notable changes to this project are documented here.

## [Unreleased]

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
- Hack selection and the actual install are still not built.
