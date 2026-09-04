package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/6abe/kage/internal/cli"
	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/mcp"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
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

var wantTools = []string{
	"kage_doctor", "kage_see", "kage_windows", "kage_focus",
	"kage_click", "kage_type", "kage_press", "kage_hotkey",
}

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
			"grim":     "/usr/bin/grim",
			"hyprctl":  "/usr/bin/hyprctl",
			"wtype":    "/usr/bin/wtype",
			"ydotool":  "/usr/bin/ydotool",
			"ydotoold": "/usr/bin/ydotoold",
			"wl-copy":  "/usr/bin/wl-copy",
		},
		Client: "grok",
		Disk:   []host.ClientStatus{{Name: "grok", Skill: false, MCP: false}},
	}
}

func seeHost(t *testing.T) *host.Fake {
	t.Helper()
	h := okHost()
	h.Environ["XDG_RUNTIME_DIR"] = t.TempDir()
	return h
}

func execCLI(h host.Host, args ...string) (stdout, stderr string, code int) {
	var out, err bytes.Buffer
	code = cli.Run(h, args, &out, &err)
	return out.String(), err.String(), code
}

func startSession(t *testing.T, h host.Host) *sdkmcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := sdkmcp.NewInMemoryTransports()
	ss, err := mcp.New(h, cli.Run).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestInitializeAndToolsList(t *testing.T) {
	cs := startSession(t, okHost())
	init := cs.InitializeResult()
	if init == nil || init.ServerInfo == nil || init.ServerInfo.Name != "kage" {
		t.Fatalf("initialize: %+v", init)
	}
	if init.ProtocolVersion == "" {
		t.Fatal("missing protocolVersion")
	}
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != len(wantTools) {
		t.Fatalf("tools %d: %+v", len(res.Tools), names(res.Tools))
	}
	for i, name := range wantTools {
		if res.Tools[i].Name != name {
			t.Fatalf("tool %d: %s want %s", i, res.Tools[i].Name, name)
		}
	}
	see := res.Tools[1]
	d := strings.ToLower(see.Description)
	for _, n := range []string{"path", "click", "coordinates", "window", "guess"} {
		if !strings.Contains(d, n) {
			t.Fatalf("see description missing %q: %s", n, see.Description)
		}
	}
}

func TestStdioJSONRPCInitializeAndToolsList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sr, sw := io.Pipe()
	cr, cw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- mcp.New(okHost(), cli.Run).Run(ctx, &sdkmcp.IOTransport{Reader: sr, Writer: cw})
	}()
	enc := json.NewEncoder(sw)
	dec := json.NewDecoder(cr)
	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "kage-test", "version": "0"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	var init rpcMsg
	if err := dec.Decode(&init); err != nil {
		t.Fatal(err)
	}
	if init.Error != nil {
		t.Fatalf("initialize error: %s", init.Error)
	}
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(init.Result, &result); err != nil {
		t.Fatalf("initialize result: %v %s", err, init.Result)
	}
	if result.ProtocolVersion == "" || result.ServerInfo.Name != "kage" {
		t.Fatalf("initialize: %+v", result)
	}
	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	}); err != nil {
		t.Fatal(err)
	}
	var listed rpcMsg
	if err := dec.Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if listed.Error != nil {
		t.Fatalf("tools/list error: %s", listed.Error)
	}
	var tools struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(listed.Result, &tools); err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(tools.Tools))
	for i, tl := range tools.Tools {
		got[i] = tl.Name
	}
	if !reflect.DeepEqual(got, wantTools) {
		t.Fatalf("tools/list %q", got)
	}
	_ = sw.Close()
	_ = cw.Close()
	cancel()
	<-done
}

