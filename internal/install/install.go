package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/skill"
)

var ErrUnknownClient = errors.New("unknown client")

type Result struct {
	OK     bool   `json:"ok"`
	Client string `json:"client"`
	Skill  string `json:"skill"`
	MCP    string `json:"mcp"`
}

func Resolve(h host.Host, name string) (host.ClientSpec, error) {
	if name == "" {
		name = h.DefaultClient()
	}
	if name == "" {
		name = "grok"
	}
	spec, ok := host.LookupClient(name)
	if !ok {
		return host.ClientSpec{}, fmt.Errorf("%w: %s", ErrUnknownClient, name)
	}
	return spec, nil
}

func Install(h host.Host, name string) (Result, error) {
	spec, err := Resolve(h, name)
	if err != nil {
		return Result{}, err
	}
	home, err := requireHome(h)
	if err != nil {
		return Result{}, err
	}
	mcpPath, next, err := planMCPWrite(h, home, spec)
	if err != nil {
		return Result{}, err
	}
	skillPath := filepath.Join(home, spec.SkillDir, "SKILL.md")
	if err := h.WriteFile(skillPath, []byte(skill.Markdown)); err != nil {
		return Result{}, err
	}
	if err := h.WriteFile(mcpPath, next); err != nil {
		return Result{}, err
	}
	return Result{OK: true, Client: spec.Name, Skill: skillPath, MCP: mcpPath}, nil
}

func Uninstall(h host.Host, name string) (Result, error) {
	spec, err := Resolve(h, name)
	if err != nil {
		return Result{}, err
	}
	home, err := requireHome(h)
	if err != nil {
		return Result{}, err
	}
	writes, mcpPath, err := planMCPRemove(h, home, spec)
	if err != nil {
		return Result{}, err
	}
	for _, w := range writes {
		if err := h.WriteFile(w.path, w.data); err != nil {
			return Result{}, err
		}
	}
	skillDir := filepath.Join(home, spec.SkillDir)
	if err := h.RemoveAll(skillDir); err != nil {
		return Result{}, err
	}
	return Result{
		OK:     true,
		Client: spec.Name,
		Skill:  filepath.Join(skillDir, "SKILL.md"),
		MCP:    mcpPath,
	}, nil
}

func requireHome(h host.Host) (string, error) {
	home, err := h.HomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", fmt.Errorf("cannot determine home directory")
	}
	return home, nil
}

type mcpWrite struct {
	path string
	data []byte
}

func planMCPWrite(h host.Host, home string, spec host.ClientSpec) (string, []byte, error) {
	if len(spec.MCPFiles) == 0 {
		return "", nil, fmt.Errorf("no MCP config path for %s", spec.Name)
	}
	path := filepath.Join(home, spec.MCPFiles[0])
	cur, err := readOptional(h, path)
	if err != nil {
		return "", nil, err
	}
	next, err := upsertMCP(cur, spec.MCPKind, host.MCPServerName, "kage", []string{"mcp"})
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", path, err)
	}
	return path, next, nil
}

func planMCPRemove(h host.Host, home string, spec host.ClientSpec) ([]mcpWrite, string, error) {
	primary := ""
	if len(spec.MCPFiles) > 0 {
		primary = filepath.Join(home, spec.MCPFiles[0])
	}
	var writes []mcpWrite
	for _, rel := range spec.MCPFiles {
		path := filepath.Join(home, rel)
		cur, err := readOptional(h, path)
		if err != nil {
			return nil, primary, err
		}
		if cur == nil || !host.HasMCPServer(cur, spec.MCPKind, host.MCPServerName) {
			continue
		}
		next, err := stripMCP(cur, spec.MCPKind, host.MCPServerName)
		if err != nil {
			return nil, primary, fmt.Errorf("%s: %w", path, err)
		}
		writes = append(writes, mcpWrite{path: path, data: next})
	}
	return writes, primary, nil
}

func readOptional(h host.Host, path string) ([]byte, error) {
	b, err := h.ReadFile(path)
	if err == nil {
		return b, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}
