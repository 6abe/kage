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
				"floating":false,"focusHistoryID":0
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
}
