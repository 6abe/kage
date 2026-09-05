package cli_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/6abe/kage/internal/hypr"
)

func TestClickRequiresExactlyOneTarget(t *testing.T) {
	h := okHost()
	for _, args := range [][]string{
		{"click", "--yes"},
		{"click", "--yes", "--at", "1,2", "--on", "1"},
	} {
		out, errb, code := execCLI(h, args...)
		if code == 0 || out != "" {
			t.Fatalf("%v: exit %d stdout=%s", args, code, out)
		}
		if !strings.Contains(errb, "exactly one of --at") {
			t.Fatalf("%v: %s", args, errb)
		}
		if len(h.YdotoolCalls) != 0 {
			t.Fatalf("must not click without a target: %q", h.YdotoolCalls)
		}
	}
}

func TestClickAtYdotool(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "click", "--yes", "--at", "100,200")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	if len(h.Dispatch) != 0 {
		t.Fatalf("no window: dispatch %q", h.Dispatch)
	}
	want := [][]string{
		{"mousemove", "--absolute", "-x", "100", "-y", "200"},
		{"click", "0xC0"},
	}
	if !reflect.DeepEqual(h.YdotoolCalls, want) {
		t.Fatalf("ydotool %q", h.YdotoolCalls)
	}
	var payload struct {
		OK     bool   `json:"ok"`
		At     [2]int `json:"at"`
		Button string `json:"button"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.OK || payload.At != [2]int{100, 200} || payload.Button != "left" {
		t.Fatalf("payload %s", out)
	}
}

func TestClickButtonAndNegativeAt(t *testing.T) {
	h := okHost()
	_, errb, code := execCLI(h, "click", "--yes", "--at", "-10,20", "--button", "right")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	want := [][]string{
		{"mousemove", "--absolute", "-x", "-10", "-y", "20"},
		{"click", "0xC1"},
	}
	if !reflect.DeepEqual(h.YdotoolCalls, want) {
		t.Fatalf("%q", h.YdotoolCalls)
	}

	h = okHost()
	_, errb, code = execCLI(h, "click", "--yes", "--at=5,6", "--button=middle")
	if code != 0 {
		t.Fatalf("middle %d %s", code, errb)
	}
	if !reflect.DeepEqual(h.YdotoolCalls[1], []string{"click", "0xC2"}) {
		t.Fatalf("middle %q", h.YdotoolCalls)
	}

	h = okHost()
	_, errb, code = execCLI(h, "click", "--yes", "--at", "1,2", "--button", "forward")
	if code == 0 || !strings.Contains(errb, "left, right, or middle") {
		t.Fatalf("bad button: %d %s", code, errb)
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatal("invalid button must not click")
	}
}

func TestClickAtWindowFocuses(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "click", "--yes", "--at", "1,2", "--window", "0x456")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if len(h.Dispatch) != 1 || !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x456")) {
		t.Fatalf("dispatch %q", h.Dispatch)
	}
	if !strings.Contains(out, `"address":"0x456"`) {
		t.Fatalf("window %s", out)
	}
}

func TestClickOnSnapshotCenter(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("see %d %s", code, errb)
	}
	p := decodeSee(t, out)

	out, errb, code = execCLI(h, "click", "--yes", "--on", "1")
	if code != 0 {
		t.Fatalf("click %d %s", code, errb)
	}
	if len(h.Dispatch) != 1 || !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x123")) {
		t.Fatalf("focus %q", h.Dispatch)
	}
	// window 1 at 100,80 size 1400x900 → center 800,530
	want := [][]string{
		{"mousemove", "--absolute", "-x", "800", "-y", "530"},
		{"click", "0xC0"},
	}
	if !reflect.DeepEqual(h.YdotoolCalls, want) {
		t.Fatalf("ydotool %q", h.YdotoolCalls)
	}
	if !strings.Contains(out, `"on":1`) || !strings.Contains(out, p.SnapshotID) {
		t.Fatalf("payload %s", out)
	}

	h.YdotoolCalls = nil
	h.Dispatch = nil
	_, errb, code = execCLI(h, "click", "--yes", "--on", "2", "--snapshot", p.SnapshotID)
	if code != 0 {
		t.Fatalf("on 2 %d %s", code, errb)
	}
	if !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x456")) {
		t.Fatalf("focus 2 %q", h.Dispatch)
	}
	// kitty at 0,0 800x600 → 400,300
	if !reflect.DeepEqual(h.YdotoolCalls[0], []string{"mousemove", "--absolute", "-x", "400", "-y", "300"}) {
		t.Fatalf("center 2 %q", h.YdotoolCalls)
	}
}

func TestClickOnUsesLatestSee(t *testing.T) {
	h := seeHost(t)
	if _, errb, code := execCLI(h, "see"); code != 0 {
		t.Fatal(errb)
	}
	h.JSON["clients"] = []byte(`[
	  {"address":"0x999","mapped":true,"at":[50,50],"size":[20,20],"workspace":{"id":1},"monitor":1,"class":"x","title":"n","pid":1,"focusHistoryID":0}
	]`)
	h.JSON["activewindow"] = []byte(`{"address":"0x999"}`)
	if _, errb, code := execCLI(h, "see"); code != 0 {
		t.Fatal(errb)
	}
	h.YdotoolCalls = nil
	h.Dispatch = nil
	_, errb, code := execCLI(h, "click", "--yes", "--on", "1")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if !reflect.DeepEqual(h.Dispatch[0], wantFocus("0x999")) {
		t.Fatalf("latest focus %q", h.Dispatch)
	}
	if !reflect.DeepEqual(h.YdotoolCalls[0], []string{"mousemove", "--absolute", "-x", "60", "-y", "60"}) {
		t.Fatalf("latest center %q", h.YdotoolCalls)
	}
}

func TestClickOnMissingSnapshotAndID(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "click", "--yes", "--on", "1")
	if code == 0 || out != "" || !strings.Contains(errb, "no see snapshot") {
		t.Fatalf("no snap: %d %s", code, errb)
	}
	if len(h.YdotoolCalls) != 0 || len(h.Dispatch) != 0 {
		t.Fatal("must not click without snapshot")
	}

	h = seeHost(t)
	if _, errb, code = execCLI(h, "see"); code != 0 {
		t.Fatal(errb)
	}
	_, errb, code = execCLI(h, "click", "--yes", "--on", "99")
	if code == 0 || !strings.Contains(errb, "no window id 99") {
		t.Fatalf("missing id: %s", errb)
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatal("must not click unknown id")
	}

	_, errb, code = execCLI(h, "click", "--yes", "--on", "kitty")
	if code == 0 || !strings.Contains(errb, "annotated window id") {
		t.Fatalf("query: %s", errb)
	}

	_, errb, code = execCLI(h, "click", "--yes", "--snapshot", "kage_20260904_000000_abcd", "--at", "1,2")
	if code == 0 || !strings.Contains(errb, "--snapshot requires --on") {
		t.Fatalf("snapshot+at: %s", errb)
	}
}

func TestClickGate(t *testing.T) {
	h := okHost()
	_, errb, code := execCLI(h, "click", "--at", "1,2")
	if code == 0 {
		t.Fatal("click must refuse without allow")
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatal("refused click must not move")
	}
	for _, need := range []string{"--yes", "KAGE_ALLOW_INPUT=1", "allow_input = true"} {
		if !strings.Contains(errb, need) {
			t.Fatalf("gate error must name %q: %s", need, errb)
		}
	}

	h = okHost()
	h.Environ["KAGE_ALLOW_INPUT"] = "1"
	_, _, code = execCLI(h, "click", "--at", "1,2")
	if code != 0 {
		t.Fatalf("env %d", code)
	}

	h = okHost()
	h.Allow = true
	_, _, code = execCLI(h, "click", "--at", "3,4")
	if code != 0 {
		t.Fatalf("config %d", code)
	}
}

func TestClickMissingYdotoolHint(t *testing.T) {
	h := okHost()
	delete(h.Paths, "ydotool")
	out, errb, code := execCLI(h, "click", "--yes", "--at", "1,2")
	if code == 0 || out != "" {
		t.Fatalf("missing ydotool must fail: %d %s", code, out)
	}
	if !strings.Contains(errb, `"ok":false`) || !strings.Contains(errb, "ydotool not found") {
		t.Fatalf("stderr %s", errb)
	}
	if !strings.Contains(errb, "omarchy pkg add ydotool") || !strings.Contains(errb, "ydotool.service") {
		t.Fatalf("hint %s", errb)
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatal("must not fake a click")
	}

	h = okHost()
	delete(h.Paths, "ydotoold")
	_, errb, code = execCLI(h, "click", "--yes", "--at", "1,2")
	if code == 0 || !strings.Contains(errb, "ydotoold not found") {
		t.Fatalf("missing daemon: %s", errb)
	}
	if !strings.Contains(errb, "omarchy pkg add ydotool") {
		t.Fatalf("hint %s", errb)
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatal("must not fake a click")
	}

	h = okHost()
	h.YdotoolErr = errString("ydotool not running")
	_, errb, code = execCLI(h, "click", "--yes", "--at", "1,2")
	if code == 0 || !strings.Contains(errb, "ydotool not running") {
		t.Fatalf("daemon down: %s", errb)
	}
	if !strings.Contains(errb, "systemctl --user start ydotool.service") {
		t.Fatalf("hint %s", errb)
	}
}

func TestClickNoXdotool(t *testing.T) {
	h := okHost()
	h.Paths["xdotool"] = "/usr/bin/xdotool"
	h.Lookups = nil
	_, errb, code := execCLI(h, "click", "--yes", "--at", "1,2")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	for _, name := range h.Lookups {
		if name == "xdotool" {
			t.Fatal("xdotool must not be used")
		}
	}
}

func TestHotkeyPrefersSendShortcut(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "hotkey", "--yes", "CTRL+C")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	want := []string{hypr.SendShortcutDispatch("CTRL", "C")}
	if len(h.Dispatch) != 1 || !reflect.DeepEqual(h.Dispatch[0], want) {
		t.Fatalf("dispatch %q want %q", h.Dispatch, want)
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatalf("must not ydotool when send_shortcut works: %q", h.YdotoolCalls)
	}
	if !strings.Contains(out, `"backend":"sendshortcut"`) || !strings.Contains(out, `"hotkey":"CTRL+C"`) {
		t.Fatalf("payload %s", out)
	}
}

func TestHotkeyFallsBackToYdotool(t *testing.T) {
	h := okHost()
	h.SendShortcutErr = errString("unknown dispatcher")
	out, errb, code := execCLI(h, "hotkey", "--yes", "CTRL+C")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if len(h.YdotoolCalls) != 1 || !reflect.DeepEqual(h.YdotoolCalls[0], []string{"key", "29:1", "46:1", "46:0", "29:0"}) {
		t.Fatalf("ydotool %q", h.YdotoolCalls)
	}
	if !strings.Contains(out, `"backend":"ydotool"`) {
		t.Fatalf("payload %s", out)
	}

	h = okHost()
	h.SendShortcutErr = errString("unknown dispatcher")
	delete(h.Paths, "ydotool")
	_, errb, code = execCLI(h, "hotkey", "--yes", "ALT+F4")
	if code == 0 || !strings.Contains(errb, "ydotool not found") {
		t.Fatalf("fallback missing ydotool: %s", errb)
	}
}

func TestHotkeyGateAndNoXdotool(t *testing.T) {
	h := okHost()
	_, errb, code := execCLI(h, "hotkey", "SUPER+Q")
	if code == 0 || !strings.Contains(errb, "--yes") {
		t.Fatalf("gate %d %s", code, errb)
	}
	if len(h.Dispatch) != 0 || len(h.YdotoolCalls) != 0 {
		t.Fatal("refused hotkey must not send")
	}

	h = okHost()
	h.Paths["xdotool"] = "/usr/bin/xdotool"
	h.Lookups = nil
	_, _, code = execCLI(h, "hotkey", "--yes", "SUPER+Q")
	if code != 0 {
		t.Fatal(code)
	}
	for _, name := range h.Lookups {
		if name == "xdotool" {
			t.Fatal("xdotool must not be used")
		}
	}
}

func TestDispatchJSONWrap(t *testing.T) {
	h := okHost()
	lua := "hl.dsp.window.close()"
	out, errb, code := execCLI(h, "dispatch", lua)
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if len(h.Dispatch) != 1 || !reflect.DeepEqual(h.Dispatch[0], []string{lua}) {
		t.Fatalf("dispatch %q", h.Dispatch)
	}
	if len(h.YdotoolCalls) != 0 || len(h.WtypeCalls) != 0 {
		t.Fatal("dispatch is not click or type")
	}
	var payload struct {
		OK   bool     `json:"ok"`
		Args []string `json:"args"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.OK || !reflect.DeepEqual(payload.Args, []string{lua}) {
		t.Fatalf("payload %s", out)
	}

	h = okHost()
	_, _, code = execCLI(h, "dispatch", "hl.dsp.workspace.move({ workspace = '2' })")
	if code != 0 {
		t.Fatal(code)
	}

	h = okHost()
	_, errb, code = execCLI(h, "dispatch")
	if code == 0 || !strings.Contains(errb, "kage dispatch") {
		t.Fatalf("empty: %s", errb)
	}

	h = okHost()
	_, errb, code = execCLI(h, "click", "--yes", "--at", "1,2")
	if code != 0 {
		t.Fatal(errb)
	}
	if len(h.Dispatch) != 0 {
		t.Fatalf("click --at must not hyprctl dispatch: %q", h.Dispatch)
	}
}

