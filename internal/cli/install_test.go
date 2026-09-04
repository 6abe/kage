package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/install"
	"github.com/6abe/kage/skill"
)

func installHost(t *testing.T) *host.Fake {
	t.Helper()
	h := okHost()
	h.Home = t.TempDir()
	h.Disk = nil
	h.Environ["XDG_RUNTIME_DIR"] = t.TempDir()
	return h
}

func decodeInstall(t *testing.T, out string) install.Result {
	t.Helper()
	var p install.Result
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	return p
}

func TestInstallDefaultGrok(t *testing.T) {
	h := installHost(t)
	out, errb, code := execCLI(h, "install")
	if code != 0 {
		t.Fatalf("exit %d stderr=%s", code, errb)
	}
	p := decodeInstall(t, out)
	if !p.OK || p.Client != "grok" {
		t.Fatalf("%+v", p)
	}
	if !strings.HasPrefix(p.Skill, h.Home) || !strings.HasPrefix(p.MCP, h.Home) {
		t.Fatalf("wrote outside fake home: %+v", p)
	}
	assertSkill(t, p.Skill)
	b, err := os.ReadFile(p.MCP)
	if err != nil {
		t.Fatal(err)
	}
	if !host.HasMCPServer(b, host.MCPTOML, "kage") {
		t.Fatalf("mcp:\n%s", b)
	}
	if strings.Contains(string(b), "command = \"kage\"") == false || !strings.Contains(string(b), `args = ["mcp"]`) {
		t.Fatalf("mcp block:\n%s", b)
	}
}

func TestInstallNamedClients(t *testing.T) {
	h := installHost(t)
	cases := []struct {
		name  string
		skill string
		mcp   string
		kind  host.MCPKind
	}{
		{"grok", ".grok/skills/kage/SKILL.md", ".grok/config.toml", host.MCPTOML},
		{"claude", ".claude/skills/kage/SKILL.md", ".claude.json", host.MCPJSON},
		{"cursor", ".cursor/skills/kage/SKILL.md", ".cursor/mcp.json", host.MCPJSON},
		{"codex", ".codex/skills/kage/SKILL.md", ".codex/config.toml", host.MCPTOML},
	}
	for _, tc := range cases {
		out, errb, code := execCLI(h, "install", tc.name)
		if code != 0 {
			t.Fatalf("%s: exit %d %s", tc.name, code, errb)
		}
		p := decodeInstall(t, out)
		if p.Client != tc.name {
			t.Fatalf("%s client %s", tc.name, p.Client)
		}
		wantSkill := filepath.Join(h.Home, filepath.FromSlash(tc.skill))
		wantMCP := filepath.Join(h.Home, filepath.FromSlash(tc.mcp))
		if p.Skill != wantSkill || p.MCP != wantMCP {
			t.Fatalf("%s paths %+v want %s %s", tc.name, p, wantSkill, wantMCP)
		}
		assertSkill(t, p.Skill)
		b, err := os.ReadFile(p.MCP)
		if err != nil {
			t.Fatal(err)
		}
		if !host.HasMCPServer(b, tc.kind, "kage") {
			t.Fatalf("%s mcp missing:\n%s", tc.name, b)
		}
		assertMCPCommand(t, b, tc.kind)
	}
}

func assertMCPCommand(t *testing.T, data []byte, kind host.MCPKind) {
	t.Helper()
	switch kind {
	case host.MCPJSON:
		var root struct {
			Servers map[string]struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &root); err != nil {
			t.Fatal(err)
		}
		s := root.Servers["kage"]
		if s.Command != "kage" || len(s.Args) != 1 || s.Args[0] != "mcp" {
			t.Fatalf("json mcp %+v", s)
		}
	default:
		if !strings.Contains(string(data), `command = "kage"`) || !strings.Contains(string(data), `args = ["mcp"]`) {
			t.Fatalf("toml mcp:\n%s", data)
		}
	}
}

