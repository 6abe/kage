package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strconv"

	"github.com/6abe/kage/internal/host"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const seeFirst = "Call kage_see first, read the PNG at path, then click using coordinates or window id from that snapshot. Do not guess coordinates."

const pngMagic = "\x89PNG\r\n\x1a\n"

type CLI func(h host.Host, args []string, stdout, stderr io.Writer) int

type server struct {
	h   host.Host
	run CLI
}

func New(h host.Host, run CLI) *sdkmcp.Server {
	s := &server{h: h, run: run}
	srv := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "kage", Version: "v1"}, &sdkmcp.ServerOptions{
		Instructions: seeFirst,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_doctor",
		Description: "Check compositor, capture, and input tools. " + seeFirst,
	}, s.doctor)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_see",
		Description: "Capture a screenshot PNG and JSON (path, windows, coordinates). " + seeFirst,
	}, s.see)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_windows",
		Description: "List Hyprland windows as JSON. " + seeFirst,
	}, s.windows)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_focus",
		Description: "Focus a window by address, class, or title. " + seeFirst,
	}, s.focus)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_click",
		Description: "Click at global compositor X,Y or on an annotated window id from kage_see. " + seeFirst,
	}, s.click)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_type",
		Description: "Type text into the focused or targeted window. " + seeFirst,
	}, s.typeText)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_press",
		Description: "Press a named key (Return, Tab, Escape, BackSpace, space, arrows). " + seeFirst,
	}, s.press)
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "kage_hotkey",
		Description: "Send a key chord such as SUPER+Q or CTRL+C. " + seeFirst,
	}, s.hotkey)
	srv.AddReceivingMiddleware(catalogOrder)
	return srv
}

var catalog = []string{
	"kage_doctor", "kage_see", "kage_windows", "kage_focus",
	"kage_click", "kage_type", "kage_press", "kage_hotkey",
}

func catalogOrder(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
	return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
		res, err := next(ctx, method, req)
		if err != nil || method != "tools/list" {
			return res, err
		}
		list, ok := res.(*sdkmcp.ListToolsResult)
		if !ok {
			return res, nil
		}
		byName := make(map[string]*sdkmcp.Tool, len(list.Tools))
		for _, t := range list.Tools {
			byName[t.Name] = t
		}
		ordered := make([]*sdkmcp.Tool, 0, len(catalog))
		for _, name := range catalog {
			if t, ok := byName[name]; ok {
				ordered = append(ordered, t)
			}
		}
		list.Tools = ordered
		return list, nil
	}
}

func Serve(ctx context.Context, h host.Host, run CLI) error {
	err := New(h, run).Run(ctx, &sdkmcp.StdioTransport{})
	if hangup(err) {
		return nil
	}
	return err
}

func hangup(err error) bool {
	return err == nil || errors.Is(err, io.EOF) || errors.Is(err, sdkmcp.ErrConnectionClosed)
}

type emptyArgs struct{}

type seeArgs struct {
	Monitor  string `json:"monitor,omitempty" jsonschema:"Hyprland monitor name"`
	Window   string `json:"window,omitempty" jsonschema:"window address, class, or title substring"`
	Annotate bool   `json:"annotate,omitempty" jsonschema:"draw numbered boxes on mapped windows"`
	MaxWidth int    `json:"max_width,omitempty" jsonschema:"downscale long edge in pixels; default 1920"`
	Path     string `json:"path,omitempty" jsonschema:"PNG file path"`
	All      bool   `json:"all,omitempty" jsonschema:"capture the union of all monitors"`
}

type focusArgs struct {
	Window string `json:"window" jsonschema:"window address, class, or title"`
}

type clickArgs struct {
	At       string `json:"at,omitempty" jsonschema:"global compositor pixels as X,Y"`
	On       int    `json:"on,omitempty" jsonschema:"annotated window id from the last kage_see snapshot"`
	Button   string `json:"button,omitempty" jsonschema:"left, right, or middle"`
	Snapshot string `json:"snapshot,omitempty" jsonschema:"see snapshot_id; used with on"`
	Window   string `json:"window,omitempty" jsonschema:"focus this window before clicking at"`
	Yes      bool   `json:"yes,omitempty" jsonschema:"allow input for this call (--yes)"`
}

type typeArgs struct {
	Text   string `json:"text" jsonschema:"text to type"`
	Window string `json:"window,omitempty" jsonschema:"window address, class, or title"`
	Clear  bool   `json:"clear,omitempty" jsonschema:"send Ctrl+A before typing"`
	Yes    bool   `json:"yes,omitempty" jsonschema:"allow input for this call (--yes)"`
}

type pressArgs struct {
	Key    string `json:"key" jsonschema:"Return, Tab, Escape, BackSpace, space, Left, Right, Up, Down"`
	Window string `json:"window,omitempty" jsonschema:"window address, class, or title"`
	Yes    bool   `json:"yes,omitempty" jsonschema:"allow input for this call (--yes)"`
}

