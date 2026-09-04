package host

import (
	"strings"
	"testing"
)

func TestParseDefaultClient(t *testing.T) {
	if g := ParseDefaultClient(nil); g != "grok" {
		t.Fatalf("empty: %s", g)
	}
	if g := ParseDefaultClient([]byte("default_client = \"claude\"\n")); g != "claude" {
		t.Fatalf("claude: %s", g)
	}
	if g := ParseDefaultClient([]byte("# default_client = \"x\"\n")); g != "grok" {
		t.Fatalf("comment: %s", g)
	}
}

func TestParseAllowInput(t *testing.T) {
	if ParseConfig(nil).AllowInput {
		t.Fatal("empty should be false")
	}
	if !ParseConfig([]byte("allow_input = true\n")).AllowInput {
		t.Fatal("true")
	}
	if !ParseConfig([]byte("allow_input = 1\n")).AllowInput {
		t.Fatal("1")
	}
	if ParseConfig([]byte("allow_input = false\n")).AllowInput {
		t.Fatal("false")
	}
	if ParseConfig([]byte("# allow_input = true\n")).AllowInput {
		t.Fatal("comment")
	}
}

func TestGrimProbeArgs(t *testing.T) {
	args := GrimProbeArgs("/tmp/kage/doctor-probe.png")
	if len(args) != 3 || args[0] != "-g" || args[1] != "0,0 1x1" {
		t.Fatalf("argv %q", args)
	}
	if args[1] == "0,0,1,1" || !strings.Contains(args[1], " ") {
		t.Fatalf("grim wants x,y widthxheight, got %q", args[1])
	}
	if args[2] != "/tmp/kage/doctor-probe.png" {
		t.Fatalf("output %q", args[2])
	}
}

func TestToolHint(t *testing.T) {
	if ToolHint("grim") != "omarchy pkg add grim" {
		t.Fatal(ToolHint("grim"))
	}
	if ToolHint("hyprctl") != "omarchy pkg add hyprland" {
		t.Fatal(ToolHint("hyprctl"))
	}
	if ToolHint("wl-copy") != "omarchy pkg add wl-clipboard" {
		t.Fatal(ToolHint("wl-copy"))
	}
}
