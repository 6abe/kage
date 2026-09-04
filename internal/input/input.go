package input

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	Denied       = "input not allowed; need --yes, KAGE_ALLOW_INPUT=1, or allow_input = true in config"
	Hint         = "pass --yes, set KAGE_ALLOW_INPUT=1, or allow_input = true in config"
	YdotoolHint  = "omarchy pkg add ydotool && systemctl --user start ydotoold"
	ClickNeedOne = "exactly one of --at X,Y or --on ID is required"
)

const (
	btnLeft   = "0xC0"
	btnRight  = "0xC1"
	btnMiddle = "0xC2"
)

// Allowed is the click/type/press/hotkey gate. Observe commands must not call this.
func Allowed(yes bool, env string, cfg bool) bool {
	return yes || env == "1" || cfg
}

var keys = map[string]string{
	"return":    "Return",
	"tab":       "Tab",
	"escape":    "Escape",
	"backspace": "BackSpace",
	"space":     "space",
	"left":      "Left",
	"right":     "Right",
	"up":        "Up",
	"down":      "Down",
}

// CanonicalKey maps a press name to the wtype/xkb key. Unknown names error.
func CanonicalKey(name string) (string, error) {
	k, ok := keys[strings.ToLower(name)]
	if !ok {
		return "", fmt.Errorf("unsupported key %q (Return, Tab, Escape, BackSpace, space, Left, Right, Up, Down)", name)
	}
	return k, nil
}

// PressArgs is the wtype argv for one named key (no binary).
func PressArgs(key string) []string {
	return []string{"-k", key}
}

// TypeArgs is the wtype argv for text. --clear sends Ctrl+A then the text
// (empty text also sends BackSpace so the field is actually cleared).
func TypeArgs(text string, clear bool) []string {
	var args []string
	if clear {
		args = append(args, "-M", "ctrl", "-k", "a", "-m", "ctrl")
		if text == "" {
			return append(args, "-k", "BackSpace")
		}
	}
	return append(args, "--", text)
}

// ClearSends names the keys --clear injects before TEXT, for JSON/help.
func ClearSends(text string) []string {
	if text == "" {
		return []string{"Ctrl+A", "BackSpace"}
	}
	return []string{"Ctrl+A"}
}

// ParseAt parses global compositor pixels as X,Y.
func ParseAt(s string) (x, y int, err error) {
	s = strings.TrimSpace(s)
	left, right, ok := strings.Cut(s, ",")
	if !ok {
		return 0, 0, fmt.Errorf("--at requires X,Y")
	}
	x, err = strconv.Atoi(strings.TrimSpace(left))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --at %q", s)
	}
	y, err = strconv.Atoi(strings.TrimSpace(right))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --at %q", s)
	}
	return x, y, nil
}

// CanonicalButton maps left|right|middle (default left) to the ydotool click code.
func CanonicalButton(name string) (label, code string, err error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "left":
		return "left", btnLeft, nil
	case "right":
		return "right", btnRight, nil
	case "middle":
		return "middle", btnMiddle, nil
	default:
		return "", "", fmt.Errorf("button must be left, right, or middle")
	}
}

// MoveArgs is the ydotool argv to warp to compositor pixels (no binary).
func MoveArgs(x, y int) []string {
	return []string{"mousemove", "--absolute", "-x", strconv.Itoa(x), "-y", strconv.Itoa(y)}
}

// ClickArgs is the ydotool argv for one button click (no binary).
func ClickArgs(code string) []string {
	return []string{"click", code}
}

// Chord is a parsed hotkey such as CTRL+C or SUPER+Q.
type Chord struct {
	Raw  string
	Mods string
	Key  string
}

var modNames = map[string]string{
	"ctrl":    "CTRL",
	"control": "CTRL",
	"alt":     "ALT",
	"super":   "SUPER",
	"win":     "SUPER",
	"meta":    "SUPER",
	"shift":   "SHIFT",
}

// ParseHotkey splits SUPER+Q / CTRL+C / ALT+F4 into Hyprland mods + key.
func ParseHotkey(s string) (Chord, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return Chord{}, fmt.Errorf("hotkey is required")
	}
	parts := strings.Split(raw, "+")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || strings.ContainsAny(p, `'"\`) {
			return Chord{}, fmt.Errorf("invalid hotkey %q", s)
		}
		parts[i] = p
	}
	key := parts[len(parts)-1]
	var mods []string
	for _, p := range parts[:len(parts)-1] {
		m, ok := modNames[strings.ToLower(p)]
		if !ok {
			return Chord{}, fmt.Errorf("unsupported modifier %q", p)
		}
		mods = append(mods, m)
	}
	return Chord{Raw: raw, Mods: strings.Join(mods, " "), Key: canonicalHotkeyKey(key)}, nil
}

func canonicalHotkeyKey(name string) string {
	if k, err := CanonicalKey(name); err == nil {
		return k
	}
	if len(name) == 1 && name[0] >= 'a' && name[0] <= 'z' {
		return strings.ToUpper(name)
	}
	return name
}

var (
	modKeycodes = map[string]int{
		"CTRL":  29,
		"SHIFT": 42,
		"ALT":   56,
		"SUPER": 125,
	}
	hotkeyKeycodes = map[string]int{
		"esc": 1, "escape": 1,
		"1": 2, "2": 3, "3": 4, "4": 5, "5": 6, "6": 7, "7": 8, "8": 9, "9": 10, "0": 11,
		"backspace": 14, "tab": 15,
		"q": 16, "w": 17, "e": 18, "r": 19, "t": 20, "y": 21, "u": 22, "i": 23, "o": 24, "p": 25,
		"a": 30, "s": 31, "d": 32, "f": 33, "g": 34, "h": 35, "j": 36, "k": 37, "l": 38,
		"return": 28, "enter": 28,
		"z": 44, "x": 45, "c": 46, "v": 47, "b": 48, "n": 49, "m": 50,
		"space": 57,
		"f1":    59, "f2": 60, "f3": 61, "f4": 62, "f5": 63, "f6": 64,
		"f7": 65, "f8": 66, "f9": 67, "f10": 68, "f11": 87, "f12": 88,
		"up": 103, "left": 105, "right": 106, "down": 108,
	}
)

// HotkeyYdotoolArgs is the ydotool key argv for a chord (no binary).
func HotkeyYdotoolArgs(c Chord) ([]string, error) {
	var down []string
	for _, m := range strings.Fields(c.Mods) {
		code, ok := modKeycodes[m]
		if !ok {
			return nil, fmt.Errorf("unsupported modifier %q", m)
		}
		down = append(down, strconv.Itoa(code)+":1")
	}
	k, ok := hotkeyKeycodes[strings.ToLower(c.Key)]
	if !ok {
		return nil, fmt.Errorf("unsupported hotkey key %q", c.Key)
	}
	down = append(down, strconv.Itoa(k)+":1")
	up := make([]string, 0, len(down))
	for i := len(down) - 1; i >= 0; i-- {
		tok, _, _ := strings.Cut(down[i], ":")
		up = append(up, tok+":0")
	}
	return append([]string{"key"}, append(down, up...)...), nil
}

// SendShortcutUnsupported reports whether hyprctl rejected send_shortcut as missing.
func SendShortcutUnsupported(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unknown dispatcher") ||
		strings.Contains(s, "invalid dispatcher") ||
		strings.Contains(s, "nil value") ||
		strings.Contains(s, "not a function")
}
