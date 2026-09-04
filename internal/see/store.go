package see

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/6abe/kage/internal/capture"
	"github.com/6abe/kage/internal/host"
)

var snapshotIDRe = regexp.MustCompile(`^kage_\d{8}_\d{6}_[0-9a-f]{4}$`)

func metaName(id string) string {
	return id + ".json"
}

func metaPath(dir, id string) (string, error) {
	if !snapshotIDRe.MatchString(id) {
		return "", fmt.Errorf("invalid snapshot id %q", id)
	}
	return filepath.Join(dir, metaName(id)), nil
}

func persist(h host.Host, snap Snapshot) error {
	dir := capture.Dir(h)
	if err := capture.Ensure(dir); err != nil {
		return err
	}
	lookup, err := metaPath(dir, snap.SnapshotID)
	if err != nil {
		return err
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	if err := h.WriteFile(lookup, b); err != nil {
		return err
	}
	if snap.Path == "" {
		return nil
	}
	side, err := metaPath(filepath.Dir(snap.Path), snap.SnapshotID)
	if err != nil {
		return err
	}
	if side == lookup {
		return nil
	}
	return h.WriteFile(side, b)
}

// Load reads snapshot metadata saved next to a see PNG (keyed by snapshot_id).
func Load(h host.Host, id string) (Snapshot, error) {
	path, err := metaPath(capture.Dir(h), id)
	if err != nil {
		return Snapshot{}, err
	}
	b, err := h.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot %q: %w", id, err)
	}
	var snap Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot %q: %w", id, err)
	}
	return snap, nil
}
