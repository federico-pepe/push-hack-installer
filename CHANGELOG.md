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
  "press Enter") until Push accepts the key. Hack selection and the actual
  install are still not built.
