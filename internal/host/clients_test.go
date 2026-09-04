package host

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupClient(t *testing.T) {
	if _, ok := LookupClient("grok"); !ok {
		t.Fatal("grok")
	}
	c, ok := LookupClient("Claude")
	if !ok || c.Name != "claude" {
		t.Fatalf("claude: %+v %v", c, ok)
	}
	if _, ok := LookupClient("bob"); ok {
		t.Fatal("unknown")
	}
}

func TestScanClientsDefaultOnly(t *testing.T) {
	h := &Fake{Home: t.TempDir(), Client: "grok"}
	got := ScanClients(h)
	if len(got) != 1 || got[0].Name != "grok" || got[0].Skill || got[0].MCP || got[0].ConfigDir {
		t.Fatalf("%+v", got)
	}
}

func TestScanClientsExtraConfigDir(t *testing.T) {
	h := &Fake{Home: t.TempDir(), Client: "grok"}
	if err := os.MkdirAll(filepath.Join(h.Home, ".cursor"), 0o700); err != nil {
		t.Fatal(err)
	}
	got := ScanClients(h)
	names := map[string]ClientStatus{}
	for _, c := range got {
		names[c.Name] = c
	}
	if _, ok := names["grok"]; !ok {
		t.Fatalf("default missing: %+v", got)
	}
	cur, ok := names["cursor"]
	if !ok || !cur.ConfigDir || cur.Skill || cur.MCP {
		t.Fatalf("cursor: %+v", got)
	}
}

func TestScanClientsSkillAndMCP(t *testing.T) {
	h := &Fake{Home: t.TempDir()}
	skill := filepath.Join(h.Home, ".grok", "skills", "kage", "SKILL.md")
	if err := h.WriteFile(skill, []byte("x")); err != nil {
		t.Fatal(err)
	}
	mcp := filepath.Join(h.Home, ".grok", "config.toml")
	if err := h.WriteFile(mcp, []byte("[mcp_servers.kage]\ncommand = \"kage\"\n")); err != nil {
		t.Fatal(err)
	}
	got := ScanClients(h)
	if len(got) != 1 || got[0].Name != "grok" || !got[0].Skill || !got[0].MCP || !got[0].ConfigDir {
		t.Fatalf("%+v", got)
	}
}

func TestHasMCPServer(t *testing.T) {
	if !HasMCPServer([]byte("[mcp_servers.kage]\ncommand = \"kage\"\n"), MCPTOML, "kage") {
		t.Fatal("toml")
	}
	if HasMCPServer([]byte("# [mcp_servers.kage]\n"), MCPTOML, "kage") {
		t.Fatal("comment")
	}
	if HasMCPServer([]byte("[mcp_servers.kage_other]\n"), MCPTOML, "kage") {
		t.Fatal("prefix")
	}
	js := []byte(`{"mcpServers":{"kage":{"command":"kage"}}}`)
	if !HasMCPServer(js, MCPJSON, "kage") {
		t.Fatal("json")
	}
	if HasMCPServer([]byte(`{"mcpServers":{"other":{}}}`), MCPJSON, "kage") {
		t.Fatal("other")
	}
}
