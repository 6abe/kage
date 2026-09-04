package capture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPNGSizeRejectsNonPNG(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.png")
	if err := os.WriteFile(p, []byte("not a png"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := PNGSize(p); err == nil {
		t.Fatal("expected error")
	}
}
