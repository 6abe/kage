package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/6abe/kage/internal/doctor"
	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
	"github.com/6abe/kage/internal/see"
)

type fail struct {
	OK      bool          `json:"ok"`
	Error   string        `json:"error"`
	Hint    string        `json:"hint,omitempty"`
	Matches []hypr.Window `json:"matches,omitempty"`
}

type windowsOut struct {
	OK      bool          `json:"ok"`
	Windows []hypr.Window `json:"windows"`
}

type monitorsOut struct {
	OK       bool           `json:"ok"`
	Monitors []hypr.Monitor `json:"monitors"`
}

const usage = "kage windows|monitors|doctor|see [--monitor NAME|--all] [--window ADDRESS|CLASS|TITLE] [--annotate] [--path FILE] [--max-width N] [--human]"

type parsed struct {
	cmd      string
	human    bool
	path     string
	monitor  string
	window   string
	all      bool
	annotate bool
	maxWidth int
	maxSet   bool
}

// Run is the kage CLI. Tests call this with a fake Host.
func Run(h host.Host, args []string, stdout, stderr io.Writer) int {
	p, err := parseArgs(args)
	if err != nil {
		return writeFail(stderr, err.Error(), usage)
	}
	switch p.cmd {
	case "help":
		if p.human {
			fmt.Fprintln(stdout, usage)
			return 0
		}
		return writeJSON(stdout, map[string]any{"ok": true, "usage": usage})
	case "windows":
		return runWindows(h, p.human, stdout, stderr)
	case "monitors":
		return runMonitors(h, p.human, stdout, stderr)
	case "doctor":
		return runDoctor(h, p.human, stdout, stderr)
	case "see":
		return runSee(h, p, stdout, stderr)
	default:
		return writeFail(stderr, "unknown command: "+p.cmd, usage)
	}
}

func parseArgs(args []string) (parsed, error) {
	var p parsed
	p.maxWidth = see.DefaultMaxWidth
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--human":
			p.human = true
		case a == "--help" || a == "-h":
			p.cmd = "help"
			return p, nil
		case a == "--all":
			p.all = true
		case a == "--annotate":
			p.annotate = true
		case a == "--path":
			v, err := flagValue(args, &i, "--path")
			if err != nil {
				return parsed{}, err
			}
			p.path = v
		case strings.HasPrefix(a, "--path="):
			p.path = strings.TrimPrefix(a, "--path=")
			if p.path == "" {
				return parsed{}, fmt.Errorf("flag --path requires a file path")
			}
		case a == "--monitor":
			v, err := flagValue(args, &i, "--monitor")
			if err != nil {
				return parsed{}, err
			}
			p.monitor = v
		case strings.HasPrefix(a, "--monitor="):
			p.monitor = strings.TrimPrefix(a, "--monitor=")
			if p.monitor == "" {
				return parsed{}, fmt.Errorf("flag --monitor requires a name")
			}
		case a == "--window":
			v, err := flagValue(args, &i, "--window")
			if err != nil {
				return parsed{}, err
			}
			p.window = v
		case strings.HasPrefix(a, "--window="):
			p.window = strings.TrimPrefix(a, "--window=")
			if p.window == "" {
				return parsed{}, fmt.Errorf("flag --window requires ADDRESS|CLASS|TITLE")
			}
		case a == "--max-width":
			v, err := flagValue(args, &i, "--max-width")
			if err != nil {
				return parsed{}, err
			}
			n, err := parseMaxWidth(v)
			if err != nil {
				return parsed{}, err
			}
			p.maxWidth = n
			p.maxSet = true
		case strings.HasPrefix(a, "--max-width="):
			n, err := parseMaxWidth(strings.TrimPrefix(a, "--max-width="))
			if err != nil {
				return parsed{}, err
			}
			p.maxWidth = n
			p.maxSet = true
		case strings.HasPrefix(a, "-"):
			return parsed{}, fmt.Errorf("unknown flag: %s", a)
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) == 0 {
		return parsed{}, fmt.Errorf("usage: %s", usage)
	}
	if len(pos) > 1 {
		return parsed{}, fmt.Errorf("unexpected arguments: %s", strings.Join(pos[1:], " "))
	}
	p.cmd = pos[0]
	if p.cmd != "see" {
		switch {
		case p.path != "":
			return parsed{}, fmt.Errorf("unknown flag: --path")
		case p.annotate:
			return parsed{}, fmt.Errorf("unknown flag: --annotate")
		case p.all:
			return parsed{}, fmt.Errorf("unknown flag: --all")
		case p.monitor != "":
			return parsed{}, fmt.Errorf("unknown flag: --monitor")
		case p.window != "":
			return parsed{}, fmt.Errorf("unknown flag: --window")
		case p.maxSet:
			return parsed{}, fmt.Errorf("unknown flag: --max-width")
		}
	}
	n := 0
	if p.all {
		n++
	}
	if p.monitor != "" {
		n++
	}
	if p.window != "" {
		n++
	}
	if n > 1 {
		return parsed{}, fmt.Errorf("--monitor, --window, and --all are mutually exclusive")
	}
	return p, nil
}

