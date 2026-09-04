package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/6abe/kage/internal/host"
)

func upsertMCP(data []byte, kind host.MCPKind, name, command string, args []string) ([]byte, error) {
	switch kind {
	case host.MCPJSON:
		return upsertJSON(data, name, command, args)
	default:
		return upsertTOML(data, name, command, args), nil
	}
}

func stripMCP(data []byte, kind host.MCPKind, name string) ([]byte, error) {
	switch kind {
	case host.MCPJSON:
		return stripJSON(data, name)
	default:
		return stripTOML(data, name), nil
	}
}

func upsertTOML(data []byte, name, command string, args []string) []byte {
	body := bytes.TrimRight(stripTOML(data, name), "\n")
	block := formatTOML(name, command, args)
	if len(body) == 0 {
		return block
	}
	return append(append(body, '\n', '\n'), block...)
}

func stripTOML(data []byte, name string) []byte {
	table := "mcp_servers." + name
	var keep []string
	skip := false
	for _, line := range strings.Split(string(data), "\n") {
		if hdr := tomlHeader(line); hdr != "" {
			skip = hdr == table || strings.HasPrefix(hdr, table+".")
		}
		if skip {
			continue
		}
		keep = append(keep, line)
	}
	out := strings.TrimRight(strings.Join(keep, "\n"), "\n")
	if out == "" {
		return []byte{}
	}
	return []byte(out + "\n")
}

func tomlHeader(line string) string {
	s := strings.TrimSpace(line)
	if strings.HasPrefix(s, "#") {
		return ""
	}
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return ""
	}
	return strings.TrimSpace(s[1 : len(s)-1])
}

func formatTOML(name, command string, args []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "[mcp_servers.%s]\n", name)
	fmt.Fprintf(&b, "command = %q\n", command)
	b.WriteString("args = [")
	for i, a := range args {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", a)
	}
	b.WriteString("]\n")
	b.WriteString("enabled = true\n")
	return []byte(b.String())
}

func upsertJSON(data []byte, name, command string, args []string) ([]byte, error) {
	root, err := parseJSONObject(data)
	if err != nil {
		return nil, err
	}
	servers, err := mcpServers(root, true)
	if err != nil {
		return nil, err
	}
	entry := map[string]any{"command": command}
	if len(args) > 0 {
		arr := make([]any, len(args))
		for i, a := range args {
			arr[i] = a
		}
		entry["args"] = arr
	}
	servers[name] = entry
	root["mcpServers"] = servers
	return marshalJSON(root)
}

func stripJSON(data []byte, name string) ([]byte, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return []byte{}, nil
	}
	root, err := parseJSONObject(data)
	if err != nil {
		return nil, err
	}
	servers, err := mcpServers(root, false)
	if err != nil {
		return nil, err
	}
	if servers != nil {
		delete(servers, name)
		root["mcpServers"] = servers
	}
	return marshalJSON(root)
}

func parseJSONObject(data []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if root == nil {
		return map[string]any{}, nil
	}
	return root, nil
}

func mcpServers(root map[string]any, create bool) (map[string]any, error) {
	raw, ok := root["mcpServers"]
	if !ok || raw == nil {
		if create {
			return map[string]any{}, nil
		}
		return nil, nil
	}
	servers, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mcpServers is not an object")
	}
	return servers, nil
}

func marshalJSON(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