func TestDispatchUngated(t *testing.T) {
	h := okHost()
	_, errb, code := execCLI(h, "dispatch", "hl.dsp.window.fullscreen()")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if strings.Contains(errb, "input not allowed") {
		t.Fatal("dispatch must not use the input gate")
	}
}

func TestClickHotkeyHuman(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "click", "--yes", "--human", "--at", "1,2")
	if code != 0 || errb != "" {
		t.Fatalf("%d %s", code, errb)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human click is JSON: %s", out)
	}
	out, _, code = execCLI(okHost(), "hotkey", "--yes", "--human", "CTRL+C")
	if code != 0 || strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human hotkey: %s", out)
	}
	out, _, code = execCLI(okHost(), "dispatch", "--human", "hl.dsp.window.close()")
	if code != 0 || strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human dispatch: %s", out)
	}
}

func TestClickMissingFlagValues(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"click", "--yes", "--at"}, "--at requires X,Y"},
		{[]string{"click", "--yes", "--on"}, "--on requires an annotated window id"},
		{[]string{"click", "--yes", "--button"}, "left, right, or middle"},
		{[]string{"click", "--yes", "--on", "1", "--snapshot"}, "snapshot id"},
		{[]string{"windows", "--at", "1,2"}, "unknown flag: --at"},
		{[]string{"click", "--yes", "--at", "foo,bar"}, "invalid --at"},
	} {
		h := okHost()
		out, errb, code := execCLI(h, tc.args...)
		if code == 0 || out != "" {
			t.Fatalf("%v: exit %d stdout=%s", tc.args, code, out)
		}
		if !strings.Contains(errb, tc.want) {
			t.Fatalf("%v: %s", tc.args, errb)
		}
		if len(h.YdotoolCalls) != 0 || len(h.Dispatch) != 0 {
			t.Fatalf("%v: skipped input must not run: ydotool=%q dispatch=%q", tc.args, h.YdotoolCalls, h.Dispatch)
		}
	}
}

