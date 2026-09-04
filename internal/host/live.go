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

func (Live) Ydotool(args ...string) error {
	cmd := exec.Command("ydotool", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if ydotoolDaemonDown(msg, err) {
			return fmt.Errorf("ydotool not running")
		}
		if msg == "" {
			return fmt.Errorf("ydotool: %w", err)
		}
		return fmt.Errorf("ydotool: %w: %s", err, msg)
	}
	return nil
}

func ydotoolDaemonDown(msg string, err error) bool {
	s := strings.ToLower(msg + " " + err.Error())
	return strings.Contains(s, "connect") ||
		strings.Contains(s, "socket") ||
		strings.Contains(s, "no such file") ||
		strings.Contains(s, "ydotoold")
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
	return ScanClients(l)
}

func (Live) HomeDir() (string, error) {
	return os.UserHomeDir()
}

func (Live) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (Live) RemoveAll(path string) error {
	return os.RemoveAll(path)
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
