#!/usr/bin/env bash
# Refresh the embedded snapshot (offline fallback) from a checkout of
# ableton-push-hack: the three hacks' binaries and hack.json, push-display's
# init.d template, and the PushHack Remote Script.
#
#   scripts/vendor-from-push-hack.sh [PATH]     (default: ../ableton-push-hack)
#
# Run it before a release, with the checkout on the release commit.
set -euo pipefail
cd "$(dirname "$0")/.."

src="${1:-../ableton-push-hack}"
[ -d "$src/hacks/push-manager" ] || { echo "no $src/hacks/push-manager. Pass the path of an ableton-push-hack checkout." >&2; exit 1; }
dst=cmd/installer-ui/internal/pushinstall/vendor

cp "$src/hacks/push-manager/push-manager" "$src/hacks/push-manager/hack.json" "$dst/push-manager/"
cp "$src/hacks/push-catalog/push-catalog" "$src/hacks/push-catalog/hack.json" "$dst/push-catalog/"
cp "$src/hacks/push-display/push_hook.so" "$src/hacks/push-display/hack.json" "$src/hacks/push-display/service.initd" "$dst/push-display/"

rm -rf "$dst/remote-script"
mkdir -p "$dst/remote-script"
(cd "$src/remote-script" && COPYFILE_DISABLE=1 tar -cf - --exclude='__pycache__' --exclude='*.pyc' --exclude='.DS_Store' --exclude='._*' PushHack) | tar -xf - -C "$dst/remote-script"

commit="$(git -C "$src" rev-parse HEAD)"
printf 'ableton-push-hack %s\n' "$commit" > "$dst/SOURCE"
echo "vendored from ableton-push-hack $commit"
git status --short -- "$dst"
