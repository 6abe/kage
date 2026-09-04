package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
	"github.com/6abe/kage/internal/input"
	"github.com/6abe/kage/internal/see"
)

type clickOut struct {
	OK         bool         `json:"ok"`
	At         [2]int       `json:"at"`
	Button     string       `json:"button"`
	On         int          `json:"on,omitempty"`
	SnapshotID string       `json:"snapshot_id,omitempty"`
	Window     *hypr.Window `json:"window,omitempty"`
}

type hotkeyOut struct {
	OK      bool   `json:"ok"`
	Hotkey  string `json:"hotkey"`
	Backend string `json:"backend"`
}

type dispatchOut struct {
	OK   bool     `json:"ok"`
	Args []string `json:"args"`
}

func runClick(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", "kage click --at X,Y | --on ID [--yes]")
	}
	if len(inv.rest) > 0 {
		return writeFail(stderr, "unexpected arguments: "+strings.Join(inv.rest, " "), "kage click --at X,Y | --on ID [--yes]")
	}
	hasAt := inv.at != ""
	hasOn := inv.on != ""
	if hasAt == hasOn {
		return writeFail(stderr, input.ClickNeedOne, "kage click --at X,Y or kage click --on ID")
	}
	if inv.snapshot != "" && !hasOn {
		return writeFail(stderr, "--snapshot requires --on", "kage click --on ID [--snapshot ID]")
	}
	if hasOn && inv.window != "" {
		return writeFail(stderr, "--on cannot be combined with --window", "kage click --on ID or kage click --at X,Y [--window ADDRESS]")
	}

	var x, y int
	var onID int
	var snap see.Snapshot
	var win *hypr.Window
	if hasAt {
		var err error
		x, y, err = input.ParseAt(inv.at)
		if err != nil {
			return writeFail(stderr, err.Error(), "kage click --at X,Y")
		}
	} else {
		n, err := strconv.Atoi(strings.TrimSpace(inv.on))
		if err != nil || n < 1 {
			return writeFail(stderr, "--on requires an annotated window id", "kage see --annotate")
		}
		onID = n
	}
	button, code, err := input.CanonicalButton(inv.button)
	if err != nil {
		return writeFail(stderr, err.Error(), "kage click --button left|right|middle")
	}

	if code := requireInput(h, inv.yes, stderr); code != 0 {
		return code
	}
	if code := requireYdotool(h, stderr); code != 0 {
		return code
	}

	if hasOn {
		if _, err := h.LookPath("hyprctl"); err != nil {
			return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
		}
		var err error
		snap, err = loadClickSnapshot(h, inv.snapshot)
		if err != nil {
			return writeFail(stderr, err.Error(), "kage see --annotate")
		}
		sw, err := see.WindowByID(snap, onID)
		if err != nil {
			return writeFail(stderr, err.Error(), "kage see --annotate")
		}
		x, y, err = sw.Center()
		if err != nil {
			return writeFail(stderr, err.Error(), "kage see --annotate")
		}
		hw := snapshotHyprWindow(sw)
		win = &hw
		if err := h.HyprctlDispatch(hypr.FocusDispatch(sw.Address)); err != nil {
			return writeFail(stderr, err.Error(), hyprHint(h))
		}
	} else if inv.window != "" {
		w, code := resolveTarget(h, inv.window, stderr)
		if code != 0 {
			return code
		}
		if err := h.HyprctlDispatch(hypr.FocusDispatch(w.Address)); err != nil {
			return writeFail(stderr, err.Error(), hyprHint(h))
		}
		win = &w
	}

	if err := h.Ydotool(input.MoveArgs(x, y)...); err != nil {
		return writeFail(stderr, err.Error(), input.YdotoolHint)
	}
	if err := h.Ydotool(input.ClickArgs(code)...); err != nil {
		return writeFail(stderr, err.Error(), input.YdotoolHint)
	}
	h.Log(fmt.Sprintf("click at=%d,%d button=%s", x, y, button))
	out := clickOut{OK: true, At: [2]int{x, y}, Button: button, Window: win}
	if hasOn {
		out.On = onID
		out.SnapshotID = snap.SnapshotID
	}
	if inv.human {
		fmt.Fprintf(stdout, "click %s %d,%d\n", button, x, y)
		return 0
	}
	return writeJSON(stdout, out)
}

