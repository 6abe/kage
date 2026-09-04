package hypr

import "testing"

func fixtures() []Window {
	return []Window{
		{Address: "0xabc", Class: "google-chrome", Title: "GitHub", Geometry: Geometry{X: 1, Y: 2, Width: 3, Height: 4}},
		{Address: "0xdef", Class: "kitty", Title: "term", Geometry: Geometry{X: 5, Y: 6, Width: 7, Height: 8}},
		{Address: "0x111", Class: "kitty", Title: "logs", Geometry: Geometry{X: 9, Y: 10, Width: 11, Height: 12}},
		{Address: "0x222", Class: "Firefox", Title: "GitHub pull request", Geometry: Geometry{}},
	}
}

func TestMatchAddressClassTitle(t *testing.T) {
	ws := fixtures()
	got := Match(ws, "0xABC")
	if len(got) != 1 || got[0].Address != "0xabc" {
		t.Fatalf("address: %+v", got)
	}
	got = Match(ws, "0xabc")
	if len(got) != 1 || got[0].Address != "0xabc" {
		t.Fatalf("address exact: %+v", got)
	}
	got = Match(ws, "0xab")
	if len(got) != 0 {
		t.Fatalf("address is not a prefix: %+v", got)
	}
	got = Match(ws, "Google-Chrome")
	if len(got) != 1 || got[0].Address != "0xabc" {
		t.Fatalf("class fold: %+v", got)
	}
	got = Match(ws, "chrome")
	if len(got) != 0 {
		t.Fatalf("class is not a substring: %+v", got)
	}
	got = Match(ws, "pull")
	if len(got) != 1 || got[0].Address != "0x222" {
		t.Fatalf("title substring: %+v", got)
	}
	got = Match(ws, "GITHUB")
	if len(got) != 2 {
		t.Fatalf("title fold hits both GitHub titles: %+v", got)
	}
	got = Match(ws, "kitty")
	if len(got) != 2 {
		t.Fatalf("class preferred even when titles differ: %+v", got)
	}
	got = Match(ws, "term")
	if len(got) != 1 || got[0].Address != "0xdef" {
		t.Fatalf("title after no class: %+v", got)
	}
	if hits := Match(ws, "nope"); len(hits) != 0 {
		t.Fatalf("no match: %+v", hits)
	}
	if hits := Match(ws, ""); len(hits) != 0 {
		t.Fatalf("empty: %+v", hits)
	}
	if hits := Match(ws, "  0xdef  "); len(hits) != 1 {
		t.Fatalf("trim: %+v", hits)
	}
}

func TestMatchOneErrors(t *testing.T) {
	ws := fixtures()
	w, err := MatchOne(ws, "0xabc")
	if err != nil || w.Address != "0xabc" {
		t.Fatalf("one: %+v %v", w, err)
	}
	_, err = MatchOne(ws, "missing")
	if err == nil || err.Error() != `no window matches "missing"` {
		t.Fatalf("none: %v", err)
	}
	_, err = MatchOne(ws, "kitty")
	amb, ok := err.(*AmbiguousError)
	if !ok || len(amb.Matches) != 2 {
		t.Fatalf("ambiguous: %v", err)
	}
	if amb.Query != "kitty" {
		t.Fatalf("query %q", amb.Query)
	}
	_, err = MatchOne(ws, "  ")
	if err == nil || err.Error() != "empty window query" {
		t.Fatalf("empty: %v", err)
	}
}