func TestInstallUsesDefaultClient(t *testing.T) {
	h := installHost(t)
	h.Client = "claude"
	out, errb, code := execCLI(h, "install")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	p := decodeInstall(t, out)
	if p.Client != "claude" {
		t.Fatalf("%+v", p)
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".claude", "skills", "kage", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".grok", "skills", "kage", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("must not install grok: %v", err)
	}
}

func TestUninstallRemovesSkillAndMCP(t *testing.T) {
	h := installHost(t)
	if _, _, code := execCLI(h, "install", "grok"); code != 0 {
		t.Fatal(code)
	}
	cfg := filepath.Join(h.Home, ".grok", "config.toml")
	existing := []byte("[ui]\nx = 1\n\n[mcp_servers.other]\ncommand = \"x\"\n")
	cur, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, append(existing, cur...), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errb, code := execCLI(h, "uninstall", "grok")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	p := decodeInstall(t, out)
	if p.Client != "grok" || !p.OK {
		t.Fatalf("%+v", p)
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".grok", "skills", "kage")); !os.IsNotExist(err) {
		t.Fatalf("skill dir remains: %v", err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if host.HasMCPServer(b, host.MCPTOML, "kage") {
		t.Fatalf("kage still registered:\n%s", b)
	}
	if !strings.Contains(string(b), "[ui]") || !strings.Contains(string(b), "[mcp_servers.other]") {
		t.Fatalf("lost other config:\n%s", b)
	}
}

func TestUninstallDefaultClient(t *testing.T) {
	h := installHost(t)
	h.Client = "cursor"
	if _, _, code := execCLI(h, "install"); code != 0 {
		t.Fatal("install")
	}
	if _, _, code := execCLI(h, "uninstall"); code != 0 {
		t.Fatal("uninstall")
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".cursor", "skills", "kage")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(h.Home, ".cursor", "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if host.HasMCPServer(b, host.MCPJSON, "kage") {
		t.Fatalf("%s", b)
	}
}

func TestInstallUnknownClientNoWrite(t *testing.T) {
	h := installHost(t)
	_, errb, code := execCLI(h, "install", "bob")
	if code == 0 || !strings.Contains(errb, "unknown client") {
		t.Fatalf("stderr %s code %d", errb, code)
	}
	if !strings.Contains(errb, "grok, claude, cursor, or codex") {
		t.Fatalf("hint: %s", errb)
	}
	ents, err := os.ReadDir(h.Home)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 0 {
		t.Fatalf("wrote on error: %v", ents)
	}
}

func TestInstallExtraArgs(t *testing.T) {
	h := installHost(t)
	_, errb, code := execCLI(h, "install", "grok", "claude")
	if code == 0 || !strings.Contains(errb, "unexpected arguments") {
		t.Fatalf("%d %s", code, errb)
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".grok")); !os.IsNotExist(err) {
		t.Fatal("must not write")
	}
}

func TestInstallNoHome(t *testing.T) {
	h := okHost()
	h.Home = ""
	h.Disk = nil
	_, errb, code := execCLI(h, "install")
	if code == 0 || !strings.Contains(errb, "home") {
		t.Fatalf("%d %s", code, errb)
	}
}

