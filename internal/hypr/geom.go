package hypr

import (
	"fmt"
	"strings"
)

// GrimGeom is grim's slurp-style region: "x,y widthxheight" (not "x,y,w,h").
func GrimGeom(x, y, w, h int) string {
	return fmt.Sprintf("%d,%d %dx%d", x, y, w, h)
}

func (g Geometry) Grim() string {
	return GrimGeom(g.X, g.Y, g.Width, g.Height)
}

func (m Monitor) Geometry() Geometry {
	return Geometry{X: m.X, Y: m.Y, Width: m.Width, Height: m.Height}
}

// BoundingBox is the compositor union of every monitor (for see --all).
func BoundingBox(mons []Monitor) (Geometry, error) {
	if len(mons) == 0 {
		return Geometry{}, fmt.Errorf("no monitors")
	}
	minX, minY := mons[0].X, mons[0].Y
	maxX := mons[0].X + mons[0].Width
	maxY := mons[0].Y + mons[0].Height
	for _, m := range mons[1:] {
		if m.X < minX {
			minX = m.X
		}
		if m.Y < minY {
			minY = m.Y
		}
		if m.X+m.Width > maxX {
			maxX = m.X + m.Width
		}
		if m.Y+m.Height > maxY {
			maxY = m.Y + m.Height
		}
	}
	return Geometry{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}, nil
}

// MonitorByName prefers an exact name, then a unique case-insensitive match.
func MonitorByName(mons []Monitor, name string) (Monitor, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Monitor{}, fmt.Errorf("empty monitor name")
	}
	var folded []Monitor
	for _, m := range mons {
		if m.Name == name {
			return m, nil
		}
		if strings.EqualFold(m.Name, name) {
			folded = append(folded, m)
		}
	}
	if len(folded) == 1 {
		return folded[0], nil
	}
	names := make([]string, 0, len(mons))
	for _, m := range mons {
		names = append(names, m.Name)
	}
	return Monitor{}, fmt.Errorf("no monitor named %q (have %s)", name, strings.Join(names, ", "))
}
