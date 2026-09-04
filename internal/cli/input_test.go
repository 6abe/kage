package cli_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/6abe/kage/internal/host"
)

func TestFocusWindowByAddressClassTitle(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "focus", "--window", "0x456")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	if len(h.Dispatch) != 1 || !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x456")) {
		t.Fatalf("dispatch %q", h.Dispatch)
	}
	if len(h.WtypeCalls) != 0 {
		t.Fatalf("focus must not type: %q", h.WtypeCalls)
	}
	var payload struct {
		OK     bool `json:"ok"`
		Window struct {
			Address string `json:"address"`
			Class   string `json:"class"`
		} `json:"window"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.OK || payload.Window.Address != "0x456" || payload.Window.Class != "kitty" {
		t.Fatalf("payload %s", out)
	}

	h = okHost()
	_, errb, code = execCLI(h, "focus", "--window", "Google-Chrome")
	if code != 0 {
		t.Fatalf("class exit %d %s", code, errb)
	}
	if !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x123")) {
		t.Fatalf("class dispatch %q", h.Dispatch)
	}

	h = okHost()
	_, errb, code = execCLI(h, "focus", "--window", "term")
	if code != 0 {
		t.Fatalf("title exit %d %s", code, errb)
	}
	if !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x456")) {
		t.Fatalf("title dispatch %q", h.Dispatch)
	}
}

func TestFocusAmbiguousExit2(t *testing.T) {
	h := okHost()
	h.JSON["clients"] = []byte(`[
	  {"address":"0x1","mapped":true,"at":[0,0],"size":[1,1],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"a","focusHistoryID":0},
	  {"address":"0x2","mapped":true,"at":[0,0],"size":[1,1],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"b","focusHistoryID":1}
	]`)
	out, errb, code := execCLI(h, "focus", "--window", "kitty")
	if code != 2 {
		t.Fatalf("exit %d want 2 stdout=%s stderr=%s", code, out, errb)
	}
	if len(h.Dispatch) != 0 {
		t.Fatalf("must not focus on ambiguity: %q", h.Dispatch)
	}
	var f struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error"`
		Matches []struct {
			Address string `json:"address"`
		} `json:"matches"`
	}
	if err := json.Unmarshal([]byte(errb), &f); err != nil {
		t.Fatal(err)
	}
	if f.OK || f.Error != "ambiguous window match" || len(f.Matches) != 2 {
		t.Fatalf("stderr %s", errb)
	}
}

func TestTypeAndPressGate(t *testing.T) {
	h := okHost()
	_, errb, code := execCLI(h, "type", "hello")
	if code == 0 {
		t.Fatal("type must refuse without allow")
	}
	if len(h.WtypeCalls) != 0 || len(h.Dispatch) != 0 {
		t.Fatal("refused type must not send keys")
	}
	for _, need := range []string{"--yes", "KAGE_ALLOW_INPUT=1", "allow_input = true"} {
		if !strings.Contains(errb, need) {
			t.Fatalf("gate error must name %q: %s", need, errb)
		}
	}

	h = okHost()
	_, errb, code = execCLI(h, "press", "Return")
	if code == 0 || !strings.Contains(errb, "--yes") {
		t.Fatalf("press gate: %d %s", code, errb)
	}

	h = okHost()
	_, _, code = execCLI(h, "type", "--yes", "hello")
	if code != 0 {
		t.Fatalf("--yes: %d", code)
	}
	if !reflect.DeepEqual(lastWtype(t, h), []string{"--", "hello"}) {
		t.Fatalf("wtype %q", h.WtypeCalls)
	}

	h = okHost()
	h.Environ["KAGE_ALLOW_INPUT"] = "1"
	_, _, code = execCLI(h, "type", "hello")
	if code != 0 {
		t.Fatalf("env: %d", code)
	}

	h = okHost()
	h.Allow = true
	_, _, code = execCLI(h, "press", "Tab")
	if code != 0 {
		t.Fatalf("config: %d", code)
	}
}