func TestClickOnMissingHyprctl(t *testing.T) {
	h := seeHost(t)
	if _, errb, code := execCLI(h, "see"); code != 0 {
		t.Fatal(errb)
	}
	delete(h.Paths, "hyprctl")
	h.YdotoolCalls = nil
	h.Dispatch = nil
	out, errb, code := execCLI(h, "click", "--yes", "--on", "1")
	if code == 0 || out != "" {
		t.Fatalf("exit %d stdout=%s", code, out)
	}
	if !strings.Contains(errb, "hyprctl not found") || !strings.Contains(errb, "omarchy pkg add hyprland") {
		t.Fatalf("hint %s", errb)
	}
	if len(h.YdotoolCalls) != 0 || len(h.Dispatch) != 0 {
		t.Fatalf("must not click: ydotool=%q dispatch=%q", h.YdotoolCalls, h.Dispatch)
	}
}

func TestHotkeyDoesNotLogChord(t *testing.T) {
	h := okHost()
	secret := "CTRL+SHIFT+U"
	_, errb, code := execCLI(h, "hotkey", "--yes", secret)
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	for _, line := range h.Logs {
		if strings.Contains(line, secret) || strings.Contains(line, "SHIFT+U") {
			t.Fatalf("hotkey in log: %q", line)
		}
	}
}
