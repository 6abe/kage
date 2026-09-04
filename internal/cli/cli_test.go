package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	t.Cleanup(func() { _ = os.Remove(p.Path) })
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

	_, errb, code = execCLI(seeHost(t), "see", "--annotate")
	if code == 0 || !strings.Contains(errb, "unknown flag: --annotate") {
		t.Fatalf("annotate: %s", errb)
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
