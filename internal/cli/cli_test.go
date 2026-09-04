package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/6abe/kage/internal/cli"
	"github.com/6abe/kage/internal/host"
)

const monitorsJSON = `[
  {
    "id": 1,
    "name": "DP-1",
    "x": 0,
    "y": 0,
    "width": 5120,
    "height": 1440,
    "scale": 1.5,
    "focused": true
  }
]`

const clientsJSON = `[
  {
    "address": "0x123",
    "mapped": true,
    "hidden": false,
    "at": [100, 80],
    "size": [1400, 900],
    "workspace": {"id": 1, "name": "1"},
    "floating": false,
    "monitor": 1,
    "class": "google-chrome",
    "title": "GitHub",
    "focusHistoryID": 0
  },
  {
    "address": "0x456",
    "mapped": true,
    "at": [0, 0],
    "size": [800, 600],
    "workspace": {"id": 2, "name": "2"},
    "floating": true,
    "monitor": 1,
    "class": "kitty",
    "title": "term",
    "focusHistoryID": 1
  }
]`

const activeJSON = `{"address":"0x123","class":"google-chrome","title":"GitHub"}`

func okHost() *host.Fake {
	return &host.Fake{
		JSON: map[string][]byte{
			"monitors":     []byte(monitorsJSON),
			"clients":      []byte(clientsJSON),
			"activewindow": []byte(activeJSON),
		},
		Environ: map[string]string{
			"WAYLAND_DISPLAY":             "wayland-1",
			"HYPRLAND_INSTANCE_SIGNATURE": "sig",
		},
		Paths: map[string]string{
			"grim":    "/usr/bin/grim",
			"hyprctl": "/usr/bin/hyprctl",
			"wtype":   "/usr/bin/wtype",
			"ydotool": "/usr/bin/ydotool",
			"wl-copy": "/usr/bin/wl-copy",
		},
		Client: "grok",
		Disk:   []host.ClientStatus{{Name: "grok", Skill: false, MCP: false}},
	}
}

func execCLI(h host.Host, args ...string) (stdout, stderr string, code int) {
	var out, err bytes.Buffer
	code = cli.Run(h, args, &out, &err)
	return out.String(), err.String(), code
}

func TestWindowsJSONFields(t *testing.T) {
	out, errb, code := execCLI(okHost(), "windows")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	if errb != "" {
		t.Fatalf("stderr: %s", errb)
	}
	var payload struct {
		OK      bool `json:"ok"`
		Windows []struct {
			Address  string `json:"address"`
			Class    string `json:"class"`
			Title    string `json:"title"`
			Geometry struct {
				X, Y, Width, Height int
			} `json:"geometry"`
			Workspace int    `json:"workspace"`
			Monitor   string `json:"monitor"`
			Mapped    bool   `json:"mapped"`
			Floating  bool   `json:"floating"`
			Focus     bool   `json:"focus"`
		} `json:"windows"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if !payload.OK || len(payload.Windows) != 2 {
		t.Fatalf("payload: %+v", payload)
	}
	w := payload.Windows[0]
	if w.Address != "0x123" || w.Class != "google-chrome" || w.Title != "GitHub" {
		t.Fatalf("identity: %+v", w)
	}
	if w.Geometry.X != 100 || w.Geometry.Y != 80 || w.Geometry.Width != 1400 || w.Geometry.Height != 900 {
		t.Fatalf("geometry: %+v", w.Geometry)
	}
	if w.Workspace != 1 || w.Monitor != "DP-1" || !w.Mapped || w.Floating || !w.Focus {
		t.Fatalf("flags: %+v", w)
	}
	if !payload.Windows[1].Floating || payload.Windows[1].Focus {
		t.Fatalf("second: %+v", payload.Windows[1])
	}
}

func TestMonitorsJSONScale(t *testing.T) {
	out, errb, code := execCLI(okHost(), "monitors")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	var payload struct {
		OK       bool `json:"ok"`
		Monitors []struct {
			Name  string  `json:"name"`
			Scale float64 `json:"scale"`
			Width int     `json:"width"`
		} `json:"monitors"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if !payload.OK || len(payload.Monitors) != 1 {
		t.Fatalf("payload: %+v", payload)
	}
	if payload.Monitors[0].Name != "DP-1" || payload.Monitors[0].Scale != 1.5 || payload.Monitors[0].Width != 5120 {
		t.Fatalf("monitor: %+v", payload.Monitors[0])
	}
}

func TestDoctorOK(t *testing.T) {
	h := okHost()
	out, errb, code := execCLI(h, "doctor")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s stdout=%s", code, errb, out)
	}
	if !h.Probed {
		t.Fatal("doctor must run the grim capture probe")
	}
	var r map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"compositor", "wayland_display", "grim", "hyprctl", "wtype", "ydotool",
		"wl_copy", "capture", "input_backend", "default_client",
	} {
		if _, ok := r[key]; !ok {
			t.Fatalf("missing key %s in %s", key, out)
		}
	}
	if string(r["compositor"]) != `"hyprland"` {
		t.Fatalf("compositor %s", r["compositor"])
	}
	if string(r["wayland_display"]) != `"wayland-1"` {
		t.Fatalf("wayland %s", r["wayland_display"])
	}
	if string(r["input_backend"]) != `"wtype+ydotool"` {
		t.Fatalf("input %s", r["input_backend"])
	}
	if string(r["default_client"]) != `"grok"` {
		t.Fatalf("client %s", r["default_client"])
	}
}

