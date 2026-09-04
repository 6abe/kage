package hypr

import (
	"fmt"
	"strings"
)

// AmbiguousError means a query matched more than one client; do not pick a winner.
type AmbiguousError struct {
	Query   string
	Matches []Window
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("ambiguous window match %q (%d clients)", e.Query, len(e.Matches))
}

// Match returns clients for query: exact address (0x...), else case-insensitive
// class, else case-insensitive title substring. Never ranks a silent winner.
func Match(windows []Window, query string) []Window {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(query), "0x") {
		var out []Window
		for _, w := range windows {
			if strings.EqualFold(w.Address, query) {
				out = append(out, w)
			}
		}
		return out
	}
	var classHits []Window
	for _, w := range windows {
		if strings.EqualFold(w.Class, query) {
			classHits = append(classHits, w)
		}
	}
	if len(classHits) > 0 {
		return classHits
	}
	q := strings.ToLower(query)
	var titleHits []Window
	for _, w := range windows {
		if strings.Contains(strings.ToLower(w.Title), q) {
			titleHits = append(titleHits, w)
		}
	}
	return titleHits
}

func MatchOne(windows []Window, query string) (Window, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Window{}, fmt.Errorf("empty window query")
	}
	hits := Match(windows, query)
	switch len(hits) {
	case 0:
		return Window{}, fmt.Errorf("no window matches %q", query)
	case 1:
		return hits[0], nil
	default:
		return Window{}, &AmbiguousError{Query: query, Matches: hits}
	}
}
