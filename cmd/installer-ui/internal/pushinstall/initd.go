package pushinstall

import (
	"fmt"
	"strings"
)

// genericInitd is a byte-for-byte port of lib/common.sh's
// generate_initd_script — used for any hack (push-manager, push-catalog)
// that doesn't ship its own service.initd template. push-display does ship
// one (vendored in hacks.go) because it needs to do more than start/stop a
// plain binary — see renderCustomInitd.
func genericInitd(hackID, binaryPath, configPath, logPath string) string {
	svcName := serviceName(hackID)
	return fmt.Sprintf(`#!/bin/sh
### BEGIN INIT INFO
# Provides:          %[1]s
# Required-Start:    $network $local_fs
# Required-Stop:     $network
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: push-hack: %[2]s
### END INIT INFO

DAEMON="%[3]s"
PIDFILE="/var/run/%[1]s.pid"
LOGFILE="%[4]s"
CONFIG="%[5]s"

[ -x "$DAEMON" ] || exit 0

start() {
    echo "Starting %[1]s..."
    if [ -f "$PIDFILE" ] && kill -0 $(cat "$PIDFILE") 2>/dev/null; then
        echo "%[1]s already running"
        return 0
    fi
    mkdir -p "$(dirname "$LOGFILE")"
    # Run at lowest CPU priority (nice 19), limit to 64MB virtual memory
    nice -n 19 "$DAEMON" -config "$CONFIG" >> "$LOGFILE" 2>&1 &
    echo $! > "$PIDFILE"
    echo "%[1]s started (PID $(cat $PIDFILE))"
}

stop() {
    echo "Stopping %[1]s..."
    if [ -f "$PIDFILE" ]; then
        kill $(cat "$PIDFILE") 2>/dev/null || true
        rm -f "$PIDFILE"
    fi
    echo "%[1]s stopped"
}

status() {
    if [ -f "$PIDFILE" ] && kill -0 $(cat "$PIDFILE") 2>/dev/null; then
        echo "%[1]s is running (PID $(cat $PIDFILE))"
        return 0
    else
        echo "%[1]s is not running"
        return 1
    fi
}

case "$1" in
    start)   start   ;;
    stop)    stop    ;;
    restart) stop; sleep 1; start ;;
    status)  status  ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac
`, svcName, hackID, binaryPath, logPath, configPath)
}

// renderCustomInitd substitutes the same {{...}} placeholders install.sh's
// install_hack_service does before deploying a hack's own service.initd
// template. Only push-display uses one; port/hack-id placeholders are
// substituted for parity even though its template doesn't reference them.
func renderCustomInitd(tmpl []byte, hackID, hackDir, logDir string) []byte {
	s := string(tmpl)
	s = strings.ReplaceAll(s, "{{HACK_ID}}", hackID)
	s = strings.ReplaceAll(s, "{{SVC_NAME}}", serviceName(hackID))
	s = strings.ReplaceAll(s, "{{HACK_DIR}}", hackDir)
	s = strings.ReplaceAll(s, "{{LOG_DIR}}", logDir)
	s = strings.ReplaceAll(s, "{{PORT}}", "")
	return []byte(s)
}
