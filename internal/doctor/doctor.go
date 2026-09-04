package doctor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/6abe/kage/internal/host"
)

const (
	ExitOK           = 0
	ExitCaptureFail  = 1
	ExitInputMissing = 3
)

type Tool struct {
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type Capture struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

type Report struct {
	OK             bool                `json:"ok"`
	Compositor     string              `json:"compositor"`
	WaylandDisplay string              `json:"wayland_display"`
	Grim           Tool                `json:"grim"`
	Hyprctl        Tool                `json:"hyprctl"`
	Wtype          Tool                `json:"wtype"`
	Ydotool        Tool                `json:"ydotool"`
	WlCopy         Tool                `json:"wl_copy"`
	Capture        Capture             `json:"capture"`
	InputBackend   string              `json:"input_backend"`
	DefaultClient  string              `json:"default_client"`
	Clients        []host.ClientStatus `json:"clients"`
}

func lookup(h host.Host, name string) Tool {
	p, err := h.LookPath(name)
	if err != nil {
		return Tool{Present: false, Hint: host.ToolHint(name)}
	}
	return Tool{Present: true, Path: p}
}

func Run(h host.Host) (Report, int) {
	r := Report{
		WaylandDisplay: h.Env("WAYLAND_DISPLAY"),
		Grim:           lookup(h, "grim"),
		Hyprctl:        lookup(h, "hyprctl"),
		Wtype:          lookup(h, "wtype"),
		Ydotool:        lookup(h, "ydotool"),
		WlCopy:         lookup(h, "wl-copy"),
		DefaultClient:  h.DefaultClient(),
		Clients:        h.ClientsOnDisk(),
	}
	if r.DefaultClient == "" {
		r.DefaultClient = "grok"
	}
	if sig := h.Env("HYPRLAND_INSTANCE_SIGNATURE"); sig != "" {
		r.Compositor = "hyprland"
	} else if r.Hyprctl.Present {
		r.Compositor = "hyprland"
	} else {
		r.Compositor = "none"
	}

	switch {
	case !r.Grim.Present:
		r.Capture = Capture{OK: false, Error: "grim not found", Hint: host.ToolHint("grim")}
	default:
		if err := probeCapture(h); err != nil {
			cap := Capture{OK: false, Error: err.Error()}
			if r.WaylandDisplay == "" {
				cap.Hint = "set WAYLAND_DISPLAY (Hyprland session)"
			}
			r.Capture = cap
		} else {
			r.Capture = Capture{OK: true}
		}
	}

	switch {
	case r.Wtype.Present && r.Ydotool.Present:
		r.InputBackend = "wtype+ydotool"
	case r.Wtype.Present:
		r.InputBackend = "wtype"
	case r.Ydotool.Present:
		r.InputBackend = "ydotool"
	default:
		r.InputBackend = "none"
	}

	r.OK = r.Capture.OK
	code := ExitOK
	if !r.Capture.OK {
		code = ExitCaptureFail
	} else if r.InputBackend == "none" {
		code = ExitInputMissing
	}
	return r, code
}

func Human(r Report) string {
	line := func(name, val string) string {
		return fmt.Sprintf("%-16s %s\n", name, val)
	}
	tool := func(t Tool) string {
		if t.Present {
			return t.Path
		}
		if t.Hint != "" {
			return "missing  (" + t.Hint + ")"
		}
		return "missing"
	}
	cap := "ok"
	if !r.Capture.OK {
		cap = "fail"
		if r.Capture.Error != "" {
			cap += "  " + r.Capture.Error
		}
		if r.Capture.Hint != "" {
			cap += "  (" + r.Capture.Hint + ")"
		}
	}
	s := ""
	s += line("compositor", r.Compositor)
	s += line("WAYLAND_DISPLAY", dash(r.WaylandDisplay))
	s += line("grim", tool(r.Grim))
	s += line("hyprctl", tool(r.Hyprctl))
	s += line("wtype", tool(r.Wtype))
	s += line("ydotool", tool(r.Ydotool))
	s += line("wl-copy", tool(r.WlCopy))
	s += line("capture", cap)
	s += line("input_backend", r.InputBackend)
	s += line("default_client", r.DefaultClient)
	for _, c := range r.Clients {
		s += line("client:"+c.Name, fmt.Sprintf("skill=%v mcp=%v", c.Skill, c.MCP))
	}
	return s
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func probeCapture(h host.Host) error {
	dir := h.Env("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "kage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "doctor-probe.png")
	defer os.Remove(path)
	return h.Grim(host.GrimProbeArgs(path)...)
}
