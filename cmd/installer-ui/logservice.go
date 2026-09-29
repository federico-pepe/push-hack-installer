package main

import (
	"time"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/applog"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// LogService is bound to the frontend for the "Export Debug Logs" footer
// link — available from any screen, not just the install screen, since a
// user can hit trouble anywhere in the flow (e.g. reachability checks on
// the Connect screen).
type LogService struct{}

// Export prompts for a save location and writes the full current-run debug
// log there as a .txt file. Returns "" on success; also "" (not an error)
// if the user cancels the save dialog.
func (s *LogService) Export() string {
	defaultName := "push-hack-installer-log-" + time.Now().Format("20060102-150405") + ".txt"

	destPath, err := application.Get().Dialog.SaveFile().
		SetMessage("Export Debug Logs").
		SetFilename(defaultName).
		AddFilter("Text Files", "*.txt").
		PromptForSingleSelection()
	if err != nil {
		return err.Error()
	}
	if destPath == "" {
		return ""
	}

	if err := applog.Export(destPath); err != nil {
		applog.Printf("export debug logs failed: %v", err)
		return err.Error()
	}
	applog.Printf("debug logs exported to %s", destPath)
	return ""
}
