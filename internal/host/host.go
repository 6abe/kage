package host

// Host is the session seam: compositor, capture, input tools, and client-config disk.
// Production talks to the live machine. Tests inject a fake.
type Host interface {
	// HyprctlJSON runs `hyprctl -j <resource>` (monitors, clients, activewindow).
	HyprctlJSON(resource string) ([]byte, error)
	// Env returns an environment variable (empty if unset).
	Env(key string) string
	// LookPath resolves an executable on PATH. Missing tools return an error.
	LookPath(name string) (string, error)
	// CaptureProbe actually runs grim. Tests stub this.
	CaptureProbe() error
	// DefaultClient is "grok" unless config says otherwise.
	DefaultClient() string
	// ClientsOnDisk reports skill/MCP presence. Always includes DefaultClient.
	// Extra clients appear when their config directories exist.
	ClientsOnDisk() []ClientStatus
}

// ClientStatus is one agent client's skill + MCP files on disk.
type ClientStatus struct {
	Name      string `json:"name"`
	Skill     bool   `json:"skill"`
	MCP       bool   `json:"mcp"`
	ConfigDir bool   `json:"config_dir"`
}

// ToolHint is the omarchy install line for a missing binary.
func ToolHint(bin string) string {
	pkg := bin
	switch bin {
	case "hyprctl":
		pkg = "hyprland"
	case "wl-copy":
		pkg = "wl-clipboard"
	}
	return "omarchy pkg add " + pkg
}
