// Package applog is a single append-only debug log for the whole app run —
// every screen transition, SSH command, and error funnels through here so
// the "Export Debug Logs" footer link can hand the user one file that
// covers everything, without them having to reproduce a problem while
// someone watches over their shoulder.
package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu     sync.Mutex
	file   *os.File
	logger *log.Logger
	path   string
)

// Init opens a fresh log file for this run under the OS's standard cache
// directory. Safe to call once at startup; if it fails, Printf and Export
// become no-ops/errors instead of crashing the app — a missing debug log
// isn't worth taking the installer down over.
func Init() error {
	dir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("find cache directory: %w", err)
	}
	logDir := filepath.Join(dir, "push-hack-installer", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	name := "install-" + time.Now().Format("20060102-150405") + ".log"
	f, err := os.OpenFile(filepath.Join(logDir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	mu.Lock()
	file = f
	path = f.Name()
	logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	mu.Unlock()
	return nil
}

// Printf appends one timestamped line to the log. Nil-safe — before Init
// (or if it failed), this quietly does nothing rather than panicking every
// call site into checking an error it can't usefully act on.
func Printf(format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if logger == nil {
		return
	}
	logger.Printf(format, args...)
}

// Path returns the current run's log file path, or "" if logging never
// started.
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return path
}

// Export copies the full current log to destPath, for the "Export Debug
// Logs" footer link.
func Export(destPath string) error {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return fmt.Errorf("no log file to export")
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("flush log: %w", err)
	}

	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read log: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("write log: %w", err)
	}
	return nil
}