func flagValue(args []string, i *int, name string) (string, error) {
	if *i+1 >= len(args) || strings.HasPrefix(args[*i+1], "-") {
		switch name {
		case "--path":
			return "", fmt.Errorf("flag --path requires a file path")
		case "--monitor":
			return "", fmt.Errorf("flag --monitor requires a name")
		case "--window":
			return "", fmt.Errorf("flag --window requires ADDRESS|CLASS|TITLE")
		case "--max-width":
			return "", fmt.Errorf("flag --max-width requires a positive integer")
		default:
			return "", fmt.Errorf("flag %s requires a value", name)
		}
	}
	*i++
	return args[*i], nil
}

func parseMaxWidth(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("flag --max-width requires a positive integer")
	}
	return n, nil
}

func runWindows(h host.Host, human bool, stdout, stderr io.Writer) int {
	if _, err := h.LookPath("hyprctl"); err != nil {
		return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}
	wins, err := hypr.ListWindows(h)
	if err != nil {
		return writeFail(stderr, err.Error(), hyprHint(h))
	}
	if human {
		for _, w := range wins {
			focus := ""
			if w.Focus {
				focus = " focus"
			}
			float := ""
			if w.Floating {
				float = " floating"
			}
			mapped := "unmapped"
			if w.Mapped {
				mapped = "mapped"
			}
			fmt.Fprintf(stdout, "%s  %s  %s  %d,%d %dx%d  ws=%d  %s  %s%s%s\n",
				w.Address, w.Class, w.Title,
				w.Geometry.X, w.Geometry.Y, w.Geometry.Width, w.Geometry.Height,
				w.Workspace, w.Monitor, mapped, float, focus)
		}
		return 0
	}
	return writeJSON(stdout, windowsOut{OK: true, Windows: wins})
}

func runMonitors(h host.Host, human bool, stdout, stderr io.Writer) int {
	if _, err := h.LookPath("hyprctl"); err != nil {
		return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}
	mons, err := hypr.ListMonitors(h)
	if err != nil {
		return writeFail(stderr, err.Error(), hyprHint(h))
	}
	if human {
		for _, m := range mons {
			focus := ""
			if m.Focused {
				focus = " focused"
			}
			fmt.Fprintf(stdout, "%s  %dx%d+%d+%d  scale=%g%s\n",
				m.Name, m.Width, m.Height, m.X, m.Y, m.Scale, focus)
		}
		return 0
	}
	return writeJSON(stdout, monitorsOut{OK: true, Monitors: mons})
}

func runSee(h host.Host, p parsed, stdout, stderr io.Writer) int {
	if _, err := h.LookPath("hyprctl"); err != nil {
		return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}
	if _, err := h.LookPath("grim"); err != nil {
		return writeFail(stderr, "grim not found", host.ToolHint("grim"))
	}
	snap, err := see.Capture(h, see.Options{
		Path:     p.path,
		Monitor:  p.monitor,
		Window:   p.window,
		All:      p.all,
		Annotate: p.annotate,
		MaxWidth: p.maxWidth,
	})
	if err != nil {
		var amb *hypr.AmbiguousError
		if errors.As(err, &amb) {
			return writeFailCode(stderr, fail{
				OK:      false,
				Error:   amb.Error(),
				Hint:    "disambiguate with the window address (0x...)",
				Matches: amb.Matches,
			}, 2)
		}
		hint := hyprHint(h)
		switch {
		case strings.Contains(err.Error(), "no window matches"):
			hint = "kage windows"
		case strings.Contains(err.Error(), "no monitor named"):
			hint = "kage monitors"
		case strings.Contains(err.Error(), "grim") && h.Env("WAYLAND_DISPLAY") == "":
			hint = "set WAYLAND_DISPLAY (Hyprland session)"
		}
		return writeFail(stderr, err.Error(), hint)
	}
	if p.human {
		fmt.Fprintf(stdout, "%s  %s  %dx%d  %s\n",
			snap.SnapshotID, snap.Monitor.Name, snap.Width, snap.Height, snap.Path)
		return 0
	}
	return writeJSON(stdout, snap)
}

func runDoctor(h host.Host, human bool, stdout, stderr io.Writer) int {
	r, code := doctor.Run(h)
	if human {
		fmt.Fprint(stdout, doctor.Human(r))
		return code
	}
	if writeJSON(stdout, r) != 0 {
		return writeFail(stderr, "encode doctor report", "")
	}
	return code
}

func hyprHint(h host.Host) string {
	if h.Env("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return "start a Hyprland session (HYPRLAND_INSTANCE_SIGNATURE is empty)"
	}
	return ""
}

func writeFail(stderr io.Writer, msg, hint string) int {
	return writeFailCode(stderr, fail{OK: false, Error: msg, Hint: hint}, 1)
}

func writeFailCode(stderr io.Writer, v fail, code int) int {
	_ = encode(stderr, v)
	return code
}

func writeJSON(w io.Writer, v any) int {
	if err := encode(w, v); err != nil {
		return 1
	}
	return 0
}

func encode(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
