package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/6abe/kage/internal/host"
)

func TestDirRuntimeAndFallback(t *testing.T) {
	h := &host.Fake{Environ: map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}}
	if got := Dir(h); got != "/run/user/1000/kage" {
		t.Fatalf("runtime dir: %s", got)
	}
	h.Environ = nil
	want := filepath.Join(os.TempDir(), fmt.Sprintf("kage-%d", os.Getuid()))
	if got := Dir(h); got != want {
		t.Fatalf("fallback: got %s want %s", got, want)
	}
}

func TestEnsureMode0700(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kage")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(dir); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o700 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
}
