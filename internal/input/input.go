package input

import (
	"fmt"
	"strings"
)

const (
	Denied = "input not allowed; need --yes, KAGE_ALLOW_INPUT=1, or allow_input = true in config"
	Hint   = "pass --yes, set KAGE_ALLOW_INPUT=1, or allow_input = true in config"
)

// Allowed is the type/press gate. Observe commands must not call this.
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
