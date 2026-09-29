// Command installer-ui is the desktop app that installs push-hack onto an
// Ableton Push 3 over SSH, without the user opening a terminal. See
// plans/2026-09-27-gui-installer.md for the planned install/connect flow.
package main

import (
	"embed"
	"log"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/applog"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Non-fatal: a debug log we couldn't open isn't worth refusing to start
	// the installer over. applog.Printf silently no-ops if this failed.
	if err := applog.Init(); err != nil {
		log.Printf("debug log unavailable: %v", err)
	} else {
		applog.Printf("push-hack-installer starting, log at %s", applog.Path())
	}

	app := application.New(application.Options{
		Name:        "Push Hack Installer",
		Description: "Installer for push-hack on Ableton Push 3",
		Services: []application.Service{
			application.NewService(&ConnectService{}),
			application.NewService(&SSHKeyService{}),
			application.NewService(&PushInstallService{}),
			application.NewService(&LogService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Push Hack Installer",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:  1000,
		Height: 618,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(6, 7, 15),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
