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

const usage = "kage windows|monitors|doctor|see|focus|type|press [--human]"

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

type invocation struct {
	cmd      string
	human    bool
	yes      bool
	clear    bool
	window   string
	path     string
	rest     []string
	monitor  string
	all      bool
	annotate bool
	maxWidth int
	maxSet   bool
}

// Run is the kage CLI. Tests call this with a fake Host.
func Run(h host.Host, args []string, stdout, stderr io.Writer) int {
	inv, err := parseArgs(args)
	if err != nil {
		return writeFail(stderr, err.Error(), usage)
	}
	switch inv.cmd {
	case "help":
		msg := usage + "\n  see [--monitor NAME|--all] [--window ADDRESS|CLASS|TITLE] [--annotate] [--path FILE] [--max-width N]\n  focus --window ADDRESS|CLASS|TITLE\n  type TEXT [--window ADDRESS|CLASS|TITLE] [--clear] [--yes]\n  press KEY [--window ADDRESS|CLASS|TITLE] [--yes]\n  --clear sends Ctrl+A then TEXT (empty TEXT also sends BackSpace)\n  type/press need --yes, KAGE_ALLOW_INPUT=1, or allow_input = true in config"
		if inv.human {
			fmt.Fprintln(stdout, msg)
			return 0
		}
		return writeJSON(stdout, map[string]any{"ok": true, "usage": msg})
	case "windows":
		if code := rejectExtra(inv, stderr); code != 0 {
			return code
		}
		return runWindows(h, inv.human, stdout, stderr)
	case "monitors":
		if code := rejectExtra(inv, stderr); code != 0 {
			return code
		}
		return runMonitors(h, inv.human, stdout, stderr)
	case "doctor":
		if code := rejectExtra(inv, stderr); code != 0 {
			return code
		}
		return runDoctor(h, inv.human, stdout, stderr)
	case "see":
		if code := rejectSee(inv, stderr); code != 0 {
			return code
		}
		return runSee(h, inv, stdout, stderr)
	case "focus":
		if code := rejectSeeOnly(inv, stderr); code != 0 {
			return code
		}
		return runFocus(h, inv, stdout, stderr)
	case "type":
		if code := rejectSeeOnly(inv, stderr); code != 0 {
			return code
		}
		return runType(h, inv, stdout, stderr)
	case "press":
		if code := rejectSeeOnly(inv, stderr); code != 0 {
			return code
		}
		return runPress(h, inv, stdout, stderr)
	default:
		return writeFail(stderr, "unknown command: "+inv.cmd, usage)
	}
}

func parseArgs(args []string) (invocation, error) {
	var inv invocation
	inv.maxWidth = see.DefaultMaxWidth
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		switch {
		case a == "--human":
			inv.human = true
		case a == "--yes":
			inv.yes = true
		case a == "--clear":
			inv.clear = true
		case a == "--help" || a == "-h":
			inv.cmd = "help"
			return inv, nil
		case a == "--all":
			inv.all = true
		case a == "--annotate":
			inv.annotate = true
		case a == "--path":
			v, err := flagValue(args, &i, "--path")
			if err != nil {
				return inv, err
			}
			inv.path = v
		case strings.HasPrefix(a, "--path="):
			inv.path = strings.TrimPrefix(a, "--path=")
			if inv.path == "" {
				return inv, fmt.Errorf("flag --path requires a file path")
			}
		case a == "--monitor":
			v, err := flagValue(args, &i, "--monitor")
			if err != nil {
				return inv, err
			}
			inv.monitor = v
		case strings.HasPrefix(a, "--monitor="):
			inv.monitor = strings.TrimPrefix(a, "--monitor=")
			if inv.monitor == "" {
				return inv, fmt.Errorf("flag --monitor requires a name")
			}
		case a == "--window":
			v, err := flagValue(args, &i, "--window")
			if err != nil {
				return inv, err
			}
			inv.window = v
		case strings.HasPrefix(a, "--window="):
			inv.window = strings.TrimPrefix(a, "--window=")
			if inv.window == "" {
				return inv, fmt.Errorf("--window requires a value")
			}
		case a == "--max-width":
			v, err := flagValue(args, &i, "--max-width")
			if err != nil {
				return inv, err
			}
			n, err := parseMaxWidth(v)
			if err != nil {
				return inv, err
			}
			inv.maxWidth = n
			inv.maxSet = true
		case strings.HasPrefix(a, "--max-width="):
			n, err := parseMaxWidth(strings.TrimPrefix(a, "--max-width="))
			if err != nil {
				return inv, err
			}
			inv.maxWidth = n
			inv.maxSet = true
		case strings.HasPrefix(a, "-"):
			return inv, fmt.Errorf("unknown flag: %s", a)
		default:
			pos = append(pos, a)
		}
	}
	if inv.cmd == "help" {
		return inv, nil
	}
	if len(pos) == 0 {
		return inv, fmt.Errorf("usage: %s", usage)
	}
	inv.cmd = pos[0]
	inv.rest = pos[1:]
	if inv.path != "" && inv.cmd != "see" {
		return inv, fmt.Errorf("unknown flag: --path")
	}
	return inv, nil
}

func rejectExtra(inv invocation, stderr io.Writer) int {
	if inv.window != "" {
		return writeFail(stderr, "unexpected flag: --window", usage)
	}
	if code := rejectSeeOnly(inv, stderr); code != 0 {
		return code
	}
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", usage)
	}
	if len(inv.rest) > 0 {
		return writeFail(stderr, "unexpected arguments: "+strings.Join(inv.rest, " "), usage)
	}
	return 0
}

func rejectSeeOnly(inv invocation, stderr io.Writer) int {
	switch {
	case inv.annotate:
		return writeFail(stderr, "unknown flag: --annotate", usage)
	case inv.all:
		return writeFail(stderr, "unknown flag: --all", usage)
	case inv.monitor != "":
		return writeFail(stderr, "unknown flag: --monitor", usage)
	case inv.maxSet:
		return writeFail(stderr, "unknown flag: --max-width", usage)
	}
	return 0
}

func rejectSee(inv invocation, stderr io.Writer) int {
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", usage)
	}
	if len(inv.rest) > 0 {
		return writeFail(stderr, "unexpected arguments: "+strings.Join(inv.rest, " "), usage)
	}
	n := 0
	if inv.all {
		n++
	}
	if inv.monitor != "" {
		n++
	}
	if inv.window != "" {
		n++
	}
	if n > 1 {
		return writeFail(stderr, "--monitor, --window, and --all are mutually exclusive", usage)
	}
	return 0
}

func flagValue(args []string, i *int, name string) (string, error) {
	if *i+1 >= len(args) || strings.HasPrefix(args[*i+1], "-") {
		switch name {
		case "--path":
			return "", fmt.Errorf("flag --path requires a file path")
		case "--monitor":
			return "", fmt.Errorf("flag --monitor requires a name")
		case "--window":
			return "", fmt.Errorf("--window requires a value")
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

func runSee(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if _, err := h.LookPath("hyprctl"); err != nil {
		return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}
	if _, err := h.LookPath("grim"); err != nil {
		return writeFail(stderr, "grim not found", host.ToolHint("grim"))
	}
	snap, err := see.Capture(h, see.Options{
		Path:     inv.path,
		Monitor:  inv.monitor,
		Window:   inv.window,
		All:      inv.all,
		Annotate: inv.annotate,
		MaxWidth: inv.maxWidth,
	})
	if err != nil {
		var amb *see.AmbiguousError
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
	if inv.human {
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
