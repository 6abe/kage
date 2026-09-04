package hypr

import (
	"errors"
	"strings"
)

var (
	ErrNotFound  = errors.New("no matching window")
	ErrAmbiguous = errors.New("ambiguous window match")
	ErrNoFocus   = errors.New("no focused window")
)

// MatchWindow resolves ADDRESS (0x...), then class (case-insensitive), then title substring.
// Two or more hits return ErrAmbiguous with the list; never pick silently.
func MatchWindow(wins []Window, query string) (Window, []Window, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return Window{}, nil, ErrNotFound
	}
	if strings.HasPrefix(strings.ToLower(q), "0x") {
		return unique(filter(wins, func(w Window) bool {
			return strings.EqualFold(w.Address, q)
		}))
	}
	classHits := filter(wins, func(w Window) bool {
		return strings.EqualFold(w.Class, q)
	})
	if len(classHits) > 0 {
		return unique(classHits)
	}
	return unique(filter(wins, func(w Window) bool {
		return strings.Contains(w.Title, q)
	}))
}

// FocusedWindow returns the focused client, or ErrNoFocus.
func FocusedWindow(wins []Window) (Window, error) {
	for _, w := range wins {
		if w.Focus {
			return w, nil
		}
	}
	return Window{}, ErrNoFocus
}

func filter(wins []Window, ok func(Window) bool) []Window {
	var out []Window
	for _, w := range wins {
		if ok(w) {
			out = append(out, w)
		}
	}
	return out
}

func unique(hits []Window) (Window, []Window, error) {
	switch len(hits) {
	case 0:
		return Window{}, nil, ErrNotFound
	case 1:
		return hits[0], nil, nil
	default:
		return Window{}, hits, ErrAmbiguous
	}
}
