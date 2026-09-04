package host

import "fmt"

// Fake is an in-memory Host for tests. It never talks to a compositor.
type Fake struct {
	JSON       map[string][]byte
	HyprctlErr error
	Environ    map[string]string
	Paths      map[string]string
	Probe      error
	Probed     bool
	Client     string
	Disk       []ClientStatus
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

func (f *Fake) LookPath(name string) (string, error) {
	if f.Paths != nil {
		if p, ok := f.Paths[name]; ok && p != "" {
			return p, nil
		}
	}
	return "", fmt.Errorf("executable file not found in $PATH")
}

func (f *Fake) CaptureProbe() error {
	f.Probed = true
	return f.Probe
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
