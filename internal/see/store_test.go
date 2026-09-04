package see

import (
	"testing"

	"github.com/6abe/kage/internal/host"
)

func TestLatestAndWindowByID(t *testing.T) {
	h := &host.Fake{Environ: map[string]string{"XDG_RUNTIME_DIR": t.TempDir()}}
	first := Snapshot{OK: true, SnapshotID: "kage_20260904_000001_aaaa", Windows: []Window{{ID: 1, Address: "0x1", Size: [2]int{10, 10}}}}
	second := Snapshot{OK: true, SnapshotID: "kage_20260904_000002_bbbb", Windows: []Window{
		{ID: 1, Address: "0x1", At: [2]int{0, 0}, Size: [2]int{100, 40}},
		{ID: 2, Address: "0x2", At: [2]int{10, 20}, Size: [2]int{8, 6}},
	}}
	if err := persist(h, first); err != nil {
		t.Fatal(err)
	}
	if err := persist(h, second); err != nil {
		t.Fatal(err)
	}
	got, err := Latest(h)
	if err != nil || got.SnapshotID != second.SnapshotID {
		t.Fatalf("%+v %v", got, err)
	}
	w, err := WindowByID(got, 2)
	if err != nil || w.Address != "0x2" {
		t.Fatalf("%+v %v", w, err)
	}
	x, y, err := w.Center()
	if err != nil || x != 14 || y != 23 {
		t.Fatalf("center %d %d %v", x, y, err)
	}
	if _, err := WindowByID(got, 9); err == nil {
		t.Fatal("missing id")
	}
	if _, err := Load(h, "../etc/passwd"); err == nil {
		t.Fatal("traversal")
	}
	empty := Window{Address: "0x3", Size: [2]int{0, 10}}
	if _, _, err := empty.Center(); err == nil {
		t.Fatal("empty geom")
	}
	h2 := &host.Fake{Environ: map[string]string{"XDG_RUNTIME_DIR": t.TempDir()}}
	if _, err := Latest(h2); err == nil {
		t.Fatal("no snapshot")
	}
}
