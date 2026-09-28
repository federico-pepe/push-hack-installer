package pushinstall

import (
	"encoding/json"
	"fmt"
	"strings"
)

// rewriteHackJSON mirrors install.sh's deploy_hack hack.json handling:
// resolve the ${USER_DATA} placeholder (used in allowed_roots) and inject
// the two remote-path fields the running hack needs to find its own files.
// install.sh does this with sed as a text substitution before a JSON parse
// ever happens on the Push side, so the substitution here is also done on
// raw text — never inside an already-decoded string value — to behave
// identically regardless of where ${USER_DATA} appears in the file.
func rewriteHackJSON(raw []byte, userDataDir, pushHackDir, remoteHackDir string) ([]byte, error) {
	text := strings.ReplaceAll(string(raw), "${USER_DATA}", userDataDir)

	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &fields); err != nil {
		return nil, fmt.Errorf("parse hack.json: %w", err)
	}

	pushHackDirJSON, err := json.Marshal(pushHackDir)
	if err != nil {
		return nil, err
	}
	remoteHackDirJSON, err := json.Marshal(remoteHackDir)
	if err != nil {
		return nil, err
	}
	fields["_push_hack_dir"] = pushHackDirJSON
	fields["_hack_dir"] = remoteHackDirJSON

	out, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode hack.json: %w", err)
	}
	return out, nil
}
