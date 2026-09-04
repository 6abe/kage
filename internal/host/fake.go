package host

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Fake is an in-memory Host for tests. It never talks to a compositor.
type Fake struct {
	JSON            map[string][]byte
	HyprctlErr      error
	Environ         map[string]string
	Paths           map[string]string
	Lookups         []string
	Probe           error
	GrimArgs        []string
	Client          string
	Disk            []ClientStatus
	Home            string
	Allow           bool
	Logs            []string
	Dispatch        [][]string
	DispatchErr     error
	WtypeCalls      [][]string
	WtypeErr        error
	YdotoolCalls    [][]string
	YdotoolErr      error
	SendShortcutErr error
	// ImageSize, if set, is the PNG grim writes. Otherwise -g WxH, else 8x4.
	ImageSize image.Point
}

func (f *Fake) HyprctlJSON(resource string) ([]byte, error) {
	if f.HyprctlErr != nil {
		return nil, f.HyprctlErr
	}
	b, ok := f.JSON[resource]
	if !ok {
		return nil, fmt.Errorf("hyprctl -j %s: no fake payload", resource)
	}
	return b, nil
}

func (f *Fake) Env(key string) string {
	if f.Environ == nil {
		return ""
	}
	return f.Environ[key]
}

func (f *Fake) HyprctlDispatch(args ...string) error {
	f.Dispatch = append(f.Dispatch, append([]string(nil), args...))
	if len(args) > 0 && strings.Contains(args[0], "send_shortcut") && f.SendShortcutErr != nil {
		return f.SendShortcutErr
	}
	return f.DispatchErr
}

func (f *Fake) LookPath(name string) (string, error) {
	f.Lookups = append(f.Lookups, name)
	if f.Paths != nil {
		if p, ok := f.Paths[name]; ok && p != "" {
			return p, nil
		}
	}
	return "", fmt.Errorf("executable file not found in $PATH")
}

func (f *Fake) Wtype(args ...string) error {
	f.WtypeCalls = append(f.WtypeCalls, append([]string(nil), args...))
	return f.WtypeErr
}

func (f *Fake) Ydotool(args ...string) error {
	f.YdotoolCalls = append(f.YdotoolCalls, append([]string(nil), args...))
	return f.YdotoolErr
}

func (f *Fake) AllowInput() bool {
	return f.Allow
}

func (f *Fake) Log(line string) {
	f.Logs = append(f.Logs, line)
}

func (f *Fake) Grim(args ...string) error {
	f.GrimArgs = append([]string(nil), args...)
	if f.Probe != nil {
		return f.Probe
	}
	if len(args) == 0 {
		return fmt.Errorf("grim: missing output")
	}
	out := args[len(args)-1]
	if strings.HasPrefix(out, "-") {
		return fmt.Errorf("grim: missing output")
	}
	w, h := 8, 4
	if f.ImageSize.X > 0 && f.ImageSize.Y > 0 {
		w, h = f.ImageSize.X, f.ImageSize.Y
	} else {
		for i, a := range args {
			if a == "-g" && i+1 < len(args) {
				if gw, gh, ok := parseGrimSize(args[i+1]); ok {
					w, h = gw, gh
				}
			}
		}
	}
	return writeFakePNG(out, w, h)
}

func parseGrimSize(g string) (w, h int, ok bool) {
	var x, y int
	n, err := fmt.Sscanf(g, "%d,%d %dx%d", &x, &y, &w, &h)
	if err != nil || n != 4 || w < 1 || h < 1 {
		return 0, 0, false
	}
	return w, h, true
}

func writeFakePNG(path string, w, h int) error {
	if w < 1 {
		w = 8
	}
	if h < 1 {
		h = 4
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func (f *Fake) WriteFile(path string, data []byte) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o600)
}

func (f *Fake) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *Fake) DefaultClient() string {
	if f.Client == "" {
		return "grok"
	}
	return f.Client
}

func (f *Fake) ClientsOnDisk() []ClientStatus {
	if f.Home != "" {
		return ScanClients(f)
	}
	if len(f.Disk) > 0 {
		return f.Disk
	}
	return []ClientStatus{{Name: f.DefaultClient()}}
}

func (f *Fake) HomeDir() (string, error) {
	if f.Home == "" {
		return "", fmt.Errorf("home directory not set")
	}
	return f.Home, nil
}

func (f *Fake) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *Fake) RemoveAll(path string) error {
	if f.Home == "" {
		return fmt.Errorf("home directory not set")
	}
	if !underDir(f.Home, path) {
		return fmt.Errorf("remove outside fake home")
	}
	return os.RemoveAll(path)
}

func underDir(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	sep := string(os.PathSeparator)
	return path == root || strings.HasPrefix(path, root+sep)
}
