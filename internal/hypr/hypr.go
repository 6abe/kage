package hypr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/6abe/kage/internal/host"
)

type Geometry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Window struct {
	Address   string   `json:"address"`
	Class     string   `json:"class"`
	Title     string   `json:"title"`
	Geometry  Geometry `json:"geometry"`
	Workspace int      `json:"workspace"`
	Monitor   string   `json:"monitor"`
	Mapped    bool     `json:"mapped"`
	Floating  bool     `json:"floating"`
	Focus     bool     `json:"focus"`
}

type Monitor struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Scale   float64 `json:"scale"`
	Focused bool    `json:"focused"`
}

type hyprWorkspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type hyprClient struct {
	Address        string          `json:"address"`
	Mapped         bool            `json:"mapped"`
	At             [2]int          `json:"at"`
	Size           [2]int          `json:"size"`
	Workspace      hyprWorkspace   `json:"workspace"`
	Monitor        json.RawMessage `json:"monitor"`
	Class          string          `json:"class"`
	Title          string          `json:"title"`
	Floating       bool            `json:"floating"`
	FocusHistoryID int             `json:"focusHistoryID"`
}

type hyprActive struct {
	Address string `json:"address"`
}

func ListMonitors(h host.Host) ([]Monitor, error) {
	raw, err := h.HyprctlJSON("monitors")
	if err != nil {
		return nil, err
	}
	var out []Monitor
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse hyprctl monitors: %w", err)
	}
	if out == nil {
		out = []Monitor{}
	}
	return out, nil
}

func ListWindows(h host.Host) ([]Window, error) {
	mons, err := ListMonitors(h)
	if err != nil {
		return nil, err
	}
	raw, err := h.HyprctlJSON("clients")
	if err != nil {
		return nil, err
	}
	var clients []hyprClient
	if err := json.Unmarshal(raw, &clients); err != nil {
		return nil, fmt.Errorf("parse hyprctl clients: %w", err)
	}
	focusAddr := ""
	if aw, err := h.HyprctlJSON("activewindow"); err == nil {
		var a hyprActive
		if json.Unmarshal(aw, &a) == nil {
			focusAddr = a.Address
		}
	}
	out := make([]Window, 0, len(clients))
	for _, c := range clients {
		w := Window{
			Address: c.Address,
			Class:   c.Class,
			Title:   c.Title,
			Geometry: Geometry{
				X:      c.At[0],
				Y:      c.At[1],
				Width:  c.Size[0],
				Height: c.Size[1],
			},
			Workspace: c.Workspace.ID,
			Monitor:   monitorName(mons, c.Monitor),
			Mapped:    c.Mapped,
			Floating:  c.Floating,
			Focus:     focusAddr != "" && c.Address == focusAddr,
		}
		if !w.Focus && focusAddr == "" && c.FocusHistoryID == 0 {
			w.Focus = true
		}
		out = append(out, w)
	}
	return out, nil
}

func monitorName(mons []Monitor, raw json.RawMessage) string {
	s := string(bytes.TrimSpace(raw))
	s = trimQuotes(s)
	if s == "" || s == "null" {
		return ""
	}
	id, err := strconv.Atoi(s)
	if err != nil {
		return s
	}
	for _, m := range mons {
		if m.ID == id {
			return m.Name
		}
	}
	if id >= 0 && id < len(mons) {
		return mons[id].Name
	}
	return strconv.Itoa(id)
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
