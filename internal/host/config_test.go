package host

import "testing"

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
