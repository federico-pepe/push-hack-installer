package pushinstall

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// repoRawBase serves raw file content from ableton-push-hack's main branch.
// There's no tagged-release artifact story for individual hack binaries
// there (see that repo's CLAUDE.md: "Committed pre-built binaries" — they
// live straight in the tree), so main is the only source of "latest".
const repoRawBase = "https://raw.githubusercontent.com/federico-pepe/ableton-push-hack/main"

// fetchTimeout bounds each individual file download.
const fetchTimeout = 15 * time.Second

// remoteFiles maps a hack ID to the ableton-push-hack repo paths this
// installer needs for it — the same set vendor/ (hacks.go) embeds a
// build-time snapshot of.
var remoteFiles = map[string]struct {
	binary      string
	hackJSON    string
	customInitd string // empty if the hack has none (see hacks.go)
}{
	"push-manager": {
		binary:   "hacks/push-manager/push-manager",
		hackJSON: "hacks/push-manager/hack.json",
	},
	"push-catalog": {
		binary:   "hacks/push-catalog/push-catalog",
		hackJSON: "hacks/push-catalog/hack.json",
	},
	"push-display": {
		binary:      "hacks/push-display/push_hook.so",
		hackJSON:    "hacks/push-display/hack.json",
		customInitd: "hacks/push-display/service.initd",
	},
}

// fetchLatest tries to download h's binary, hack.json, and (if it has one)
// custom init.d script fresh from ableton-push-hack's main branch. ok is
// false if anything about the fetch failed — a network error, a non-200,
// or an empty body — in which case the caller falls back to the binary
// embedded at this installer's own build time (see hacks.go's vendor/).
// This is all-or-nothing per hack: mixing a freshly-fetched binary with a
// stale embedded hack.json (or vice versa) risks a version mismatch that's
// worse than just using one consistent source.
func fetchLatest(h hack) (binary, hackJSON, customInitd []byte, ok bool) {
	paths, known := remoteFiles[h.id]
	if !known {
		return nil, nil, nil, false
	}

	binary, err := fetchFile(paths.binary)
	if err != nil {
		return nil, nil, nil, false
	}
	hackJSON, err = fetchFile(paths.hackJSON)
	if err != nil {
		return nil, nil, nil, false
	}
	if paths.customInitd != "" {
		customInitd, err = fetchFile(paths.customInitd)
		if err != nil {
			return nil, nil, nil, false
		}
	}
	return binary, hackJSON, customInitd, true
}

func fetchFile(relPath string) ([]byte, error) {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(repoRawBase + "/" + relPath)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", relPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", relPath, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", relPath, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("fetch %s: empty response", relPath)
	}
	return data, nil
}
