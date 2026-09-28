package pushinstall

import "fmt"

// userDataCandidates matches lib/common.sh's detect_user_data_dir — /data
// is the big NVMe partition AbletonOS uses for all user content; the other
// two are fallbacks for older/different OS layouts.
var userDataCandidates = []string{"/data", "/data/UserData", "/home/ableton"}

// detectUserDataDir finds the first writable candidate directory, as
// ableton. Falls back to "/data" if none test writable, same as the bash
// original (which warns rather than failing outright).
func detectUserDataDir(ableton *client) string {
	for _, dir := range userDataCandidates {
		cmd := fmt.Sprintf("test -d %s && test -w %s", shellQuote(dir), shellQuote(dir))
		if ableton.runOK(cmd) {
			return dir
		}
	}
	return "/data"
}
