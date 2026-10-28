package pushinstall

import (
	"reflect"
	"strings"
	"testing"
)

func TestScriptPathsFromTree(t *testing.T) {
	body := []byte(`{"truncated":false,"tree":[
	 {"path":"remote-script/PushHack/__init__.py","type":"blob"},
	 {"path":"remote-script/PushHack/ns","type":"tree"},
	 {"path":"remote-script/PushHack/ns/track.py","type":"blob"},
	 {"path":"remote-script/PushHack/PushHack.py","type":"blob"},
	 {"path":"remote-script/PushHack/__pycache__/x.cpython-311.pyc","type":"blob"},
	 {"path":"remote-script/PushHack/ns/.DS_Store","type":"blob"},
	 {"path":"remote-script/PushHack/._hub.py","type":"blob"},
	 {"path":"remote-script/tests/test_hub.py","type":"blob"},
	 {"path":"README.md","type":"blob"}]}`)
	got, err := scriptPathsFromTree(body)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"PushHack.py", "__init__.py", "ns/track.py"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestScriptPathsFromTreeRefusals(t *testing.T) {
	for name, body := range map[string]string{
		"truncated": `{"truncated":true,"tree":[{"path":"remote-script/PushHack/a.py","type":"blob"}]}`,
		"traversal": `{"tree":[{"path":"remote-script/PushHack/../../etc/passwd","type":"blob"}]}`,
		"absolute":  `{"tree":[{"path":"remote-script/PushHack//etc/x","type":"blob"}]}`,
		"backslash": `{"tree":[{"path":"remote-script/PushHack/a\\b.py","type":"blob"}]}`,
		"not json":  `nope`,
		"dot part":  `{"tree":[{"path":"remote-script/PushHack/./a.py","type":"blob"}]}`,
	} {
		if _, err := scriptPathsFromTree([]byte(body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSafeScriptPath(t *testing.T) {
	for _, ok := range []string{"a.py", "ns/track.py", "a/b/c.py"} {
		if !safeScriptPath(ok) {
			t.Errorf("%q refused", ok)
		}
	}
	for _, bad := range []string{"", "/a.py", "../a.py", "a/../b.py", "a//b.py", "a/./b.py", `a\b.py`, "a/"} {
		if safeScriptPath(bad) {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestCheckScript(t *testing.T) {
	if err := checkScript(scriptFiles{"__init__.py": []byte("x"), "PushHack.py": []byte("x")}); err != nil {
		t.Fatal(err)
	}
	if checkScript(scriptFiles{"__init__.py": []byte("x")}) == nil {
		t.Error("a script without PushHack.py passed")
	}
	if checkScript(scriptFiles{"__init__.py": nil, "PushHack.py": []byte("x")}) == nil {
		t.Error("an empty __init__.py passed")
	}
	if checkScript(nil) == nil {
		t.Error("no files passed")
	}
}

func TestRemoteDirsParentsFirst(t *testing.T) {
	got := remoteDirs(scriptFiles{"a.py": nil, "ns/track.py": nil, "ns/deep/x.py": nil, "other/y.py": nil})
	want := []string{"", "ns", "other", "ns/deep"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := remoteDirs(scriptFiles{"a.py": nil}); !reflect.DeepEqual(got, []string{""}) {
		t.Fatalf("flat: %v", got)
	}
}

// The snapshot built into the installer must be a script Live can load.
func TestEmbeddedScript(t *testing.T) {
	files, err := embeddedScriptFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{"__init__.py", "PushHack.py", "protocol.py", "hub.py", "server.py", "ns/__init__.py", "ns/transport.py"} {
		if len(files[need]) == 0 {
			t.Errorf("embedded script lacks %s", need)
		}
	}
	for p := range files {
		if !safeScriptPath(p) || skipScriptFile(p) {
			t.Errorf("embedded path %q should not be there", p)
		}
		if strings.HasSuffix(p, ".pyc") {
			t.Errorf("compiled file %q embedded", p)
		}
	}
	if !strings.Contains(string(files["protocol.py"]), "SCRIPT_VERSION") {
		t.Error("protocol.py has no SCRIPT_VERSION")
	}
}

func TestScriptDir(t *testing.T) {
	if got := scriptDir("/data"); got != "/data/Music/Ableton/User Library/Remote Scripts/PushHack" {
		t.Fatalf("scriptDir = %q", got)
	}
}
