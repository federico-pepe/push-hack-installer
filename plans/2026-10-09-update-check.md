# Update check

Port of push-tethered-app's update check (`internal/updatecheck`).

- Copy `internal/updatecheck` with its tests. Set `Repo` to
  `federico-pepe/push-hack-installer`.
- Add `internal/version` with `Version = "dev"`. Set it with `-ldflags -X`
  in the darwin, linux, and windows Taskfiles from `APP_VERSION`. CI
  already sets `APP_VERSION` to the release tag.
- Add `UpdateService` (`Check`, `OpenReleasePage`). The frontend never
  passes a URL, so it cannot open an arbitrary page.
- Add a yellow banner above the footer. The check always runs at startup.
  There is no setting to turn it off.
- A failed check is logged and shows nothing.
