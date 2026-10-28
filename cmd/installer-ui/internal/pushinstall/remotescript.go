package pushinstall

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
)

// The PushHack Remote Script is the one Live script every hack uses. It goes
// to Live's User Library, not to hacks/<id>/, and has no service. Live reads
// it at its next start, so the user still selects it once in a control-surface
// slot (see the screen text in the frontend).
//
// Like the three hacks, it comes fresh from ableton-push-hack's main branch
// when GitHub answers, else from the snapshot embedded at build time.

//go:embed all:vendor/remote-script
var embeddedScript embed.FS

const (
	scriptName      = "PushHack"
	scriptRepoDir   = "remote-script/" + scriptName
	scriptEmbedRoot = "vendor/remote-script/" + scriptName
	// treeAPI lists the repo's files in one call. 60 calls an hour without a
	// token; the embedded copy covers a miss.
	treeAPI = "https://api.github.com/repos/federico-pepe/ableton-push-hack/git/trees/main?recursive=1"
	// userLibraryScripts is relative to the user data dir (/data).
	userLibraryScripts = "/Music/Ableton/User Library/Remote Scripts"
)

// oldScripts are the per-hack scripts PushHack replaces. The installer only
// reports them; removing one is the user's call (Push Manager, System page).
var oldScripts = []string{"PushHackBrowser", "PushHackArrangement"}

// scriptFiles is a script as a path -> content map. Paths are relative to the
// script folder, forward slashes ("ns/track.py").
type scriptFiles map[string][]byte

