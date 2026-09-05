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

func TestA2Contracts(t *testing.T) {
	dir := pluginDir(t)
	service, err := os.ReadFile(filepath.Join(dir, "Service.qml"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile(filepath.Join(dir, "Overlay.qml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"annotated.png",
		"pw-record",
		`"16000"`,
		"--channels",
		"voxtype",
		"transcribe",
		"function startRecording",
		"function cancelRecording",
		"function stopMic",
		"function stopAndTranscribe",
		"function beginTranscribe",
		"transcribeWhenRecEnds",
		"unlinkWav",
		`"rm"`,
		"-f",
		"/tmp/kage-",
		"composerText",
	} {
		if !bytes.Contains(service, []byte(want)) {
			t.Errorf("Service.qml missing %q", want)
		}
	}
	if bytes.Contains(service, []byte("record start")) || bytes.Contains(service, []byte(`"record"`)) {
		t.Error("Service.qml must not use voxtype record (type-mode injection)")
	}
	if bytes.Contains(service, []byte(`return "/tmp"`)) || bytes.Contains(service, []byte(`"/tmp/kage/ask"`)) {
		t.Error("Service.qml must not fall back to shared /tmp/kage")
	}
	for _, want := range []string{
		"TextArea",
		"objectName: \"composer\"",
		"recLight",
		"cancelRecording",
		"stopAndTranscribe",
		"startRecording",
		"stopMic",
		"rawPath",
		"Canvas.Image",
		"burnCanvas",
		"annotatedPath",
		"markAnnotated",
		"inkLocked",
		"grabReady",
		"burnSource",
		"sourceSize",
		"onTranscriptReady",
		"pendingBurn",
	} {
		if !bytes.Contains(overlay, []byte(want)) {
			t.Errorf("Overlay.qml missing %q", want)
		}
	}
	if bytes.Contains(overlay, []byte("text: root.service.composerText")) || bytes.Contains(overlay, []byte("text: root.service ? root.service.composerText")) {
		t.Error("composer must not two-way bind text to composerText")
	}
	if bytes.Contains(overlay, []byte("Canvas.FramebufferObject")) {
		t.Error("Overlay.qml must not use FBO canvas for ink")
	}
	if bytes.Contains(overlay, []byte("&& root.statusText.length === 0")) {
		t.Error("grab Image must stay visible while statusText is set")
	}
	for name, body := range map[string][]byte{"Service.qml": service, "Overlay.qml": overlay} {
		for _, banned := range [][]byte{
			[]byte("grok agent"),
			[]byte("session/prompt"),
			[]byte("--prompt-json"),
			[]byte("grim"),
			[]byte("slurp"),
			[]byte("omarchy screenshot"),
		} {
			if bytes.Contains(body, banned) {
				t.Errorf("%s contains %q", name, banned)
			}
		}
	}
	if !bytes.Contains(overlay, []byte("!root.service.error")) {
		t.Error("Overlay.qml must skip startRecording when grab failed")
	}

	cancelIdx := bytes.Index(service, []byte("function cancelRecording()"))
	beginIdx := bytes.Index(service, []byte("function beginTranscribe()"))
	if cancelIdx < 0 || beginIdx < 0 {
		t.Fatal("missing cancelRecording or beginTranscribe")
	}
	cancelEnd := bytes.Index(service[cancelIdx:], []byte("function stopMic()"))
	if cancelEnd < 0 {
		t.Fatal("cancelRecording not followed by stopMic")
	}
	cancelBody := service[cancelIdx : cancelIdx+cancelEnd]
	if !bytes.Contains(cancelBody, []byte("transcribeWhenRecEnds = false")) {
		t.Error("cancelRecording must clear transcribeWhenRecEnds")
	}
	if bytes.Contains(cancelBody, []byte("beginTranscribe")) || bytes.Contains(cancelBody, []byte("transcribeProc.running = true")) {
		t.Error("cancelRecording must not start transcribe")
	}

	stopIdx := bytes.Index(service, []byte("function stopAndTranscribe()"))
	stopEnd := bytes.Index(service[stopIdx:], []byte("function beginTranscribe()"))
	if stopIdx < 0 || stopEnd < 0 {
		t.Fatal("stopAndTranscribe body bounds")
	}
	stopBody := service[stopIdx : stopIdx+stopEnd]
	if bytes.Contains(stopBody, []byte("transcribeProc.running = true")) {
		t.Error("stopAndTranscribe must wait for recProc onExited before transcribe")
	}
	if !bytes.Contains(stopBody, []byte("transcribeWhenRecEnds = true")) {
		t.Error("stopAndTranscribe must set transcribeWhenRecEnds")
	}

	recExited := bytes.Index(service, []byte("id: recProc"))
	if recExited < 0 {
		t.Fatal("missing recProc")
	}
	recTail := service[recExited:]
	if !bytes.Contains(recTail, []byte("if (root.transcribeWhenRecEnds)")) {
		t.Error("recProc onExited must gate transcribe on transcribeWhenRecEnds")
	}
	if !bytes.Contains(recTail, []byte("unlinkWav()")) {
		t.Error("recProc onExited must unlink wav when not transcribing")
	}

	transExited := bytes.Index(service, []byte("id: transcribeProc"))
	if transExited < 0 {
		t.Fatal("missing transcribeProc")
	}
	transBody := service[transExited:]
	failBranch := bytes.Index(transBody, []byte("exitCode !== 0"))
	if failBranch < 0 {
		t.Fatal("transcribeProc missing error path")
	}
	if !bytes.Contains(transBody, []byte("unlinkWav()")) {
		t.Error("transcribeProc must unlink wav including on failure")
	}
	if !bytes.Contains(transBody, []byte("recProc.running")) || !bytes.Contains(transBody, []byte("wavGen")) {
		t.Error("transcribeProc must skip unlinkWav when a new rec owns rec.wav")
	}
	startIdx := bytes.Index(service, []byte("function startRecording()"))
	startEnd := bytes.Index(service[startIdx:], []byte("function unlinkWav()"))
	if startIdx < 0 || startEnd < 0 {
		t.Fatal("startRecording body bounds")
	}
	startBody := service[startIdx : startIdx+startEnd]
	if bytes.Contains(startBody, []byte("transcribeProc.running = false")) {
		t.Error("startRecording must not abort transcribe onto the same wav")
	}
	if !bytes.Contains(startBody, []byte("recAfterTranscribe")) {
		t.Error("startRecording must wait for transcribe before a new rec")
	}
	if bytes.Contains(overlay, []byte("drawImage(rawPreview")) {
		t.Error("burn must not stretch the fitted preview; use burnSource at snapshot size")
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
