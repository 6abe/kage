package see

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/6abe/kage/internal/capture"
	"github.com/6abe/kage/internal/host"
	"github.com/6abe/kage/internal/hypr"
)

const (
	CoordinateSpace = "global_compositor_pixels"
	DefaultMaxWidth = 1920
)

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

type Options struct {
	Path     string
	Monitor  string
	Window   string
	All      bool
	Annotate bool
	MaxWidth int
}

// AmbiguousError is a window query that matched more than one client.
type AmbiguousError struct {
	Query   string
	Matches []hypr.Window
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("ambiguous window match %q (%d clients)", e.Query, len(e.Matches))
}

func (e *AmbiguousError) Unwrap() error { return hypr.ErrAmbiguous }

type target struct {
	grim   []string
	origin hypr.Geometry
	mon    Monitor
}

func Capture(h host.Host, opt Options) (Snapshot, error) {
	if n := opt.targets(); n > 1 {
		return Snapshot{}, fmt.Errorf("--monitor, --window, and --all are mutually exclusive")
	}
	if opt.MaxWidth <= 0 {
		opt.MaxWidth = DefaultMaxWidth
	}
	mons, err := hypr.ListMonitors(h)
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
	tgt, err := resolveTarget(mons, wins, opt)
	if err != nil {
		return Snapshot{}, err
	}
	path, err := resolvePath(h, opt.Path, id)
	if err != nil {
		return Snapshot{}, err
	}
	args := append(append([]string{}, tgt.grim...), path)
	if err := h.Grim(args...); err != nil {
		return Snapshot{}, err
	}
	width, height, err := downscaleFile(path, opt.MaxWidth)
	if err != nil {
		return Snapshot{}, err
	}
	snap := Snapshot{
		OK:              true,
		SnapshotID:      id,
		Path:            path,
		Width:           width,
		Height:          height,
		Monitor:         tgt.mon,
		Focused:         snapshotFocused(wins),
		Windows:         snapshotWindows(wins),
		CoordinateSpace: CoordinateSpace,
	}
	if opt.Annotate {
		if err := annotate(path, snap.Windows, tgt.origin.X, tgt.origin.Y, tgt.origin.Width, tgt.origin.Height); err != nil {
			return Snapshot{}, err
		}
	}
	if err := persist(h, snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

func (o Options) targets() int {
	n := 0
	if o.All {
		n++
	}
	if o.Monitor != "" {
		n++
	}
	if o.Window != "" {
		n++
	}
	return n
}

func resolveTarget(mons []hypr.Monitor, wins []hypr.Window, opt Options) (target, error) {
	switch {
	case opt.All:
		box, err := hypr.BoundingBox(mons)
		if err != nil {
			return target{}, err
		}
		return target{
			grim:   []string{"-g", box.Grim()},
			origin: box,
			mon: Monitor{
				Name:   "all",
				X:      box.X,
				Y:      box.Y,
				Width:  box.Width,
				Height: box.Height,
				Scale:  1,
			},
		}, nil
	case opt.Window != "":
		w, matches, err := hypr.MatchWindow(wins, opt.Window)
		if errors.Is(err, hypr.ErrAmbiguous) {
			return target{}, &AmbiguousError{Query: opt.Window, Matches: matches}
		}
		if errors.Is(err, hypr.ErrNotFound) {
			return target{}, fmt.Errorf("no window matches %q", opt.Window)
		}
		if err != nil {
			return target{}, err
		}
		g := w.Geometry
		if g.Width <= 0 || g.Height <= 0 {
			return target{}, fmt.Errorf("window %s has empty geometry", w.Address)
		}
		mon := monitorForWindow(mons, w)
		return target{
			grim:   []string{"-g", g.Grim()},
			origin: g,
			mon:    mon,
		}, nil
	case opt.Monitor != "":
		m, err := hypr.MonitorByName(mons, opt.Monitor)
		if err != nil {
			return target{}, err
		}
		return monitorTarget(m), nil
	default:
		m, err := hypr.FocusedMonitor(mons)
		if err != nil {
			return target{}, err
		}
		return monitorTarget(m), nil
	}
}

func monitorTarget(m hypr.Monitor) target {
	return target{
		grim:   []string{"-o", m.Name},
		origin: m.Geometry(),
		mon: Monitor{
			Name:   m.Name,
			X:      m.X,
			Y:      m.Y,
			Width:  m.Width,
			Height: m.Height,
			Scale:  m.Scale,
		},
	}
}

func monitorForWindow(mons []hypr.Monitor, w hypr.Window) Monitor {
	for _, m := range mons {
		if m.Name == w.Monitor {
			return Monitor{
				Name:   m.Name,
				X:      m.X,
				Y:      m.Y,
				Width:  m.Width,
				Height: m.Height,
				Scale:  m.Scale,
			}
		}
	}
	g := w.Geometry
	return Monitor{Name: w.Monitor, X: g.X, Y: g.Y, Width: g.Width, Height: g.Height, Scale: 1}
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
