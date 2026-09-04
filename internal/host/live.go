package host

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Live is the production Host. It shells out; it does not reimplement Wayland.
type Live struct{}

func (Live) HyprctlJSON(resource string) ([]byte, error) {
	cmd := exec.Command("hyprctl", "-j", resource)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("hyprctl -j %s: %w: %s", resource, err, bytes.TrimSpace(ee.Stderr))
		}
		return nil, fmt.Errorf("hyprctl -j %s: %w", resource, err)
	}
	return out, nil
}

func (Live) Env(key string) string {
	return os.Getenv(key)
}

func (Live) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (Live) Grim(args ...string) error {
	cmd := exec.Command("grim", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("grim: %w", err)
		}
		return fmt.Errorf("grim: %w: %s", err, msg)
	}
	return nil
}

func (Live) HyprctlDispatch(args ...string) error {
	cmd := exec.Command("hyprctl", append([]string{"dispatch"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("hyprctl dispatch: %w", err)
		}
		return fmt.Errorf("hyprctl dispatch: %w: %s", err, msg)
	}
	return nil
}

func (Live) Wtype(args ...string) error {
	cmd := exec.Command("wtype", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("wtype: %w", err)
		}
		return fmt.Errorf("wtype: %w: %s", err, msg)
	}
	return nil
}

func (Live) AllowInput() bool {
	return ParseConfig(readUserConfig()).AllowInput
}

func (Live) Log(line string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".local", "state", "kage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "kage.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintln(f, line)
	_ = f.Close()
}

func (Live) DefaultClient() string {
	return ParseDefaultClient(readUserConfig())
}

func readUserConfig() []byte {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(home, ".config", "kage", "config.toml"))
	if err != nil {
		return nil
	}
	return b
}

func (l Live) ClientsOnDisk() []ClientStatus {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	def := l.DefaultClient()
	known := []struct {
		name      string
		dir       string
		skill     string
		mcpFiles  []string
		mcpNeedle string
	}{
		{"grok", ".grok", ".grok/skills/kage", []string{".grok/config.toml"}, "kage"},
		{"claude", ".claude", ".claude/skills/kage", []string{".claude.json", ".claude/settings.json"}, "kage"},
		{"cursor", ".cursor", ".cursor/skills/kage", []string{".cursor/mcp.json"}, "kage"},
		{"codex", ".codex", ".codex/skills/kage", []string{".codex/config.toml", ".codex/mcp.json"}, "kage"},
	}
	var out []ClientStatus
	seen := map[string]bool{}
	for _, k := range known {
		st := ClientStatus{Name: k.name}
		if home != "" {
			st.ConfigDir = exists(filepath.Join(home, k.dir))
			st.Skill = exists(filepath.Join(home, k.skill))
			for _, rel := range k.mcpFiles {
				p := filepath.Join(home, rel)
				if b, err := os.ReadFile(p); err == nil && bytes.Contains(bytes.ToLower(b), []byte(k.mcpNeedle)) {
					st.MCP = true
					break
				}
			}
		}
		if k.name == def || st.ConfigDir || st.Skill || st.MCP {
			out = append(out, st)
			seen[k.name] = true
		}
	}
	if !seen[def] {
		out = append([]ClientStatus{{Name: def}}, out...)
	}
	return out
}

// FileConfig is the parsed ~/.config/kage/config.toml.
type FileConfig struct {
	DefaultClient string
	AllowInput    bool
}

// ParseConfig reads default_client and allow_input from config.toml bytes.
func ParseConfig(data []byte) FileConfig {
	cfg := FileConfig{DefaultClient: "grok"}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v := strings.Trim(strings.TrimSpace(rest), `"'`)
		switch strings.TrimSpace(key) {
		case "default_client":
			if v != "" {
				cfg.DefaultClient = v
			}
		case "allow_input":
			cfg.AllowInput = v == "true" || v == "1"
		}
	}
	return cfg
}

// ParseDefaultClient reads default_client from config.toml bytes.
func ParseDefaultClient(data []byte) string {
	return ParseConfig(data).DefaultClient
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (Live) WriteFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o600)
}

func (Live) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
