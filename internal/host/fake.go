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
	JSON        map[string][]byte
	HyprctlErr  error
	Environ     map[string]string
	Paths       map[string]string
	Lookups     []string
	Probe       error
	GrimArgs    []string
	Client      string
	Disk        []ClientStatus
	Allow       bool
	Logs        []string
	Dispatch    [][]string
	DispatchErr error
	WtypeCalls  [][]string
	WtypeErr    error
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
	return writeFakePNG(out)
}

func writeFakePNG(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, 8, 4))
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

func (f *Fake) DefaultClient() string {
	if f.Client == "" {
		return "grok"
	}
	return f.Client
}

func (f *Fake) ClientsOnDisk() []ClientStatus {
	if len(f.Disk) > 0 {
		return f.Disk
	}
	return []ClientStatus{{Name: f.DefaultClient()}}
}