func TestInstallPreservesJSON(t *testing.T) {
	h := installHost(t)
	p := filepath.Join(h.Home, ".claude.json")
	if err := os.MkdirAll(h.Home, 0o700); err != nil {
		t.Fatal(err)
	}
	orig := []byte(`{"firstStartTime":"keep","mcpServers":{"other":{"command":"x"}}}`)
	if err := os.WriteFile(p, orig, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, errb, code := execCLI(h, "install", "claude"); code != 0 {
		t.Fatal(errb)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"firstStartTime": "keep"`) || !strings.Contains(string(b), `"other"`) {
		t.Fatalf("%s", b)
	}
	if !host.HasMCPServer(b, host.MCPJSON, "kage") {
		t.Fatalf("missing kage: %s", b)
	}
	bad := filepath.Join(h.Home, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(bad), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, errb, code := execCLI(h, "install", "cursor")
	if code == 0 || !strings.Contains(errb, "invalid JSON") {
		t.Fatalf("%d %s", code, errb)
	}
	got, err := os.ReadFile(bad)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{" {
		t.Fatalf("clobbered: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".cursor", "skills", "kage", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("must not write skill when MCP config is invalid")
	}
}

func TestDoctorReportsInstall(t *testing.T) {
	h := installHost(t)
	if err := os.MkdirAll(filepath.Join(h.Home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	out, errb, code := execCLI(h, "doctor")
	if code != 0 {
		t.Fatalf("exit %d %s", code, errb)
	}
	var r struct {
		DefaultClient string `json:"default_client"`
		Clients       []host.ClientStatus
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	if r.DefaultClient != "grok" {
		t.Fatalf("default %s", r.DefaultClient)
	}
	by := map[string]host.ClientStatus{}
	for _, c := range r.Clients {
		by[c.Name] = c
	}
	if g, ok := by["grok"]; !ok || g.Skill || g.MCP {
		t.Fatalf("pre: %+v", r.Clients)
	}
	if c, ok := by["codex"]; !ok || !c.ConfigDir || c.Skill {
		t.Fatalf("codex extra: %+v", r.Clients)
	}
	if _, _, code := execCLI(h, "install"); code != 0 {
		t.Fatal("install")
	}
	out, _, code = execCLI(h, "doctor")
	if code != 0 {
		t.Fatal(code)
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	by = map[string]host.ClientStatus{}
	for _, c := range r.Clients {
		by[c.Name] = c
	}
	g := by["grok"]
	if !g.Skill || !g.MCP {
		t.Fatalf("after install: %+v", r.Clients)
	}
	human, _, _ := execCLI(h, "doctor", "--human")
	if !strings.Contains(human, "client:grok") || !strings.Contains(human, "skill=true") || !strings.Contains(human, "mcp=true") {
		t.Fatalf("human: %s", human)
	}
	if !strings.Contains(human, "client:codex") {
		t.Fatalf("extra row: %s", human)
	}
}

func TestInstallHuman(t *testing.T) {
	h := installHost(t)
	out, errb, code := execCLI(h, "install", "grok", "--human")
	if code != 0 {
		t.Fatalf("%d %s", code, errb)
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("json: %s", out)
	}
	if !strings.Contains(out, "grok") || !strings.Contains(out, "SKILL.md") {
		t.Fatalf("%s", out)
	}
}

func TestUninstallLeavesOtherClients(t *testing.T) {
	h := installHost(t)
	if _, _, code := execCLI(h, "install", "grok"); code != 0 {
		t.Fatal("grok")
	}
	if _, _, code := execCLI(h, "install", "claude"); code != 0 {
		t.Fatal("claude")
	}
	if _, _, code := execCLI(h, "uninstall", "grok"); code != 0 {
		t.Fatal("uninstall")
	}
	if _, err := os.Stat(filepath.Join(h.Home, ".grok", "skills", "kage")); !os.IsNotExist(err) {
		t.Fatal("grok skill remains")
	}
	claudeSkill := filepath.Join(h.Home, ".claude", "skills", "kage", "SKILL.md")
	if _, err := os.Stat(claudeSkill); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(h.Home, ".claude.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !host.HasMCPServer(b, host.MCPJSON, "kage") {
		t.Fatalf("claude mcp removed:\n%s", b)
	}
	assertMCPCommand(t, b, host.MCPJSON)
}

func TestUninstallIdempotent(t *testing.T) {
	h := installHost(t)
	if _, _, code := execCLI(h, "uninstall", "grok"); code != 0 {
		t.Fatal("uninstall missing")
	}
	ents, err := os.ReadDir(h.Home)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 0 {
		t.Fatalf("uninstall wrote: %v", ents)
	}
}

func assertSkill(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != skill.Markdown {
		t.Fatalf("skill copy mismatch (%d vs %d bytes)", len(b), len(skill.Markdown))
	}
}
