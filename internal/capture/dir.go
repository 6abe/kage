package capture

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/6abe/kage/internal/host"
)

// Dir is $XDG_RUNTIME_DIR/kage, or /tmp/kage-$UID when the runtime dir is unset.
func Dir(h host.Host) string {
	if d := h.Env("XDG_RUNTIME_DIR"); d != "" {
		return filepath.Join(d, "kage")
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("kage-%d", os.Getuid()))
}

// Ensure creates dir at mode 0700. MkdirAll honors umask, so chmod is required.
func Ensure(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}
