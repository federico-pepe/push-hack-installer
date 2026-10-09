package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/applog"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/openurl"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/updatecheck"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/version"
)

// UpdateService is bound to the frontend for the "new version" banner. It
// only looks and opens a web page; it never downloads or installs anything.
type UpdateService struct {
	mu   sync.Mutex
	last *updatecheck.Result
}

// Check asks GitHub for a newer release. Returns "" when the app is up to
// date, when this is a dev build, or when GitHub is not reachable (a failed
// check is logged, never shown: the installer works offline from the
// user's point of view). Otherwise returns the newer version tag.
func (s *UpdateService) Check() string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	res, err := updatecheck.Check(ctx, version.Version)
	if err != nil {
		applog.Printf("update check failed: %v", err)
		return ""
	}
	if res == nil {
		applog.Printf("update check: %s is up to date", version.Version)
		return ""
	}
	applog.Printf("update check: %s is available (running %s)", res.Version, version.Version)
	s.mu.Lock()
	s.last = res
	s.mu.Unlock()
	return res.Version
}

// OpenReleasePage opens the release page found by the last Check in the
// default browser. It takes no URL from the frontend. Returns "" on success.
func (s *UpdateService) OpenReleasePage() string {
	s.mu.Lock()
	res := s.last
	s.mu.Unlock()
	if res == nil {
		return "no update found"
	}
	if err := openurl.Open(res.URL); err != nil {
		applog.Printf("open release page failed: %v", err)
		return fmt.Sprint(err)
	}
	return ""
}
