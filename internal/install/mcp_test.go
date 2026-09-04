package install

import (
	"strings"
	"testing"

	"github.com/6abe/kage/internal/host"
)

func TestUpsertStripTOML(t *testing.T) {
	in := []byte(`[ui]
x = 1

[mcp_servers.other]
command = "x"

[mcp_servers.kage]
command = "old"
args = ["nope"]

[mcp_servers.kage.env]
FOO = "bar"
`)
	out := upsertTOML(in, "kage", "kage", []string{"mcp"})
	s := string(out)
	if !strings.Contains(s, "[ui]") || !strings.Contains(s, `command = "x"`) {
		t.Fatalf("preserved:\n%s", s)
	}
	if strings.Contains(s, "old") || strings.Contains(s, "[mcp_servers.kage.env]") {
		t.Fatalf("stale kage:\n%s", s)
	}
	if !strings.Contains(s, "[mcp_servers.kage]") || !strings.Contains(s, `args = ["mcp"]`) || !strings.Contains(s, "enabled = true") {
		t.Fatalf("new kage:\n%s", s)
	}
	if host.HasMCPServer([]byte(s), host.MCPTOML, "other") == false {
		t.Fatal("other")
	}
	stripped := string(stripTOML(out, "kage"))
	if strings.Contains(stripped, "mcp_servers.kage") {
		t.Fatalf("strip:\n%s", stripped)
	}
	if !strings.Contains(stripped, "[mcp_servers.other]") || !strings.Contains(stripped, "[ui]") {
		t.Fatalf("strip kept:\n%s", stripped)
	}
}

func TestUpsertStripJSON(t *testing.T) {
	in := []byte(`{
  "firstStartTime": "x",
  "mcpServers": {
    "other": {"command": "x"},
    "kage": {"command": "old"}
  }
}`)
	out, err := upsertJSON(in, "kage", "kage", []string{"mcp"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"firstStartTime": "x"`) || !strings.Contains(s, `"other"`) {
		t.Fatalf("preserved:\n%s", s)
	}
	if strings.Contains(s, "old") {
		t.Fatalf("stale:\n%s", s)
	}
	if !strings.Contains(s, `"command": "kage"`) || !strings.Contains(s, `"mcp"`) {
		t.Fatalf("new:\n%s", s)
	}
	stripped, err := stripJSON(out, "kage")
	if err != nil {
		t.Fatal(err)
	}
	if host.HasMCPServer(stripped, host.MCPJSON, "kage") {
		t.Fatalf("still present:\n%s", stripped)
	}
	if !host.HasMCPServer(stripped, host.MCPJSON, "other") {
		t.Fatalf("lost other:\n%s", stripped)
	}
}

func TestUpsertJSONRejectsArray(t *testing.T) {
	_, err := upsertJSON([]byte(`{"mcpServers":[]}`), "kage", "kage", []string{"mcp"})
	if err == nil || !strings.Contains(err.Error(), "mcpServers") {
		t.Fatalf("err %v", err)
	}
}

func TestUpsertJSONInvalid(t *testing.T) {
	_, err := upsertJSON([]byte(`{`), "kage", "kage", []string{"mcp"})
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("err %v", err)
	}
}
