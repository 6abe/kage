package host

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
)

const MCPServerName = "kage"

type MCPKind int

const (
	MCPTOML MCPKind = iota
	MCPJSON
)

// ClientSpec is one agent client's skill directory and MCP config files.
type ClientSpec struct {
	Name     string
	Dir      string
	SkillDir string
	MCPFiles []string
	MCPKind  MCPKind
}

func KnownClients() []ClientSpec {
	return []ClientSpec{
		{Name: "grok", Dir: ".grok", SkillDir: ".grok/skills/kage", MCPFiles: []string{".grok/config.toml"}, MCPKind: MCPTOML},
		{Name: "claude", Dir: ".claude", SkillDir: ".claude/skills/kage", MCPFiles: []string{".claude.json", ".claude/settings.json"}, MCPKind: MCPJSON},
		{Name: "cursor", Dir: ".cursor", SkillDir: ".cursor/skills/kage", MCPFiles: []string{".cursor/mcp.json"}, MCPKind: MCPJSON},
		{Name: "codex", Dir: ".codex", SkillDir: ".codex/skills/kage", MCPFiles: []string{".codex/config.toml", ".codex/mcp.json"}, MCPKind: MCPTOML},
	}
}

// LookupClient matches a known client name (case-insensitive).
func LookupClient(name string) (ClientSpec, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, c := range KnownClients() {
		if c.Name == want {
			return c, true
		}
	}
	return ClientSpec{}, false
}

// ScanClients reports skill/MCP presence under HomeDir.
func ScanClients(h Host) []ClientStatus {
	def := h.DefaultClient()
	if def == "" {
		def = "grok"
	}
	home, err := h.HomeDir()
	if err != nil {
		home = ""
	}
	var out []ClientStatus
	seen := map[string]bool{}
	for _, spec := range KnownClients() {
		st := ClientStatus{Name: spec.Name}
		if home != "" {
			st.ConfigDir = h.Exists(filepath.Join(home, spec.Dir))
			st.Skill = h.Exists(filepath.Join(home, spec.SkillDir, "SKILL.md"))
			st.MCP = mcpOnDisk(h, home, spec)
		}
		if spec.Name == def || st.ConfigDir || st.Skill || st.MCP {
			out = append(out, st)
			seen[spec.Name] = true
		}
	}
	if !seen[def] {
		out = append([]ClientStatus{{Name: def}}, out...)
	}
	return out
}

func mcpOnDisk(h Host, home string, spec ClientSpec) bool {
	for _, rel := range spec.MCPFiles {
		b, err := h.ReadFile(filepath.Join(home, rel))
		if err != nil {
			continue
		}
		if HasMCPServer(b, spec.MCPKind, MCPServerName) {
			return true
		}
	}
	return false
}

// HasMCPServer reports whether data registers the named stdio MCP server.
func HasMCPServer(data []byte, kind MCPKind, name string) bool {
	switch kind {
	case MCPJSON:
		return jsonHasMCP(data, name)
	default:
		return tomlHasTable(data, "mcp_servers."+name)
	}
}

func jsonHasMCP(data []byte, name string) bool {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return false
	}
	servers, _ := root["mcpServers"].(map[string]any)
	if servers == nil {
		return false
	}
	_, ok := servers[name]
	return ok
}

func tomlHasTable(data []byte, name string) bool {
	for _, line := range bytes.Split(data, []byte("\n")) {
		s := strings.TrimSpace(string(line))
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if tomlTableName(s) == name || strings.HasPrefix(tomlTableName(s), name+".") {
			return true
		}
	}
	return false
}

func tomlTableName(line string) string {
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
		return ""
	}
	return strings.TrimSpace(line[1 : len(line)-1])
}
