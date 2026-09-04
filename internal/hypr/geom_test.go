package hypr

import "testing"

func TestGrimGeomSlurpStyle(t *testing.T) {
	got := GrimGeom(100, 80, 1400, 900)
	if got != "100,80 1400x900" {
		t.Fatalf("%q", got)
	}
	if got == "100,80,1400,900" {
		t.Fatal("x,y,w,h is invalid grim geometry")
	}
	neg := Geometry{X: -1920, Y: 0, Width: 1920, Height: 1080}.Grim()
	if neg != "-1920,0 1920x1080" {
		t.Fatalf("neg %q", neg)
	}
}

func TestBoundingBox(t *testing.T) {
	box, err := BoundingBox([]Monitor{
		{Name: "DP-1", X: 0, Y: 0, Width: 100, Height: 50},
		{Name: "HDMI-A-1", X: 100, Y: -10, Width: 80, Height: 60},
	})
	if err != nil {
		t.Fatal(err)
	}
	if box != (Geometry{X: 0, Y: -10, Width: 180, Height: 60}) {
		t.Fatalf("%+v", box)
	}
	if _, err := BoundingBox(nil); err == nil {
		t.Fatal("empty")
	}
}

func TestMonitorByName(t *testing.T) {
	mons := []Monitor{{Name: "DP-1"}, {Name: "HDMI-A-1"}}
	m, err := MonitorByName(mons, "HDMI-A-1")
	if err != nil || m.Name != "HDMI-A-1" {
		t.Fatalf("%+v %v", m, err)
	}
	m, err = MonitorByName(mons, "dp-1")
	if err != nil || m.Name != "DP-1" {
		t.Fatalf("fold %+v %v", m, err)
	}
	_, err = MonitorByName(mons, "NOPE")
	if err == nil || err.Error() != `no monitor named "NOPE" (have DP-1, HDMI-A-1)` {
		t.Fatalf("missing: %v", err)
	}
}