func runHotkey(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", "kage hotkey CHORD [--yes]")
	}
	if inv.window != "" {
		return writeFail(stderr, "unexpected flag: --window", "kage hotkey CHORD [--yes]")
	}
	if len(inv.rest) != 1 {
		return writeFail(stderr, "usage: kage hotkey CHORD [--yes]", `chords: SUPER+Q, CTRL+C, ALT+F4`)
	}
	chord, err := input.ParseHotkey(inv.rest[0])
	if err != nil {
		return writeFail(stderr, err.Error(), `chords: SUPER+Q, CTRL+C, ALT+F4`)
	}
	if code := requireInput(h, inv.yes, stderr); code != 0 {
		return code
	}
	if _, err := h.LookPath("hyprctl"); err != nil {
		return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}

	backend := "sendshortcut"
	sendErr := h.HyprctlDispatch(hypr.SendShortcutDispatch(chord.Mods, chord.Key))
	if sendErr != nil {
		if !input.SendShortcutUnsupported(sendErr) {
			return writeFail(stderr, sendErr.Error(), hyprHint(h))
		}
		if code := requireYdotool(h, stderr); code != 0 {
			return code
		}
		args, err := input.HotkeyYdotoolArgs(chord)
		if err != nil {
			return writeFail(stderr, err.Error(), "kage hotkey SUPER+Q")
		}
		if err := h.Ydotool(args...); err != nil {
			return writeFail(stderr, err.Error(), input.YdotoolHint)
		}
		backend = "ydotool"
	}
	h.Log("hotkey backend=" + backend)
	if inv.human {
		fmt.Fprintf(stdout, "hotkey %s %s\n", chord.Raw, backend)
		return 0
	}
	return writeJSON(stdout, hotkeyOut{OK: true, Hotkey: chord.Raw, Backend: backend})
}

func runDispatch(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", "kage dispatch <hyprctl dispatch args...>")
	}
	if inv.window != "" {
		return writeFail(stderr, "unexpected flag: --window", "kage dispatch <hyprctl dispatch args...>")
	}
	if len(inv.rest) == 0 {
		return writeFail(stderr, "usage: kage dispatch <hyprctl dispatch args...>", "hyprctl dispatch, e.g. hl.dsp.window.close()")
	}
	if _, err := h.LookPath("hyprctl"); err != nil {
		return writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}
	if err := h.HyprctlDispatch(inv.rest...); err != nil {
		return writeFail(stderr, err.Error(), hyprHint(h))
	}
	h.Log("dispatch")
	if inv.human {
		fmt.Fprintf(stdout, "dispatch %s\n", strings.Join(inv.rest, " "))
		return 0
	}
	return writeJSON(stdout, dispatchOut{OK: true, Args: inv.rest})
}

func requireYdotool(h host.Host, stderr io.Writer) int {
	if _, err := h.LookPath("ydotool"); err != nil {
		return writeFail(stderr, "ydotool not found", input.YdotoolHint)
	}
	if _, err := h.LookPath("ydotoold"); err != nil {
		return writeFail(stderr, "ydotoold not found", input.YdotoolHint)
	}
	return 0
}

func loadClickSnapshot(h host.Host, id string) (see.Snapshot, error) {
	if id != "" {
		return see.Load(h, id)
	}
	return see.Latest(h)
}

func snapshotHyprWindow(w see.Window) hypr.Window {
	return hypr.Window{
		Address:   w.Address,
		Class:     w.Class,
		Title:     w.Title,
		Geometry:  hypr.Geometry{X: w.At[0], Y: w.At[1], Width: w.Size[0], Height: w.Size[1]},
		Workspace: w.Workspace,
		Monitor:   w.Monitor,
		Mapped:    w.Mapped,
		Floating:  w.Floating,
		Focus:     w.Focus,
	}
}
