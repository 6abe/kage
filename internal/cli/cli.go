package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/6abe/kage/internal/doctor"
	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
)

type fail struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Hint  string `json:"hint,omitempty"`
}

type windowsOut struct {
	OK      bool          `json:"ok"`
	Windows []hypr.Window `json:"windows"`
}

type monitorsOut struct {
	OK       bool           `json:"ok"`
	Monitors []hypr.Monitor `json:"monitors"`
}

// Run is the kage CLI. Tests call this with a fake Host.
func Run(h host.Host, args []string, stdout, stderr io.Writer) int {
	cmd, human, err := parseArgs(args)
	if err != nil {
		return writeFail(stderr, err.Error(), "kage windows|monitors|doctor [--human]")
	}
	switch cmd {
	case "help":
		msg := "kage windows|monitors|doctor [--human]"
		if human {
			fmt.Fprintln(stdout, msg)
			return 0
		}
		return writeJSON(stdout, map[string]any{"ok": true, "usage": msg})
	case "windows":
		return runWindows(h, human, stdout, stderr)
	case "monitors":
		return runMonitors(h, human, stdout, stderr)
	case "doctor":
		return runDoctor(h, human, stdout, stderr)
	default:
		return writeFail(stderr, "unknown command: "+cmd, "kage windows|monitors|doctor [--human]")
	}
}

func parseArgs(args []string) (cmd string, human bool, err error) {
	var pos []string
	for _, a := range args {
		switch a {
		case "--human":
			human = true
		case "--help", "-h":
			return "help", human, nil
		default:
			if strings.HasPrefix(a, "-") {
				return "", false, fmt.Errorf("unknown flag: %s", a)
			}
			pos = append(pos, a)
		}
	}
	if len(pos) == 0 {
		return "", human, fmt.Errorf("usage: kage windows|monitors|doctor [--human]")
	}
	if len(pos) > 1 {
		return "", human, fmt.Errorf("unexpected arguments: %s", strings.Join(pos[1:], " "))
	}
	return pos[0], human, nil
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
	_ = encode(stderr, fail{OK: false, Error: msg, Hint: hint})
	return 1
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