func TestDoctorCaptureFailExit1(t *testing.T) {
	h := okHost()
	h.Probe = errString("grim exploded")
	out, _, code := execCLI(h, "doctor")
	if code != 1 {
		t.Fatalf("exit %d want 1 stdout=%s", code, out)
	}
	var r struct {
		OK      bool `json:"ok"`
		Capture struct {
			OK bool `json:"ok"`
		} `json:"capture"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	if r.OK || r.Capture.OK {
		t.Fatalf("expected capture fail: %s", out)
	}
	if !h.Probed {
		t.Fatal("probe should still run when grim is present")
	}
}

func TestDoctorInputMissingExit3(t *testing.T) {
	h := okHost()
	h.Paths = map[string]string{
		"grim":    "/usr/bin/grim",
		"hyprctl": "/usr/bin/hyprctl",
		"wl-copy": "/usr/bin/wl-copy",
	}
	out, _, code := execCLI(h, "doctor")
	if code != 3 {
		t.Fatalf("exit %d want 3 stdout=%s", code, out)
	}
	if !strings.Contains(out, `"input_backend":"none"`) {
		t.Fatalf("backend: %s", out)
	}
}

func TestDoctorMissingToolHint(t *testing.T) {
	h := okHost()
	h.Paths = map[string]string{
		"hyprctl": "/usr/bin/hyprctl",
		"wtype":   "/usr/bin/wtype",
		"wl-copy": "/usr/bin/wl-copy",
	}
	out, _, code := execCLI(h, "doctor")
	if code != 1 {
		t.Fatalf("exit %d want 1 (grim missing)", code)
	}
	if !strings.Contains(out, "omarchy pkg add grim") {
		t.Fatalf("missing grim hint: %s", out)
	}
	if h.Probed {
		t.Fatal("must not run grim when it is missing")
	}
	h2 := okHost()
	delete(h2.Paths, "ydotool")
	out2, _, code2 := execCLI(h2, "doctor")
	if code2 != 0 {
		t.Fatalf("wtype still present, want exit 0 got %d %s", code2, out2)
	}
	if !strings.Contains(out2, "omarchy pkg add ydotool") {
		t.Fatalf("missing ydotool hint: %s", out2)
	}
	if !strings.Contains(out2, "omarchy pkg add") {
		t.Fatal("expected omarchy pkg add")
	}
}

func TestHumanIsOnlyProse(t *testing.T) {
	out, errb, code := execCLI(okHost(), "windows", "--human")
	if code != 0 || errb != "" {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human windows is JSON: %s", out)
	}
	if !strings.Contains(out, "0x123") || !strings.Contains(out, "google-chrome") {
		t.Fatalf("human windows: %s", out)
	}
	out, _, code = execCLI(okHost(), "doctor", "--human")
	if code != 0 {
		t.Fatalf("doctor human exit %d", code)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human doctor is JSON: %s", out)
	}
	if !strings.Contains(out, "compositor") || !strings.Contains(out, "WAYLAND_DISPLAY") {
		t.Fatalf("human doctor: %s", out)
	}
	out, _, _ = execCLI(okHost(), "windows")
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("default must be JSON: %s", out)
	}
	out, _, code = execCLI(okHost(), "--human", "monitors")
	if code != 0 || strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human monitors: exit %d %s", code, out)
	}
	if !strings.Contains(out, "DP-1") || !strings.Contains(out, "scale=1.5") {
		t.Fatalf("human monitors: %s", out)
	}
}

func TestErrorsJSONOnStderr(t *testing.T) {
	h := okHost()
	h.Paths = map[string]string{} // no hyprctl
	out, errb, code := execCLI(h, "windows")
	if code == 0 {
		t.Fatal("want non-zero")
	}
	if out != "" {
		t.Fatalf("stdout should be empty: %s", out)
	}
	var f struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Hint  string `json:"hint"`
	}
	if err := json.Unmarshal([]byte(errb), &f); err != nil {
		t.Fatalf("stderr json: %v (%s)", err, errb)
	}
	if f.OK {
		t.Fatal("ok should be false")
	}
	if f.Error == "" {
		t.Fatal("error empty")
	}
	if !strings.Contains(f.Hint, "omarchy pkg add") {
		t.Fatalf("hint: %s", f.Hint)
	}
}

func TestDefaultClientGrok(t *testing.T) {
	h := okHost()
	h.Client = ""
	out, _, _ := execCLI(h, "doctor")
	if !strings.Contains(out, `"default_client":"grok"`) {
		t.Fatalf("default_client: %s", out)
	}
}

func TestHyprctlFailureHint(t *testing.T) {
	h := okHost()
	h.HyprctlErr = errString("connection refused")
	_, errb, code := execCLI(h, "monitors")
	if code == 0 {
		t.Fatal("want fail")
	}
	if !strings.Contains(errb, `"ok":false`) {
		t.Fatalf("stderr: %s", errb)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