type rpcMsg struct {
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

func TestSeeAndClickMatchCLI(t *testing.T) {
	h := seeHost(t)
	cs := startSession(t, h)
	ctx := context.Background()

	seeRes, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "kage_see",
		Arguments: map[string]any{"annotate": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seeRes.IsError {
		t.Fatalf("see error: %+v", seeRes)
	}
	text := textJSON(t, seeRes)
	assertSameJSONShape(t, text, "path", "windows", "snapshot_id")
	var snap struct {
		OK    bool   `json:"ok"`
		Path  string `json:"path"`
		Data  string `json:"data"`
		Image string `json:"image"`
	}
	if err := json.Unmarshal([]byte(text), &snap); err != nil {
		t.Fatal(err)
	}
	if !snap.OK || snap.Path == "" {
		t.Fatalf("see json: %s", text)
	}
	if snap.Data != "" || snap.Image != "" || strings.Contains(text, `"data"`) {
		t.Fatalf("screenshot bytes in JSON: %s", text)
	}
	if _, err := h.ReadFile(snap.Path); err != nil {
		t.Fatal(err)
	}
	img := imageBlock(seeRes)
	if img == nil {
		t.Fatal("missing image content block")
	}
	if img.MIMEType != "image/png" || !bytes.HasPrefix(img.Data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("image block: mime=%s n=%d", img.MIMEType, len(img.Data))
	}

	h.Allow = true
	clickRes, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      "kage_click",
		Arguments: map[string]any{"on": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if clickRes.IsError {
		t.Fatalf("click: %s", textJSON(t, clickRes))
	}
	clickText := textJSON(t, clickRes)
	var click struct {
		OK bool   `json:"ok"`
		At [2]int `json:"at"`
		On int    `json:"on"`
	}
	if err := json.Unmarshal([]byte(clickText), &click); err != nil {
		t.Fatal(err)
	}
	if !click.OK || click.On != 1 || click.At != [2]int{800, 530} {
		t.Fatalf("click json: %s", clickText)
	}
	if len(h.YdotoolCalls) < 2 {
		t.Fatalf("ydotool %q", h.YdotoolCalls)
	}
}

func TestToolsMatchCLIJSON(t *testing.T) {
	cases := []struct {
		tool string
		args map[string]any
		cli  []string
	}{
		{tool: "kage_doctor", cli: []string{"doctor"}},
		{tool: "kage_windows", cli: []string{"windows"}},
		{tool: "kage_focus", args: map[string]any{"window": "0x456"}, cli: []string{"focus", "--window", "0x456"}},
		{tool: "kage_type", args: map[string]any{"text": "hi", "yes": true}, cli: []string{"type", "--yes", "hi"}},
		{tool: "kage_press", args: map[string]any{"key": "Return", "yes": true}, cli: []string{"press", "--yes", "Return"}},
		{tool: "kage_hotkey", args: map[string]any{"hotkey": "SUPER+Q", "yes": true}, cli: []string{"hotkey", "--yes", "SUPER+Q"}},
		{tool: "kage_click", args: map[string]any{"at": "10,20", "yes": true}, cli: []string{"click", "--yes", "--at", "10,20"}},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			h := okHost()
			cs := startSession(t, h)
			res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatal(err)
			}
			if res.IsError {
				t.Fatalf("mcp error: %s", textJSON(t, res))
			}
			cliOut, errb, code := execCLI(okHost(), tc.cli...)
			if code != 0 {
				t.Fatalf("cli %d %s", code, errb)
			}
			if !jsonEqual(t, textJSON(t, res), cliOut) {
				t.Fatalf("mcp %s\ncli %s", textJSON(t, res), cliOut)
			}
		})
	}
}

func TestClickDeniedMatchesCLI(t *testing.T) {
	h := okHost()
	cs := startSession(t, h)
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "kage_click",
		Arguments: map[string]any{"at": "1,2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("want input denied")
	}
	_, errb, code := execCLI(okHost(), "click", "--at", "1,2")
	if code == 0 {
		t.Fatal("cli should deny")
	}
	if !jsonEqual(t, textJSON(t, res), errb) {
		t.Fatalf("mcp %s\ncli %s", textJSON(t, res), errb)
	}
	if len(h.YdotoolCalls) != 0 {
		t.Fatalf("must not click: %q", h.YdotoolCalls)
	}
}

func TestSeeErrorHasNoImage(t *testing.T) {
	h := okHost()
	delete(h.Paths, "grim")
	cs := startSession(t, h)
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "kage_see"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("want see error")
	}
	if imageBlock(res) != nil {
		t.Fatal("failed see must not attach image content")
	}
	text := textJSON(t, res)
	var payload struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Data  string `json:"data"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OK || payload.Error == "" || payload.Data != "" {
		t.Fatalf("see error json: %s", text)
	}
	h2 := okHost()
	delete(h2.Paths, "grim")
	_, errb, code := execCLI(h2, "see")
	if code == 0 {
		t.Fatal("cli should fail")
	}
	if !jsonEqual(t, text, errb) {
		t.Fatalf("mcp %s\ncli %s", text, errb)
	}
}

func TestSeeDescriptionOnEveryTool(t *testing.T) {
	cs := startSession(t, okHost())
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range res.Tools {
		d := strings.ToLower(tl.Description)
		if !strings.Contains(d, "kage_see") || !strings.Contains(d, "path") || !strings.Contains(d, "guess") {
			t.Fatalf("%s description: %s", tl.Name, tl.Description)
		}
	}
}

func names(tools []*sdkmcp.Tool) []string {
	out := make([]string, len(tools))
	for i, tl := range tools {
		out[i] = tl.Name
	}
	return out
}

func textJSON(t *testing.T, res *sdkmcp.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no text content")
	return ""
}

func imageBlock(res *sdkmcp.CallToolResult) *sdkmcp.ImageContent {
	for _, c := range res.Content {
		if img, ok := c.(*sdkmcp.ImageContent); ok {
			return img
		}
	}
	return nil
}

func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal([]byte(a), &x); err != nil {
		t.Fatalf("json a: %v (%s)", err, a)
	}
	if err := json.Unmarshal([]byte(b), &y); err != nil {
		t.Fatalf("json b: %v (%s)", err, b)
	}
	return reflect.DeepEqual(x, y)
}

func assertSameJSONShape(t *testing.T, raw string, keys ...string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s in %s", k, raw)
		}
	}
}
