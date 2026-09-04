package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/6abe/kage/internal/cli"
	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
	"github.com/6abe/kage/internal/see"
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
    "pid": 4321,
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
    "pid": 99,
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
	if len(h.GrimArgs) < 2 || h.GrimArgs[0] != "-g" || h.GrimArgs[1] != host.GrimRegion {
		t.Fatalf("grim argv want -g %q, got %q", host.GrimRegion, h.GrimArgs)
	}
	if strings.Contains(strings.Join(h.GrimArgs, " "), "0,0,1,1") {
		t.Fatalf("comma geometry is invalid for grim: %q", h.GrimArgs)
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
	if len(h.GrimArgs) == 0 {
		t.Fatal("probe should still run when grim is present")
	}
	if strings.Contains(out, "omarchy pkg add grim") {
		t.Fatalf("must not hint pkg add grim when grim is present: %s", out)
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
	if h.GrimArgs != nil {
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

func seeHost(t *testing.T) *host.Fake {
	t.Helper()
	h := okHost()
	h.Environ["XDG_RUNTIME_DIR"] = t.TempDir()
	return h
}

type seePayload struct {
	OK         bool   `json:"ok"`
	SnapshotID string `json:"snapshot_id"`
	Path       string `json:"path"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Monitor    struct {
		Name   string  `json:"name"`
		X      int     `json:"x"`
		Y      int     `json:"y"`
		Width  int     `json:"width"`
		Height int     `json:"height"`
		Scale  float64 `json:"scale"`
	} `json:"monitor"`
	Focused *struct {
		Address   string `json:"address"`
		Class     string `json:"class"`
		Title     string `json:"title"`
		At        [2]int `json:"at"`
		Size      [2]int `json:"size"`
		Workspace int    `json:"workspace"`
		PID       int    `json:"pid"`
	} `json:"focused"`
	Windows []struct {
		ID        int    `json:"id"`
		Address   string `json:"address"`
		Class     string `json:"class"`
		Title     string `json:"title"`
		At        [2]int `json:"at"`
		Size      [2]int `json:"size"`
		Workspace int    `json:"workspace"`
		Monitor   string `json:"monitor"`
		Floating  bool   `json:"floating"`
		Mapped    bool   `json:"mapped"`
		Focus     bool   `json:"focus"`
	} `json:"windows"`
	CoordinateSpace string `json:"coordinate_space"`
}

var snapshotIDRe = regexp.MustCompile(`^kage_\d{8}_\d{6}_[0-9a-f]{4}$`)

func decodeSee(t *testing.T, out string) seePayload {
	t.Helper()
	var p seePayload
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	return p
}

func assertPNGFile(t *testing.T, path string) (w, h int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	var hdr [8]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		t.Fatal(err)
	}
	if string(hdr[:]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("not a PNG: %q", hdr)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height
}

func TestSeeJSONAndPNG(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	if errb != "" {
		t.Fatalf("stderr: %s", errb)
	}
	p := decodeSee(t, out)
	if !p.OK {
		t.Fatal("ok false")
	}
	if !snapshotIDRe.MatchString(p.SnapshotID) {
		t.Fatalf("snapshot_id %q", p.SnapshotID)
	}
	if p.CoordinateSpace != "global_compositor_pixels" {
		t.Fatalf("coordinate_space %q", p.CoordinateSpace)
	}
	if p.Monitor.Name != "DP-1" || p.Monitor.Width != 5120 || p.Monitor.Height != 1440 || p.Monitor.Scale != 1.5 {
		t.Fatalf("monitor %+v", p.Monitor)
	}
	if p.Focused == nil || p.Focused.Address != "0x123" || p.Focused.Class != "google-chrome" || p.Focused.PID != 4321 {
		t.Fatalf("focused %+v", p.Focused)
	}
	if p.Focused.At != [2]int{100, 80} || p.Focused.Size != [2]int{1400, 900} || p.Focused.Workspace != 1 {
		t.Fatalf("focused geom %+v", p.Focused)
	}
	if len(p.Windows) != 2 || p.Windows[0].ID != 1 || p.Windows[1].ID != 2 {
		t.Fatalf("windows %+v", p.Windows)
	}
	if p.Windows[0].At != [2]int{100, 80} || p.Windows[0].Size != [2]int{1400, 900} || !p.Windows[0].Focus {
		t.Fatalf("window0 %+v", p.Windows[0])
	}
	if !p.Windows[1].Floating || p.Windows[1].Focus {
		t.Fatalf("window1 %+v", p.Windows[1])
	}
	if len(h.GrimArgs) != 3 || h.GrimArgs[0] != "-o" || h.GrimArgs[1] != "DP-1" {
		t.Fatalf("grim argv %q", h.GrimArgs)
	}
	if h.GrimArgs[2] != p.Path {
		t.Fatalf("grim out %q path %q", h.GrimArgs[2], p.Path)
	}
	runtime := h.Environ["XDG_RUNTIME_DIR"]
	dir := filepath.Join(runtime, "kage")
	if filepath.Dir(p.Path) != dir {
		t.Fatalf("path %s not under %s", p.Path, dir)
	}
	if !strings.HasSuffix(p.Path, p.SnapshotID+".png") {
		t.Fatalf("filename %s id %s", p.Path, p.SnapshotID)
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %o", st.Mode().Perm())
	}
	w, ht := assertPNGFile(t, p.Path)
	if p.Width != w || p.Height != ht {
		t.Fatalf("json %dx%d file %dx%d", p.Width, p.Height, w, ht)
	}
	if strings.Contains(out, "base64") || strings.Contains(out, "iVBORw0KGgo") {
		t.Fatalf("JSON must not carry image bytes: %s", out)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"ok", "snapshot_id", "path", "width", "height", "monitor", "focused", "windows", "coordinate_space",
	} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("missing key %s in %s", key, out)
		}
	}
	for _, banned := range []string{"image", "bytes", "base64", "png_b64", "data"} {
		if _, ok := raw[banned]; ok {
			t.Fatalf("banned key %s in %s", banned, out)
		}
	}
}

func TestSeePathFlag(t *testing.T) {
	h := seeHost(t)
	path := filepath.Join(t.TempDir(), "nested", "shot.png")
	out, errb, code := execCLI(h, "see", "--path", path)
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Path != abs {
		t.Fatalf("path %s want %s", p.Path, abs)
	}
	assertPNGFile(t, abs)
	if len(h.GrimArgs) != 3 || h.GrimArgs[0] != "-o" || h.GrimArgs[2] != abs {
		t.Fatalf("grim argv %q", h.GrimArgs)
	}
	out, _, code = execCLI(seeHost(t), "see", "--path="+path+".eq.png")
	if code != 0 {
		t.Fatalf("equals form exit %d", code)
	}
	p = decodeSee(t, out)
	if !strings.HasSuffix(p.Path, "shot.png.eq.png") {
		t.Fatalf("equals path %s", p.Path)
	}
	assertPNGFile(t, p.Path)
}

func TestSeeDefaultFocusedMonitor(t *testing.T) {
	h := seeHost(t)
	h.JSON["monitors"] = []byte(`[
		{"id":0,"name":"HDMI-A-1","x":5120,"y":0,"width":1920,"height":1080,"scale":1,"focused":false},
		{"id":1,"name":"DP-1","x":0,"y":0,"width":5120,"height":1440,"scale":1.5,"focused":true}
	]`)
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Monitor.Name != "DP-1" {
		t.Fatalf("captured %s", p.Monitor.Name)
	}
	if len(h.GrimArgs) < 2 || h.GrimArgs[0] != "-o" || h.GrimArgs[1] != "DP-1" {
		t.Fatalf("grim argv %q", h.GrimArgs)
	}
	joined := strings.Join(h.GrimArgs, " ")
	if strings.Contains(joined, "0,0,1,1") || strings.Contains(joined, "slurp") {
		t.Fatalf("invalid grim geometry: %q", h.GrimArgs)
	}
}

func TestSeeFallbackDir(t *testing.T) {
	h := seeHost(t)
	delete(h.Environ, "XDG_RUNTIME_DIR")
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	wantDir := filepath.Join(os.TempDir(), fmt.Sprintf("kage-%d", os.Getuid()))
	if filepath.Dir(p.Path) != wantDir {
		t.Fatalf("path %s want under %s", p.Path, wantDir)
	}
	assertPNGFile(t, p.Path)
	st, err := os.Stat(wantDir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o700 {
		t.Fatalf("fallback dir mode %o", st.Mode().Perm())
	}
	t.Cleanup(func() {
		_ = os.Remove(p.Path)
		_ = os.Remove(filepath.Join(wantDir, p.SnapshotID+".json"))
	})
}

func TestSeeHuman(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "see", "--human")
	if code != 0 || errb != "" {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("human see is JSON: %s", out)
	}
	if !strings.Contains(out, "DP-1") {
		t.Fatalf("human see: %s", out)
	}
	if len(h.GrimArgs) != 3 || h.GrimArgs[0] != "-o" {
		t.Fatalf("human still captures: %q", h.GrimArgs)
	}
	assertPNGFile(t, h.GrimArgs[2])
}

func TestSeeFocusedNull(t *testing.T) {
	h := seeHost(t)
	h.JSON["activewindow"] = []byte(`{"address":"0xdead"}`)
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Focused != nil {
		t.Fatalf("focused %+v", p.Focused)
	}
	if !strings.Contains(out, `"focused":null`) {
		t.Fatalf("want focused null: %s", out)
	}
	assertPNGFile(t, p.Path)
}

func TestSeeErrors(t *testing.T) {
	h := seeHost(t)
	delete(h.Paths, "grim")
	out, errb, code := execCLI(h, "see")
	if code == 0 || out != "" {
		t.Fatalf("missing grim: exit %d stdout=%s", code, out)
	}
	if !strings.Contains(errb, `"ok":false`) || !strings.Contains(errb, "omarchy pkg add grim") {
		t.Fatalf("grim hint: %s", errb)
	}

	h = seeHost(t)
	h.Probe = errString("grim exploded")
	out, errb, code = execCLI(h, "see")
	if code == 0 || out != "" {
		t.Fatalf("grim fail: exit %d stdout=%s", code, out)
	}
	if !strings.Contains(errb, "grim exploded") {
		t.Fatalf("stderr: %s", errb)
	}

	h = seeHost(t)
	h.JSON["monitors"] = []byte(`[{"id":1,"name":"DP-1","x":0,"y":0,"width":10,"height":10,"scale":1,"focused":false}]`)
	_, errb, code = execCLI(h, "see")
	if code == 0 || !strings.Contains(errb, "no focused monitor") {
		t.Fatalf("no focus: exit %d %s", code, errb)
	}

	_, errb, code = execCLI(seeHost(t), "see", "--path")
	if code == 0 || !strings.Contains(errb, "flag --path requires a file path") {
		t.Fatalf("--path missing: %s", errb)
	}
	_, errb, code = execCLI(okHost(), "windows", "--path", "/tmp/x.png")
	if code == 0 || !strings.Contains(errb, "unknown flag: --path") {
		t.Fatalf("windows --path: %s", errb)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func failPayload(t *testing.T, errb string) (f struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Hint    string `json:"hint"`
	Matches []struct {
		Address string `json:"address"`
		Class   string `json:"class"`
		Title   string `json:"title"`
	} `json:"matches"`
}) {
	t.Helper()
	if err := json.Unmarshal([]byte(errb), &f); err != nil {
		t.Fatalf("stderr json: %v (%s)", err, errb)
	}
	if f.OK {
		t.Fatalf("ok true: %s", errb)
	}
	return f
}

func rgbAt(t *testing.T, img image.Image, x, y int) (r, g, b uint8) {
	t.Helper()
	rr, gg, bb, _ := img.At(x, y).RGBA()
	return uint8(rr >> 8), uint8(gg >> 8), uint8(bb >> 8)
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func TestSeeMonitorFlag(t *testing.T) {
	h := seeHost(t)
	h.JSON["monitors"] = []byte(`[
		{"id":0,"name":"HDMI-A-1","x":5120,"y":0,"width":1920,"height":1080,"scale":1,"focused":false},
		{"id":1,"name":"DP-1","x":0,"y":0,"width":5120,"height":1440,"scale":1.5,"focused":true}
	]`)
	out, errb, code := execCLI(h, "see", "--monitor", "HDMI-A-1")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Monitor.Name != "HDMI-A-1" || p.Monitor.Width != 1920 {
		t.Fatalf("monitor %+v", p.Monitor)
	}
	if len(h.GrimArgs) < 2 || h.GrimArgs[0] != "-o" || h.GrimArgs[1] != "HDMI-A-1" {
		t.Fatalf("grim argv %q", h.GrimArgs)
	}
	assertPNGFile(t, p.Path)

	out, errb, code = execCLI(seeHost(t), "see", "--monitor=dp-1")
	if code != 0 {
		t.Fatalf("fold exit %d %s", code, errb)
	}
	p = decodeSee(t, out)
	if p.Monitor.Name != "DP-1" {
		t.Fatalf("fold %s", p.Monitor.Name)
	}

	_, errb, code = execCLI(seeHost(t), "see", "--monitor", "NOPE")
	if code == 0 || strings.TrimSpace(errb) == "" {
		t.Fatal("missing monitor must fail")
	}
	f := failPayload(t, errb)
	if !strings.Contains(f.Error, "no monitor named") || f.Hint != "kage monitors" {
		t.Fatalf("missing monitor: %s", errb)
	}
}

func TestSeeWindowFlag(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "see", "--window", "0x123")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Windows[0].ID != 1 || p.Windows[0].Address != "0x123" {
		t.Fatalf("windows %+v", p.Windows)
	}
	want := hypr.GrimGeom(100, 80, 1400, 900)
	if len(h.GrimArgs) != 3 || h.GrimArgs[0] != "-g" || h.GrimArgs[1] != want {
		t.Fatalf("grim argv %q want -g %q", h.GrimArgs, want)
	}
	if strings.Contains(strings.Join(h.GrimArgs, " "), "100,80,1400,900") {
		t.Fatalf("comma geometry is invalid: %q", h.GrimArgs)
	}
	w, ht := assertPNGFile(t, p.Path)
	if w != 1400 || ht != 900 {
		t.Fatalf("png %dx%d (window geom, under max-width)", w, ht)
	}

	h = seeHost(t)
	out, errb, code = execCLI(h, "see", "--window", "KITTY")
	if code != 0 {
		t.Fatalf("class: exit %d %s", code, errb)
	}
	p = decodeSee(t, out)
	if len(h.GrimArgs) < 2 || h.GrimArgs[1] != hypr.GrimGeom(0, 0, 800, 600) {
		t.Fatalf("class grim %q", h.GrimArgs)
	}
	if p.Windows[1].Class != "kitty" {
		t.Fatalf("still lists all windows: %+v", p.Windows)
	}

	h = seeHost(t)
	_, errb, code = execCLI(h, "see", "--window=Git")
	if code != 0 {
		t.Fatalf("title: exit %d %s", code, errb)
	}
	if h.GrimArgs[1] != hypr.GrimGeom(100, 80, 1400, 900) {
		t.Fatalf("title grim %q", h.GrimArgs)
	}
}

func TestSeeWindowNoMatchAndAmbiguous(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "see", "--window", "no-such-client")
	if code == 0 || out != "" {
		t.Fatalf("no match must fail, stdout=%s", out)
	}
	if h.GrimArgs != nil {
		t.Fatalf("no match must not capture: %q", h.GrimArgs)
	}
	f := failPayload(t, errb)
	if !strings.Contains(f.Error, "no window matches") || f.Hint != "kage windows" {
		t.Fatalf("no match: %s", errb)
	}

	h = seeHost(t)
	h.JSON["clients"] = []byte(`[
		{"address":"0x1","mapped":true,"at":[0,0],"size":[10,10],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"a","pid":1,"focusHistoryID":0},
		{"address":"0x2","mapped":true,"at":[10,0],"size":[10,10],"workspace":{"id":1},"monitor":1,"class":"kitty","title":"b","pid":2,"focusHistoryID":1}
	]`)
	out, errb, code = execCLI(h, "see", "--window", "kitty")
	if code != 2 {
		t.Fatalf("ambiguous exit %d want 2 stdout=%s stderr=%s", code, out, errb)
	}
	if out != "" {
		t.Fatalf("stdout should be empty: %s", out)
	}
	if h.GrimArgs != nil {
		t.Fatalf("ambiguous must not capture: %q", h.GrimArgs)
	}
	f = failPayload(t, errb)
	if !strings.Contains(f.Error, "ambiguous") {
		t.Fatalf("error: %s", errb)
	}
	if len(f.Matches) != 2 {
		t.Fatalf("matches %+v", f.Matches)
	}
	got := f.Matches[0].Address + "," + f.Matches[1].Address
	if got != "0x1,0x2" {
		t.Fatalf("match addresses %s", got)
	}
}

func TestSeeAllUnion(t *testing.T) {
	h := seeHost(t)
	h.JSON["monitors"] = []byte(`[
		{"id":0,"name":"DP-1","x":0,"y":0,"width":100,"height":50,"scale":1,"focused":true},
		{"id":1,"name":"HDMI-A-1","x":100,"y":-10,"width":80,"height":60,"scale":1,"focused":false}
	]`)
	out, errb, code := execCLI(h, "see", "--all")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Monitor.Name != "all" || p.Monitor.X != 0 || p.Monitor.Y != -10 || p.Monitor.Width != 180 || p.Monitor.Height != 60 {
		t.Fatalf("union monitor %+v", p.Monitor)
	}
	want := hypr.GrimGeom(0, -10, 180, 60)
	if len(h.GrimArgs) != 3 || h.GrimArgs[0] != "-g" || h.GrimArgs[1] != want {
		t.Fatalf("grim argv %q want -g %q", h.GrimArgs, want)
	}
	w, ht := assertPNGFile(t, p.Path)
	if w != 180 || ht != 60 {
		t.Fatalf("png %dx%d", w, ht)
	}
	if p.Path == "" || strings.Contains(out, "base64") {
		t.Fatalf("one path, no bytes: %s", out)
	}
}

func TestSeeMutuallyExclusiveTargets(t *testing.T) {
	for _, args := range [][]string{
		{"see", "--all", "--monitor", "DP-1"},
		{"see", "--all", "--window", "0x123"},
		{"see", "--monitor", "DP-1", "--window", "0x123"},
	} {
		_, errb, code := execCLI(seeHost(t), args...)
		if code == 0 || !strings.Contains(errb, "mutually exclusive") {
			t.Fatalf("%v: %s", args, errb)
		}
	}
}

func TestSeeMaxWidth(t *testing.T) {
	h := seeHost(t)
	h.ImageSize = image.Pt(2000, 1000)
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Width != 1920 || p.Height != 960 {
		t.Fatalf("default long-edge 1920, got %dx%d", p.Width, p.Height)
	}
	w, ht := assertPNGFile(t, p.Path)
	if w != 1920 || ht != 960 {
		t.Fatalf("file %dx%d", w, ht)
	}

	h = seeHost(t)
	h.ImageSize = image.Pt(400, 200)
	out, errb, code = execCLI(h, "see", "--max-width", "200")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	p = decodeSee(t, out)
	if p.Width != 200 || p.Height != 100 {
		t.Fatalf("max-width 200: %dx%d", p.Width, p.Height)
	}
	w, ht = assertPNGFile(t, p.Path)
	if w != 200 || ht != 100 {
		t.Fatalf("file %dx%d", w, ht)
	}

	h = seeHost(t)
	h.ImageSize = image.Pt(100, 400)
	out, _, code = execCLI(h, "see", "--max-width=100")
	if code != 0 {
		t.Fatal(code)
	}
	p = decodeSee(t, out)
	if p.Width != 25 || p.Height != 100 {
		t.Fatalf("portrait long edge: %dx%d", p.Width, p.Height)
	}

	_, errb, code = execCLI(seeHost(t), "see", "--max-width", "0")
	if code == 0 || !strings.Contains(errb, "positive integer") {
		t.Fatalf("max-width 0: %s", errb)
	}
	_, errb, code = execCLI(seeHost(t), "see", "--max-width", "nope")
	if code == 0 || !strings.Contains(errb, "positive integer") {
		t.Fatalf("max-width nope: %s", errb)
	}
}

func TestSeeAnnotateDrawsIDs(t *testing.T) {
	h := seeHost(t)
	h.ImageSize = image.Pt(400, 240)
	h.JSON["monitors"] = []byte(`[{"id":1,"name":"DP-1","x":0,"y":0,"width":400,"height":240,"scale":1,"focused":true}]`)
	h.JSON["clients"] = []byte(`[
		{"address":"0x1","mapped":true,"at":[40,40],"size":[100,80],"workspace":{"id":1},"monitor":1,"class":"a","title":"one","pid":1,"focusHistoryID":0},
		{"address":"0x2","mapped":false,"at":[240,40],"size":[100,80],"workspace":{"id":1},"monitor":1,"class":"b","title":"two","pid":2,"focusHistoryID":1},
		{"address":"0x3","mapped":true,"at":[40,140],"size":[80,60],"workspace":{"id":1},"monitor":1,"class":"c","title":"three","pid":3,"focusHistoryID":2}
	]`)
	h.JSON["activewindow"] = []byte(`{"address":"0x1"}`)
	out, errb, code := execCLI(h, "see", "--annotate", "--max-width", "200")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeSee(t, out)
	if p.Width != 200 || p.Height != 120 {
		t.Fatalf("downscaled %dx%d", p.Width, p.Height)
	}
	if len(p.Windows) != 3 || p.Windows[0].ID != 1 || p.Windows[1].ID != 2 || p.Windows[2].ID != 3 {
		t.Fatalf("ids %+v", p.Windows)
	}
	if p.Windows[0].Mapped == false || p.Windows[1].Mapped != false {
		t.Fatalf("mapped flags %+v", p.Windows)
	}
	if p.Windows[0].At != [2]int{40, 40} || p.Windows[2].ID != 3 {
		t.Fatalf("compositor coords stay unscaled: %+v", p.Windows)
	}
	img := decodePNG(t, p.Path)
	r, g, b := rgbAt(t, img, 20, 20)
	if r < 200 || g > 40 || b > 40 {
		t.Fatalf("window 1 box not red at scaled 20,20: %d,%d,%d", r, g, b)
	}
	r, g, b = rgbAt(t, img, 20, 70)
	if r < 200 || g > 40 || b > 40 {
		t.Fatalf("window 3 box not red at scaled 20,70: %d,%d,%d", r, g, b)
	}
	r, g, b = rgbAt(t, img, 120, 20)
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("unmapped window 2 must not be boxed: %d,%d,%d", r, g, b)
	}
	if !regionHasWhite(img, 22, 22, 24, 20) {
		t.Fatal("window 1 id glyph missing (want white pixels in label)")
	}
	if !regionHasWhite(img, 22, 72, 24, 20) {
		t.Fatal("window 3 id glyph missing")
	}
}

func regionHasWhite(img image.Image, x, y, w, h int) bool {
	b := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !image.Pt(xx, yy).In(b) {
				continue
			}
			rr, gg, bb, _ := img.At(xx, yy).RGBA()
			if rr>>8 > 200 && gg>>8 > 200 && bb>>8 > 200 {
				return true
			}
		}
	}
	return false
}

func TestSeeSnapshotPersistAndLoad(t *testing.T) {
	h := seeHost(t)
	out, errb, code := execCLI(h, "see")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	p := decodeSee(t, out)
	meta := filepath.Join(filepath.Dir(p.Path), p.SnapshotID+".json")
	if _, err := os.Stat(meta); err != nil {
		t.Fatalf("sidecar: %v", err)
	}
	loaded, err := see.Load(h, p.SnapshotID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SnapshotID != p.SnapshotID || len(loaded.Windows) != 2 || loaded.Windows[0].ID != 1 {
		t.Fatalf("loaded %+v", loaded)
	}
	if loaded.Windows[0].Address != "0x123" || loaded.Windows[1].ID != 2 {
		t.Fatalf("ids must round-trip: %+v", loaded.Windows)
	}

	path := filepath.Join(t.TempDir(), "custom.png")
	h = seeHost(t)
	out, errb, code = execCLI(h, "see", "--path", path)
	if code != 0 {
		t.Fatalf("path exit %d %s", code, errb)
	}
	p = decodeSee(t, out)
	side := filepath.Join(filepath.Dir(p.Path), p.SnapshotID+".json")
	if _, err := os.Stat(side); err != nil {
		t.Fatalf("sidecar next to png: %v", err)
	}
	if _, err := see.Load(h, p.SnapshotID); err != nil {
		t.Fatalf("lookup store: %v", err)
	}
	_, err = see.Load(h, "../etc/passwd")
	if err == nil {
		t.Fatal("path traversal must fail")
	}
	if _, err = see.Load(h, "kage_20000101_000000_abcd"); err == nil {
		t.Fatal("missing snapshot must fail")
	}
}

func TestSeeMissingFlagValues(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"see", "--window"}, "--window requires a value"},
		{[]string{"see", "--monitor"}, "flag --monitor requires a name"},
		{[]string{"see", "--max-width"}, "flag --max-width requires a positive integer"},
	} {
		out, errb, code := execCLI(seeHost(t), tc.args...)
		if code == 0 || out != "" {
			t.Fatalf("%v: exit %d stdout=%s", tc.args, code, out)
		}
		if !strings.Contains(errb, tc.want) {
			t.Fatalf("%v: %s", tc.args, errb)
		}
	}
}

func TestSeeWindowEmptyGeometry(t *testing.T) {
	h := seeHost(t)
	h.JSON["clients"] = []byte(`[
		{"address":"0x123","mapped":true,"at":[10,10],"size":[0,0],"workspace":{"id":1},"monitor":1,"class":"x","title":"z","pid":1,"focusHistoryID":0}
	]`)
	out, errb, code := execCLI(h, "see", "--window", "0x123")
	if code == 0 || out != "" {
		t.Fatalf("exit %d stdout=%s", code, out)
	}
	if h.GrimArgs != nil {
		t.Fatalf("must not grim empty geom: %q", h.GrimArgs)
	}
	if !strings.Contains(errb, "empty geometry") {
		t.Fatalf("stderr: %s", errb)
	}
}

func TestSeeFlagsRejectedOnOtherCommands(t *testing.T) {
	_, errb, code := execCLI(okHost(), "windows", "--annotate")
	if code == 0 || !strings.Contains(errb, "unknown flag: --annotate") {
		t.Fatalf("windows --annotate: %s", errb)
	}
	_, errb, code = execCLI(okHost(), "monitors", "--all")
	if code == 0 || !strings.Contains(errb, "unknown flag: --all") {
		t.Fatalf("monitors --all: %s", errb)
	}
}