type hotkeyArgs struct {
	Hotkey string `json:"hotkey" jsonschema:"chord such as SUPER+Q, CTRL+C, ALT+F4"`
	Yes    bool   `json:"yes,omitempty" jsonschema:"allow input for this call (--yes)"`
}

func (s *server) doctor(context.Context, *sdkmcp.CallToolRequest, emptyArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.wrap([]string{"doctor"}, false)
}

func (s *server) windows(context.Context, *sdkmcp.CallToolRequest, emptyArgs) (*sdkmcp.CallToolResult, any, error) {
	return s.wrap([]string{"windows"}, false)
}

func (s *server) see(_ context.Context, _ *sdkmcp.CallToolRequest, in seeArgs) (*sdkmcp.CallToolResult, any, error) {
	args := []string{"see"}
	if in.Monitor != "" {
		args = append(args, "--monitor", in.Monitor)
	}
	if in.Window != "" {
		args = append(args, "--window", in.Window)
	}
	if in.Annotate {
		args = append(args, "--annotate")
	}
	if in.MaxWidth > 0 {
		args = append(args, "--max-width", strconv.Itoa(in.MaxWidth))
	}
	if in.Path != "" {
		args = append(args, "--path", in.Path)
	}
	if in.All {
		args = append(args, "--all")
	}
	return s.wrap(args, true)
}

func (s *server) focus(_ context.Context, _ *sdkmcp.CallToolRequest, in focusArgs) (*sdkmcp.CallToolResult, any, error) {
	args := []string{"focus"}
	if in.Window != "" {
		args = append(args, "--window", in.Window)
	}
	return s.wrap(args, false)
}

func (s *server) click(_ context.Context, _ *sdkmcp.CallToolRequest, in clickArgs) (*sdkmcp.CallToolResult, any, error) {
	args := []string{"click"}
	if in.Yes {
		args = append(args, "--yes")
	}
	if in.At != "" {
		args = append(args, "--at", in.At)
	}
	if in.On != 0 {
		args = append(args, "--on", strconv.Itoa(in.On))
	}
	if in.Button != "" {
		args = append(args, "--button", in.Button)
	}
	if in.Snapshot != "" {
		args = append(args, "--snapshot", in.Snapshot)
	}
	if in.Window != "" {
		args = append(args, "--window", in.Window)
	}
	return s.wrap(args, false)
}

func (s *server) typeText(_ context.Context, _ *sdkmcp.CallToolRequest, in typeArgs) (*sdkmcp.CallToolResult, any, error) {
	args := []string{"type"}
	if in.Yes {
		args = append(args, "--yes")
	}
	if in.Clear {
		args = append(args, "--clear")
	}
	if in.Window != "" {
		args = append(args, "--window", in.Window)
	}
	args = append(args, in.Text)
	return s.wrap(args, false)
}

func (s *server) press(_ context.Context, _ *sdkmcp.CallToolRequest, in pressArgs) (*sdkmcp.CallToolResult, any, error) {
	args := []string{"press"}
	if in.Yes {
		args = append(args, "--yes")
	}
	if in.Window != "" {
		args = append(args, "--window", in.Window)
	}
	args = append(args, in.Key)
	return s.wrap(args, false)
}

func (s *server) hotkey(_ context.Context, _ *sdkmcp.CallToolRequest, in hotkeyArgs) (*sdkmcp.CallToolResult, any, error) {
	args := []string{"hotkey"}
	if in.Yes {
		args = append(args, "--yes")
	}
	args = append(args, in.Hotkey)
	return s.wrap(args, false)
}

func (s *server) wrap(args []string, attachPNG bool) (*sdkmcp.CallToolResult, any, error) {
	var stdout, stderr bytes.Buffer
	code := s.run(s.h, args, &stdout, &stderr)
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		raw = bytes.TrimSpace(stderr.Bytes())
	}
	text := string(raw)
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		parsed = map[string]any{"ok": false, "error": text}
	}
	isErr := code != 0 && !jsonOK(parsed)
	content := []sdkmcp.Content{&sdkmcp.TextContent{Text: text}}
	if attachPNG && !isErr {
		if img := pngBlock(s.h, parsed); img != nil {
			content = append(content, img)
		}
	}
	return &sdkmcp.CallToolResult{Content: content, IsError: isErr}, parsed, nil
}

func jsonOK(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	b, _ := m["ok"].(bool)
	return b
}

func pngBlock(h host.Host, v any) sdkmcp.Content {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	path, _ := m["path"].(string)
	if path == "" {
		return nil
	}
	b, err := h.ReadFile(path)
	if err != nil || !bytes.HasPrefix(b, []byte(pngMagic)) {
		return nil
	}
	return &sdkmcp.ImageContent{Data: b, MIMEType: "image/png"}
}
