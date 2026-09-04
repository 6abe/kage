package see

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/6abe/kage/internal/capture"
	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
)

const CoordinateSpace = "global_compositor_pixels"

type Monitor struct {
	Name   string  `json:"name"`
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Scale  float64 `json:"scale"`
}

type Window struct {
	ID        int    `json:"id"`
	Address   string `json:"address"`
	Class     string `json:"class"`
	Title     string `json:"title"`
	At        [2]int `json:"at"`
	Size      [2]int `json:"size"`
	Workspace int    `json:"workspace"`
	Monitor   string `json:"monitor"`
	Floating  bool   `json:"floating"`
	Mapped    bool   `json:"mapped"`
	Focus     bool   `json:"focus"`
}

type Focused struct {
	Address   string `json:"address"`
	Class     string `json:"class"`
	Title     string `json:"title"`
	At        [2]int `json:"at"`
	Size      [2]int `json:"size"`
	Workspace int    `json:"workspace"`
	PID       int    `json:"pid"`
}

type Snapshot struct {
	OK              bool     `json:"ok"`
	SnapshotID      string   `json:"snapshot_id"`
	Path            string   `json:"path"`
	Width           int      `json:"width"`
	Height          int      `json:"height"`
	Monitor         Monitor  `json:"monitor"`
	Focused         *Focused `json:"focused"`
	Windows         []Window `json:"windows"`
	CoordinateSpace string   `json:"coordinate_space"`
}

func Capture(h host.Host, outPath string) (Snapshot, error) {
	mons, err := hypr.ListMonitors(h)
	if err != nil {
		return Snapshot{}, err
	}
	mon, err := hypr.FocusedMonitor(mons)
	if err != nil {
		return Snapshot{}, err
	}
	wins, err := hypr.ListWindows(h)
	if err != nil {
		return Snapshot{}, err
	}
	id, err := newSnapshotID(time.Now())
	if err != nil {
		return Snapshot{}, err
	}
	path, err := resolvePath(h, outPath, id)
	if err != nil {
		return Snapshot{}, err
	}
	if err := h.Grim("-o", mon.Name, path); err != nil {
		return Snapshot{}, err
	}
	width, height, err := capture.PNGSize(path)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		OK:         true,
		SnapshotID: id,
		Path:       path,
		Width:      width,
		Height:     height,
		Monitor: Monitor{
			Name:   mon.Name,
			X:      mon.X,
			Y:      mon.Y,
			Width:  mon.Width,
			Height: mon.Height,
			Scale:  mon.Scale,
		},
		Focused:         snapshotFocused(wins),
		Windows:         snapshotWindows(wins),
		CoordinateSpace: CoordinateSpace,
	}, nil
}

func resolvePath(h host.Host, outPath, id string) (string, error) {
	if outPath == "" {
		dir := capture.Dir(h)
		if err := capture.Ensure(dir); err != nil {
			return "", err
		}
		outPath = filepath.Join(dir, id+".png")
	} else if dir := filepath.Dir(outPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(outPath)
	if err != nil {
		return outPath, nil
	}
	return abs, nil
}

func newSnapshotID(now time.Time) (string, error) {
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("snapshot id: %w", err)
	}
	return fmt.Sprintf("kage_%s_%02x%02x", now.Format("20060102_150405"), b[0], b[1]), nil
}

func snapshotWindows(ws []hypr.Window) []Window {
	out := make([]Window, 0, len(ws))
	for i, w := range ws {
		out = append(out, Window{
			ID:        i + 1,
			Address:   w.Address,
			Class:     w.Class,
			Title:     w.Title,
			At:        [2]int{w.Geometry.X, w.Geometry.Y},
			Size:      [2]int{w.Geometry.Width, w.Geometry.Height},
			Workspace: w.Workspace,
			Monitor:   w.Monitor,
			Floating:  w.Floating,
			Mapped:    w.Mapped,
			Focus:     w.Focus,
		})
	}
	return out
}

func snapshotFocused(ws []hypr.Window) *Focused {
	for _, w := range ws {
		if !w.Focus {
			continue
		}
		return &Focused{
			Address:   w.Address,
			Class:     w.Class,
			Title:     w.Title,
			At:        [2]int{w.Geometry.X, w.Geometry.Y},
			Size:      [2]int{w.Geometry.Width, w.Geometry.Height},
			Workspace: w.Workspace,
			PID:       w.Pid,
		}
	}
	return nil
}
