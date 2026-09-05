package input

import (
	"reflect"
	"strings"
	"testing"
)

func TestYdotoolHintUsesArchUnit(t *testing.T) {
	if !strings.Contains(YdotoolHint, "ydotool.service") {
		t.Fatalf("hint must name Arch unit ydotool.service: %s", YdotoolHint)
	}
	if strings.Contains(YdotoolHint, "start ydotoold") {
		t.Fatalf("no ydotoold.service on Arch: %s", YdotoolHint)
	}
}

func TestAllowed(t *testing.T) {
	if Allowed(false, "", false) {
		t.Fatal("denied")
	}
	if !Allowed(true, "", false) || !Allowed(false, "1", false) || !Allowed(false, "", true) {
		t.Fatal("allowed")
	}
	if Allowed(false, "true", false) {
		t.Fatal("only KAGE_ALLOW_INPUT=1 counts")
	}
}

func TestTypeArgsClear(t *testing.T) {
	got := TypeArgs("hi", true)
	want := []string{"-M", "ctrl", "-k", "a", "-m", "ctrl", "--", "hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
	got = TypeArgs("", true)
	want = []string{"-M", "ctrl", "-k", "a", "-m", "ctrl", "-k", "BackSpace"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
}

func TestCanonicalKey(t *testing.T) {
	k, err := CanonicalKey("return")
	if err != nil || k != "Return" {
		t.Fatalf("%s %v", k, err)
	}
	if _, err := CanonicalKey("F1"); err == nil {
		t.Fatal("F1")
	}
}

func TestParseAt(t *testing.T) {
	x, y, err := ParseAt("100,200")
	if err != nil || x != 100 || y != 200 {
		t.Fatalf("%d %d %v", x, y, err)
	}
	x, y, err = ParseAt(" -10 , 20 ")
	if err != nil || x != -10 || y != 20 {
		t.Fatalf("neg %d %d %v", x, y, err)
	}
	if _, _, err := ParseAt("100"); err == nil {
		t.Fatal("missing comma")
	}
	if _, _, err := ParseAt("a,b"); err == nil {
		t.Fatal("non-int")
	}
}

func TestCanonicalButton(t *testing.T) {
	label, code, err := CanonicalButton("")
	if err != nil || label != "left" || code != "0xC0" {
		t.Fatalf("%s %s %v", label, code, err)
	}
	if _, code, err = CanonicalButton("right"); err != nil || code != "0xC1" {
		t.Fatalf("right %s %v", code, err)
	}
	if _, code, err = CanonicalButton("middle"); err != nil || code != "0xC2" {
		t.Fatalf("middle %s %v", code, err)
	}
	if _, _, err := CanonicalButton("forward"); err == nil {
		t.Fatal("forward")
	}
}

func TestParseHotkey(t *testing.T) {
	c, err := ParseHotkey("CTRL+C")
	if err != nil || c.Mods != "CTRL" || c.Key != "C" {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = ParseHotkey("super+q")
	if err != nil || c.Mods != "SUPER" || c.Key != "Q" {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = ParseHotkey("ALT+F4")
	if err != nil || c.Mods != "ALT" || c.Key != "F4" {
		t.Fatalf("%+v %v", c, err)
	}
	c, err = ParseHotkey("ctrl+shift+c")
	if err != nil || c.Mods != "CTRL SHIFT" || c.Key != "C" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := ParseHotkey(`CTRL+'`); err == nil {
		t.Fatal("quote")
	}
	if _, err := ParseHotkey(""); err == nil {
		t.Fatal("empty")
	}
}

func TestHotkeyYdotoolArgs(t *testing.T) {
	c, err := ParseHotkey("CTRL+C")
	if err != nil {
		t.Fatal(err)
	}
	got, err := HotkeyYdotoolArgs(c)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"key", "29:1", "46:1", "46:0", "29:0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%q", got)
	}
}

func TestMoveAndClickArgs(t *testing.T) {
	if got := MoveArgs(10, -4); !reflect.DeepEqual(got, []string{"mousemove", "--absolute", "-x", "10", "-y", "-4"}) {
		t.Fatalf("%q", got)
	}
	if got := ClickArgs("0xC0"); !reflect.DeepEqual(got, []string{"click", "0xC0"}) {
		t.Fatalf("%q", got)
	}
}

func TestSendShortcutUnsupported(t *testing.T) {
	if SendShortcutUnsupported(nil) {
		t.Fatal("nil")
	}
	if !SendShortcutUnsupported(errString("unknown dispatcher")) {
		t.Fatal("unknown")
	}
	if SendShortcutUnsupported(errString("hl.send_shortcut: 'key' is required")) {
		t.Fatal("must not treat valid-dispatcher errors as missing")
	}
	if SendShortcutUnsupported(errString("unknown key")) {
		t.Fatal("unknown key is not a missing dispatcher")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