func TestTypeClearAndFocusTarget(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "type", "--yes", "--clear", "--window", "kitty", "hello")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	if !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x456")) {
		t.Fatalf("dispatch %q", h.Dispatch)
	}
	want := []string{"-M", "ctrl", "-k", "a", "-m", "ctrl", "--", "hello"}
	if !reflect.DeepEqual(lastWtype(t, h), want) {
		t.Fatalf("wtype %q want %q", h.WtypeCalls, want)
	}
	var payload struct {
		OK         bool     `json:"ok"`
		Clear      bool     `json:"clear"`
		ClearSends []string `json:"clear_sends"`
		N          int      `json:"n"`
		Window     struct {
			Address string `json:"address"`
		} `json:"window"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Clear || payload.N != 5 || payload.Window.Address != "0x456" {
		t.Fatalf("payload %s", out)
	}
	if !reflect.DeepEqual(payload.ClearSends, []string{"Ctrl+A"}) {
		t.Fatalf("clear_sends %v", payload.ClearSends)
	}

	h = okHost()
	_, _, code = execCLI(h, "type", "--yes", "--clear", "")
	if code != 0 {
		t.Fatalf("empty clear %d", code)
	}
	want = []string{"-M", "ctrl", "-k", "a", "-m", "ctrl", "-k", "BackSpace"}
	if !reflect.DeepEqual(lastWtype(t, h), want) {
		t.Fatalf("empty clear wtype %q", h.WtypeCalls)
	}
}

func TestTypeUsesFocusedWindowWithoutDispatch(t *testing.T) {
	h := okHost()
	_, errb, code := execCLI(h, "type", "--yes", "hi")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if len(h.Dispatch) != 0 {
		t.Fatalf("already focused: dispatch %q", h.Dispatch)
	}
	if !reflect.DeepEqual(lastWtype(t, h), []string{"--", "hi"}) {
		t.Fatalf("wtype %q", h.WtypeCalls)
	}
}

func TestPressKeys(t *testing.T) {
	keys := []string{"Return", "Tab", "Escape", "BackSpace", "space", "Left", "Right", "Up", "Down"}
	for _, key := range keys {
		h := okHost()
		out, errb, code := execCLI(h, "press", "--yes", "--window", "0x123", key)
		if code != 0 {
			t.Fatalf("%s: %d %s", key, code, errb)
		}
		if !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x123")) {
			t.Fatalf("%s dispatch %q", key, h.Dispatch)
		}
		if !reflect.DeepEqual(lastWtype(t, h), []string{"-k", key}) {
			t.Fatalf("%s wtype %q", key, h.WtypeCalls)
		}
		if !strings.Contains(out, `"key":"`+key+`"`) {
			t.Fatalf("%s json %s", key, out)
		}
	}
	h := okHost()
	_, errb, code := execCLI(h, "press", "--yes", "F1")
	if code == 0 || !strings.Contains(errb, "unsupported key") {
		t.Fatalf("unknown key: %d %s", code, errb)
	}
}

func TestMissingWtypeIsHint(t *testing.T) {
	h := okHost()
	delete(h.Paths, "wtype")
	out, errb, code := execCLI(h, "type", "--yes", "hello")
	if code == 0 {
		t.Fatal("missing wtype must not succeed")
	}
	if out != "" {
		t.Fatalf("stdout %s", out)
	}
	if !strings.Contains(errb, `"ok":false`) || !strings.Contains(errb, "wtype not found") {
		t.Fatalf("stderr %s", errb)
	}
	if !strings.Contains(errb, "omarchy pkg add wtype") {
		t.Fatalf("hint %s", errb)
	}
	if len(h.WtypeCalls) != 0 || len(h.Dispatch) != 0 {
		t.Fatal("must not pretend it typed")
	}
}

func TestObserveUngated(t *testing.T) {
	h := okHost()
	h.Environ["XDG_RUNTIME_DIR"] = t.TempDir()
	for _, args := range [][]string{
		{"windows"},
		{"monitors"},
		{"doctor"},
		{"see"},
		{"focus", "--window", "0x123"},
		{"dispatch", "hl.dsp.window.fullscreen()"},
	} {
		_, errb, code := execCLI(h, args...)
		if strings.Contains(errb, "input not allowed") {
			t.Fatalf("%v gated: %s", args, errb)
		}
		if code != 0 {
			t.Fatalf("%v exit %d %s", args, code, errb)
		}
	}
}

func TestKeystrokesNeverLogged(t *testing.T) {
	h := okHost()
	secret := "s3cret-keystroke-payload"
	out, errb, code := execCLI(h, "type", "--yes", secret)
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if strings.Contains(out, secret) || strings.Contains(errb, secret) {
		t.Fatal("keystrokes in command output")
	}
	for _, line := range h.Logs {
		if strings.Contains(line, secret) {
			t.Fatalf("keystrokes in log: %q", line)
		}
	}
	h.WtypeErr = errString("boom")
	_, errb, _ = execCLI(h, "type", "--yes", secret)
	if strings.Contains(errb, secret) {
		t.Fatalf("keystrokes in wtype error: %s", errb)
	}
}

func TestNoXdotool(t *testing.T) {
	h := okHost()
	h.Paths["xdotool"] = "/usr/bin/xdotool"
	h.Lookups = nil
	_, errb, code := execCLI(h, "type", "--yes", "hello")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	for _, name := range h.Lookups {
		if name == "xdotool" {
			t.Fatal("xdotool must not be used")
		}
	}
	if len(h.WtypeCalls) != 1 {
		t.Fatalf("want wtype, got %q", h.WtypeCalls)
	}
}

func TestFocusDoesNotNeedWtype(t *testing.T) {
	h := okHost()
	delete(h.Paths, "wtype")
	_, errb, code := execCLI(h, "focus", "--window", "0x123")
	if code != 0 {
		t.Fatalf("focus without wtype: %d %s", code, errb)
	}
}

func TestTypeAmbiguousDoesNotType(t *testing.T) {
	h := okHost()
	h.JSON["clients"] = []byte(`[
	  {"address":"0x1","mapped":true,"at":[0,0],"size":[1,1],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"a","focusHistoryID":0},
	  {"address":"0x2","mapped":true,"at":[0,0],"size":[1,1],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"b","focusHistoryID":1}
	]`)
	_, errb, code := execCLI(h, "type", "--yes", "--window", "kitty", "hello")
	if code != 2 {
		t.Fatalf("exit %d %s", code, errb)
	}
	if len(h.WtypeCalls) != 0 || len(h.Dispatch) != 0 {
		t.Fatalf("ambiguous must not type: wtype=%q dispatch=%q", h.WtypeCalls, h.Dispatch)
	}
}

func TestTypeNoFocusedWindow(t *testing.T) {
	h := okHost()
	h.JSON["activewindow"] = []byte(`{}`)
	h.JSON["clients"] = []byte(`[
	  {"address":"0x1","mapped":true,"at":[0,0],"size":[1,1],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"a","focusHistoryID":1}
	]`)
	_, errb, code := execCLI(h, "type", "--yes", "hello")
	if code == 0 || !strings.Contains(errb, "no focused window") {
		t.Fatalf("exit %d %s", code, errb)
	}
	if len(h.WtypeCalls) != 0 {
		t.Fatalf("wtype %q", h.WtypeCalls)
	}
}

func lastWtype(t *testing.T, h *host.Fake) []string {
	t.Helper()
	if len(h.WtypeCalls) != 1 {
		t.Fatalf("wtype calls %d: %q", len(h.WtypeCalls), h.WtypeCalls)
	}
	return h.WtypeCalls[0]
}

// argv Host.HyprctlDispatch receives, i.e. hyprctl dispatch <this>
func wantFocus(addr string) []string {
	return []string{"hl.dsp.focus({ window = 'address:" + addr + "' })"}
}
