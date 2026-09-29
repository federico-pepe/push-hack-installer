package main

import (
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/applog"
	"github.com/federico-pepe/push-hack-installer/cmd/installer-ui/internal/pushdiscover"
)

// ConnectService is bound to the frontend so the Connect screen can check
// reachability without shelling out — see pushdiscover for what "reachable"
// means at this stage (TCP only, no SSH auth yet).
type ConnectService struct{}

// CheckHost reports whether host is reachable on Push's SSH port, and a
// user-facing reason when it is not.
func (s *ConnectService) CheckHost(host string) (bool, string) {
	reachable, reason := pushdiscover.CheckHost(host)
	applog.Printf("CheckHost(%s): reachable=%v reason=%q", host, reachable, reason)
	return reachable, reason
}
