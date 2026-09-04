package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
	"github.com/6abe/kage/internal/input"
)

type focusOut struct {
	OK     bool        `json:"ok"`
	Window hypr.Window `json:"window"`
}

type typeOut struct {
	OK         bool        `json:"ok"`
	Window     hypr.Window `json:"window"`
	Clear      bool        `json:"clear"`
	ClearSends []string    `json:"clear_sends,omitempty"`
	N          int         `json:"n"`
}

type pressOut struct {
	OK     bool        `json:"ok"`
	Window hypr.Window `json:"window"`
	Key    string      `json:"key"`
}

func runFocus(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", "kage focus --window ADDRESS|CLASS|TITLE")
	}
	if inv.window == "" {
		return writeFail(stderr, "--window is required", "kage focus --window ADDRESS|CLASS|TITLE")
	}
	if len(inv.rest) > 0 {
		return writeFail(stderr, "unexpected arguments: "+strings.Join(inv.rest, " "), "kage focus --window ADDRESS|CLASS|TITLE")
	}
	win, code := resolveTarget(h, inv.window, stderr)
	if code != 0 {
		return code
	}
	if err := h.HyprctlDispatch("focuswindow", "address:"+win.Address); err != nil {
		return writeFail(stderr, err.Error(), hyprHint(h))
	}
	h.Log("focus address=" + win.Address)
	if inv.human {
		fmt.Fprintf(stdout, "%s  %s  %s\n", win.Address, win.Class, win.Title)
		return 0
	}
	return writeJSON(stdout, focusOut{OK: true, Window: win})
}

func runType(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if len(inv.rest) != 1 {
		return writeFail(stderr, "usage: kage type TEXT [--window ADDRESS|CLASS|TITLE] [--clear] [--yes]", "--clear sends Ctrl+A then TEXT (empty TEXT also sends BackSpace)")
	}
	text := inv.rest[0]
	if code := requireInput(h, inv.yes, stderr); code != 0 {
		return code
	}
	if _, err := h.LookPath("wtype"); err != nil {
		return writeFail(stderr, "wtype not found", host.ToolHint("wtype"))
	}
	win, code := resolveTarget(h, inv.window, stderr)
	if code != 0 {
		return code
	}
	if inv.window != "" {
		if err := h.HyprctlDispatch("focuswindow", "address:"+win.Address); err != nil {
			return writeFail(stderr, err.Error(), hyprHint(h))
		}
	}
	if err := h.Wtype(input.TypeArgs(text, inv.clear)...); err != nil {
		return writeFail(stderr, err.Error(), host.ToolHint("wtype"))
	}
	h.Log(fmt.Sprintf("type n=%d clear=%t address=%s", utf8.RuneCountInString(text), inv.clear, win.Address))
	out := typeOut{OK: true, Window: win, Clear: inv.clear, N: utf8.RuneCountInString(text)}
	if inv.clear {
		out.ClearSends = input.ClearSends(text)
	}
	if inv.human {
		fmt.Fprintf(stdout, "type n=%d %s\n", out.N, win.Address)
		return 0
	}
	return writeJSON(stdout, out)
}

func runPress(h host.Host, inv invocation, stdout, stderr io.Writer) int {
	if inv.clear {
		return writeFail(stderr, "unexpected flag: --clear", "kage press KEY [--window ADDRESS|CLASS|TITLE] [--yes]")
	}
	if len(inv.rest) != 1 {
		return writeFail(stderr, "usage: kage press KEY [--window ADDRESS|CLASS|TITLE] [--yes]", "keys: Return, Tab, Escape, BackSpace, space, Left, Right, Up, Down")
	}
	key, err := input.CanonicalKey(inv.rest[0])
	if err != nil {
		return writeFail(stderr, err.Error(), "keys: Return, Tab, Escape, BackSpace, space, Left, Right, Up, Down")
	}
	if code := requireInput(h, inv.yes, stderr); code != 0 {
		return code
	}
	if _, err := h.LookPath("wtype"); err != nil {
		return writeFail(stderr, "wtype not found", host.ToolHint("wtype"))
	}
	win, code := resolveTarget(h, inv.window, stderr)
	if code != 0 {
		return code
	}
	if inv.window != "" {
		if err := h.HyprctlDispatch("focuswindow", "address:"+win.Address); err != nil {
			return writeFail(stderr, err.Error(), hyprHint(h))
		}
	}
	if err := h.Wtype(input.PressArgs(key)...); err != nil {
		return writeFail(stderr, err.Error(), host.ToolHint("wtype"))
	}
	h.Log("press address=" + win.Address)
	if inv.human {
		fmt.Fprintf(stdout, "press %s %s\n", key, win.Address)
		return 0
	}
	return writeJSON(stdout, pressOut{OK: true, Window: win, Key: key})
}

func requireInput(h host.Host, yes bool, stderr io.Writer) int {
	if input.Allowed(yes, h.Env("KAGE_ALLOW_INPUT"), h.AllowInput()) {
		return 0
	}
	return writeFail(stderr, input.Denied, input.Hint)
}

func resolveTarget(h host.Host, query string, stderr io.Writer) (hypr.Window, int) {
	if _, err := h.LookPath("hyprctl"); err != nil {
		return hypr.Window{}, writeFail(stderr, "hyprctl not found", host.ToolHint("hyprctl"))
	}
	wins, err := hypr.ListWindows(h)
	if err != nil {
		return hypr.Window{}, writeFail(stderr, err.Error(), hyprHint(h))
	}
	if query == "" {
		w, err := hypr.FocusedWindow(wins)
		if err != nil {
			return hypr.Window{}, writeFail(stderr, err.Error(), "focus a window or pass --window")
		}
		return w, 0
	}
	w, matches, err := hypr.MatchWindow(wins, query)
	if errors.Is(err, hypr.ErrAmbiguous) {
		_ = encode(stderr, fail{
			OK:      false,
			Error:   "ambiguous window match",
			Hint:    "disambiguate with an exact --window address (0x...)",
			Matches: matches,
		})
		return hypr.Window{}, 2
	}
	if errors.Is(err, hypr.ErrNotFound) {
		return hypr.Window{}, writeFail(stderr, "no matching window", "kage windows")
	}
	if err != nil {
		return hypr.Window{}, writeFail(stderr, err.Error(), "")
	}
	return w, 0
}