// scriptPathsFromTree picks the script's files out of a GitHub tree listing.
// It keeps only blobs under remote-script/PushHack/, and drops caches and
// Mac metadata. A path that tries to leave the folder is an error.
func scriptPathsFromTree(body []byte) ([]string, error) {
	var t struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, fmt.Errorf("tree: %w", err)
	}
	if t.Truncated {
		return nil, fmt.Errorf("tree: listing is truncated")
	}
	var out []string
	for _, e := range t.Tree {
		if e.Type != "blob" || !strings.HasPrefix(e.Path, scriptRepoDir+"/") {
			continue
		}
		rel := strings.TrimPrefix(e.Path, scriptRepoDir+"/")
		if !safeScriptPath(rel) {
			return nil, fmt.Errorf("tree: unsafe path %q", e.Path)
		}
		if skipScriptFile(rel) {
			continue
		}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

// safeScriptPath: a relative path with no "..", no leading slash, no empty or
// dot parts, no backslash.
func safeScriptPath(rel string) bool {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, `\`) {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return path.Clean(rel) == rel
}

func skipScriptFile(rel string) bool {
	base := path.Base(rel)
	return strings.Contains(rel, "__pycache__/") || strings.HasSuffix(base, ".pyc") ||
		base == ".DS_Store" || strings.HasPrefix(base, "._")
}

// checkScript: a script Live can load has __init__.py and PushHack.py.
func checkScript(files scriptFiles) error {
	for _, need := range []string{"__init__.py", "PushHack.py"} {
		if len(files[need]) == 0 {
			return fmt.Errorf("remote script is missing %s", need)
		}
	}
	return nil
}

// remoteDirs lists every folder the files need, parents first, relative to the
// script folder ("" is the folder itself).
func remoteDirs(files scriptFiles) []string {
	seen := map[string]bool{"": true}
	for p := range files {
		for d := path.Dir(p); d != "." && !seen[d]; d = path.Dir(d) {
			seen[d] = true
		}
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) < len(out[j])
		}
		return out[i] < out[j]
	})
	return out
}

// embeddedScriptFiles reads the snapshot built into this installer.
func embeddedScriptFiles() (scriptFiles, error) {
	files := scriptFiles{}
	err := fs.WalkDir(embeddedScript, scriptEmbedRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, scriptEmbedRoot+"/")
		if skipScriptFile(rel) {
			return nil
		}
		b, err := embeddedScript.ReadFile(p)
		if err != nil {
			return err
		}
		files[rel] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, checkScript(files)
}

// fetchScript downloads the script fresh from ableton-push-hack's main branch.
// ok is false on any failure; the caller then uses the embedded snapshot. It is
// all-or-nothing: half a script from GitHub and half from the snapshot would be
// a version mix.
func fetchScript() (files scriptFiles, ok bool) {
	body, err := fetchURL(treeAPI)
	if err != nil {
		return nil, false
	}
	paths, err := scriptPathsFromTree(body)
	if err != nil || len(paths) == 0 {
		return nil, false
	}
	files = scriptFiles{}
	for _, rel := range paths {
		b, err := fetchFile(scriptRepoDir + "/" + rel)
		if err != nil {
			return nil, false
		}
		files[rel] = b
	}
	if checkScript(files) != nil {
		return nil, false
	}
	return files, true
}

func fetchURL(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// scriptSource picks the files to install and says where they came from.
func scriptSource() (scriptFiles, string, error) {
	if files, ok := fetchScript(); ok {
		return files, "latest from GitHub", nil
	}
	files, err := embeddedScriptFiles()
	if err != nil {
		return nil, "", err
	}
	return files, "bundled version", nil
}

// scriptDir is where Live looks for the script.
func scriptDir(userDataDir string) string {
	return userDataDir + userLibraryScripts + "/" + scriptName
}

// installRemoteScript copies the PushHack Remote Script into Live's User
// Library. It replaces the whole folder, so a file a newer script dropped does
// not stay behind. It returns the lines to show the user. If Live's User
// Library is not on this Push (Live has never run), it says so and installs
// nothing; that is not an error.
func installRemoteScript(abletonClient *client, userDataDir string) (lines []string, err error) {
	userLibrary := userDataDir + "/Music/Ableton/User Library"
	if !abletonClient.runOK("test -d " + shellQuote(userLibrary)) {
		return []string{"PushHack Remote Script: not installed (Live's User Library was not found; start Live once, then install again)"}, nil
	}
	files, source, err := scriptSource()
	if err != nil {
		return nil, fmt.Errorf("PushHack Remote Script: %w", err)
	}
	root := scriptDir(userDataDir)
	if _, err := abletonClient.run("mkdir -p " + shellQuote(path.Dir(root)) + " && rm -rf " + shellQuote(root)); err != nil {
		return nil, fmt.Errorf("PushHack Remote Script: prepare folder: %w", err)
	}
	for _, d := range remoteDirs(files) {
		p := root
		if d != "" {
			p = root + "/" + d
		}
		if _, err := abletonClient.run("mkdir -p " + shellQuote(p)); err != nil {
			return nil, fmt.Errorf("PushHack Remote Script: create %s: %w", p, err)
		}
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if err := abletonClient.copyBytes(files[n], root+"/"+n, 0644); err != nil {
			return nil, fmt.Errorf("PushHack Remote Script: copy %s: %w", n, err)
		}
	}
	lines = append(lines, fmt.Sprintf("PushHack Remote Script: installed (%s)", source))
	for _, old := range oldScripts {
		if abletonClient.runOK("test -d " + shellQuote(path.Dir(root)+"/"+old)) {
			lines = append(lines, fmt.Sprintf("Found the old %s script. Remove it in Push Manager (System page) once PushHack is selected in Live.", old))
		}
	}
	return lines, nil
}

// removeRemoteScript deletes the PushHack folder and nothing else in the User
// Library. Best effort, like the rest of uninstall.
func removeRemoteScript(abletonClient *client, userDataDir string) bool {
	root := scriptDir(userDataDir)
	if !abletonClient.runOK("test -d " + shellQuote(root)) {
		return false
	}
	abletonClient.runOK("rm -rf " + shellQuote(root))
	return true
}

// remoteScriptInstalled reports whether Live's User Library has the script.
func remoteScriptInstalled(abletonClient *client, userDataDir string) bool {
	return abletonClient.runOK("test -f " + shellQuote(scriptDir(userDataDir)+"/__init__.py"))
}
