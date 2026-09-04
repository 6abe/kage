package hypr

import (
	"testing"

	"github.com/6abe/kage/internal/host"
)

func TestListWindowsMonitorNameAndFocus(t *testing.T) {
	h := &host.Fake{
		JSON: map[string][]byte{
			"monitors": []byte(`[{"id":1,"name":"DP-1","x":0,"y":0,"width":100,"height":100,"scale":1,"focused":true}]`),
			"clients": []byte(`[{
				"address":"0xabc","mapped":true,"at":[1,2],"size":[3,4],
				"workspace":{"id":9,"name":"9"},"monitor":1,"class":"a","title":"b",
				"pid":4321,"floating":false,"focusHistoryID":0
			}]`),
			"activewindow": []byte(`{"address":"0xabc"}`),
		},
	}
	ws, err := ListWindows(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 {
		t.Fatalf("len %d", len(ws))
	}
	if ws[0].Monitor != "DP-1" || ws[0].Workspace != 9 || !ws[0].Focus {
		t.Fatalf("%+v", ws[0])
	}
	if ws[0].Geometry.X != 1 || ws[0].Geometry.Width != 3 {
		t.Fatalf("geom %+v", ws[0].Geometry)
	}
	if ws[0].Pid != 4321 {
		t.Fatalf("pid %d", ws[0].Pid)
	}
}

func TestFocusedMonitor(t *testing.T) {
	mons := []Monitor{
		{Name: "HDMI-A-1", Width: 1, Focused: false},
		{Name: "DP-1", Width: 2, Focused: true},
	}
	m, err := FocusedMonitor(mons)
	if err != nil || m.Name != "DP-1" {
		t.Fatalf("got %+v err=%v", m, err)
	}
	if _, err := FocusedMonitor(nil); err == nil {
		t.Fatal("expected no focused monitor")
	}
}

func sampleWins() []Window {
	return []Window{
		{Address: "0x123", Class: "google-chrome", Title: "GitHub"},
		{Address: "0x456", Class: "kitty", Title: "term"},
		{Address: "0x789", Class: "kitty", Title: "logs"},
		{Address: "0xabc", Class: "firefox", Title: "GitHub — Mozilla Firefox"},
	}
}

func TestMatchWindowAddressClassTitle(t *testing.T) {
	wins := sampleWins()
	w, _, err := MatchWindow(wins, "0x123")
	if err != nil || w.Address != "0x123" {
		t.Fatalf("address: %v %+v", err, w)
	}
	w, _, err = MatchWindow(wins, "0X123")
	if err != nil || w.Address != "0x123" {
		t.Fatalf("address fold: %v %+v", err, w)
	}
	w, _, err = MatchWindow(wins, "Google-Chrome")
	if err != nil || w.Address != "0x123" {
		t.Fatalf("class: %v %+v", err, w)
	}
	w, _, err = MatchWindow(wins, "Mozilla")
	if err != nil || w.Address != "0xabc" {
		t.Fatalf("title: %v %+v", err, w)
	}
	_, _, err = MatchWindow(wins, "mozilla")
	if err != ErrNotFound {
		t.Fatalf("title is case-sensitive: %v", err)
	}
	_, matches, err := MatchWindow(wins, "kitty")
	if err != ErrAmbiguous || len(matches) != 2 {
		t.Fatalf("ambiguous class: %v %d", err, len(matches))
	}
	_, matches, err = MatchWindow(wins, "GitHub")
	if err != ErrAmbiguous || len(matches) != 2 {
		t.Fatalf("ambiguous title: %v %d", err, len(matches))
	}
	_, _, err = MatchWindow(wins, "nope")
	if err != ErrNotFound {
		t.Fatalf("missing: %v", err)
	}
}

func TestMatchWindowClassBeforeTitle(t *testing.T) {
	wins := []Window{
		{Address: "0x1", Class: "kitty", Title: "foo"},
		{Address: "0x2", Class: "firefox", Title: "kitty docs"},
	}
	w, _, err := MatchWindow(wins, "kitty")
	if err != nil || w.Address != "0x1" {
		t.Fatalf("%v %+v", err, w)
	}
}
