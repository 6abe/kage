package plugin_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func pluginDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func TestManifest(t *testing.T) {
	dir := pluginDir(t)
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		SchemaVersion int               `json:"schemaVersion"`
		ID            string            `json:"id"`
		Name          string            `json:"name"`
		Version       string            `json:"version"`
		KeepLoaded    bool              `json:"keepLoaded"`
		Kinds         []string          `json:"kinds"`
		EntryPoints   map[string]string `json:"entryPoints"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", m.SchemaVersion)
	}
	if m.ID != "kage.ask" {
		t.Fatalf("id = %q, want kage.ask", m.ID)
	}
	if !m.KeepLoaded {
		t.Fatal("keepLoaded = false, want true")
	}
	wantKinds := map[string]bool{"service": false, "overlay": false}
	for _, k := range m.Kinds {
		if _, ok := wantKinds[k]; ok {
			wantKinds[k] = true
		}
	}
	for k, ok := range wantKinds {
		if !ok {
			t.Fatalf("missing kind %q", k)
		}
	}
	if m.EntryPoints["service"] != "Service.qml" {
		t.Fatalf("entryPoints.service = %q, want Service.qml", m.EntryPoints["service"])
	}
	if m.EntryPoints["overlay"] != "Overlay.qml" {
		t.Fatalf("entryPoints.overlay = %q, want Overlay.qml", m.EntryPoints["overlay"])
	}
}

func TestPluginFiles(t *testing.T) {
	dir := pluginDir(t)
	for _, name := range []string{"Overlay.qml", "Service.qml", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestCaptureUsesKageSee(t *testing.T) {
	dir := pluginDir(t)
	service, err := os.ReadFile(filepath.Join(dir, "Service.qml"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile(filepath.Join(dir, "Overlay.qml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(service, []byte(`"kage"`)) || !bytes.Contains(service, []byte(`"see"`)) {
		t.Fatal("Service.qml must invoke kage see")
	}
	for name, body := range map[string][]byte{"Service.qml": service, "Overlay.qml": overlay} {
		for _, banned := range [][]byte{[]byte("grim"), []byte("slurp"), []byte("omarchy screenshot")} {
			if bytes.Contains(body, banned) {
				t.Errorf("%s contains %q", name, banned)
			}
		}
	}
}

func TestOverlayContract(t *testing.T) {
	dir := pluginDir(t)
	overlay, err := os.ReadFile(filepath.Join(dir, "Overlay.qml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"property bool opened",
		"function open(",
		"function close()",
		"shell.hide",
	} {
		if !bytes.Contains(overlay, []byte(want)) {
			t.Errorf("Overlay.qml missing %q", want)
		}
	}
}

func TestInstallNotes(t *testing.T) {
	dir := pluginDir(t)
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"~/.config/omarchy/plugins/kage.ask",
		"rescanPlugins",
		"omarchy plugin enable kage.ask",
		"SUPER + SHIFT + A",
		"omarchy-shell shell summon kage.ask",
		"PRINT",
	} {
		if !bytes.Contains(readme, []byte(want)) {
			t.Errorf("README.md missing %q", want)
		}
	}
	if bytes.Contains(readme, []byte("shell toggle kage.ask")) {
		t.Error("README.md must not suggest toggle")
	}
}
