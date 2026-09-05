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
		"function burnMarks",
		"magick",
		"-draw",
		"polyline",
		"annotatedPath",
		"id: burnProc",
		"annotatedTmpPath",
		"pendingStrokes",
		"function strokeWidth",
		"unlinkAnnotated",
		`"mv"`,
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
		"annotatedPath",
		"burnMarks",
		"inkLocked",
		"grabReady",
		"onTranscriptReady",
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
		t.Error("burn must not stretch the fitted preview; composite onto raw.png at snapshot size")
	}
	if bytes.Contains(overlay, []byte("Canvas.save")) || bytes.Contains(overlay, []byte(".save(")) {
		t.Error("Overlay.qml must not use Canvas.save for annotated.png")
	}
	if bytes.Contains(overlay, []byte("id: burnCanvas")) || bytes.Contains(overlay, []byte("id: burnSource")) {
		t.Error("hidden burn canvas/source must not be used")
	}
	if !bytes.Contains(service, []byte("root.rawPath")) || !bytes.Contains(service, []byte("root.annotatedPath")) {
		t.Error("magick must write annotated.png from raw.png")
	}
	if !bytes.Contains(overlay, []byte("root.service.burnMarks(all)")) {
		t.Error("Overlay.qml must pass polylines into service.burnMarks(")
	}
	if !bytes.Contains(overlay, []byte("Math.max(3, Math.round(w / 400))")) {
		t.Error("Overlay ink stroke width must match magick rounding")
	}
	if !bytes.Contains(service, []byte("Math.max(3, Math.round(w / 400))")) {
		t.Error("magick stroke width must match ink rounding")
	}

	burnIdx := bytes.Index(service, []byte("function burnMarks("))
	if burnIdx < 0 {
		t.Fatal("missing burnMarks")
	}
	startBurnIdx := bytes.Index(service, []byte("function startBurn("))
	if startBurnIdx < 0 {
		t.Fatal("missing startBurn")
	}
	burnMarksBody := service[burnIdx:startBurnIdx]
	if !bytes.Contains(burnMarksBody, []byte("pendingStrokes")) {
		t.Error("burnMarks must queue pendingStrokes when magick/mv/rm is running")
	}
	if !bytes.Contains(burnMarksBody, []byte("burnProc.running")) || !bytes.Contains(burnMarksBody, []byte("mvBurnProc.running")) {
		t.Error("burnMarks must treat magick and mv as busy")
	}
	if !bytes.Contains(burnMarksBody, []byte("rmAnnotatedProc.running")) {
		t.Error("burnMarks must treat unlink as a busy burn")
	}
	if bytes.Contains(burnMarksBody, []byte("burnProc.running = false")) {
		t.Error("burnMarks must not kill in-flight magick")
	}
	startBurnEnd := bytes.Index(service[startBurnIdx:], []byte("function flushPendingBurn()"))
	if startBurnEnd < 0 {
		t.Fatal("startBurn not followed by flushPendingBurn")
	}
	startBurnBody := service[startBurnIdx : startBurnIdx+startBurnEnd]
	if !bytes.Contains(startBurnBody, []byte("if (!drew)")) {
		t.Error("startBurn must return when !drew")
	}
	if !bytes.Contains(startBurnBody, []byte("pts.length < 2")) {
		t.Error("startBurn must skip strokes with pts.length < 2")
	}
	drewIdx := bytes.Index(startBurnBody, []byte("if (!drew)"))
	if drewIdx < 0 {
		t.Fatal("missing !drew guard")
	}
	runIdx := bytes.Index(startBurnBody, []byte("burnProc.running = true"))
	if runIdx < 0 {
		t.Fatal("startBurn never starts burnProc")
	}
	if drewIdx > runIdx {
		t.Error("!drew path must not start burnProc")
	}
	drewReturn := startBurnBody[drewIdx:]
	if !bytes.Contains(drewReturn[:min(len(drewReturn), 80)], []byte("return")) {
		t.Error("!drew must return before starting magick")
	}
	if !bytes.Contains(startBurnBody, []byte(`["magick"`)) {
		t.Error("burn command must start with magick argv, not a shell")
	}
	if bytes.Contains(startBurnBody, []byte(`"sh"`)) || bytes.Contains(startBurnBody, []byte(`"-c"`)) {
		t.Error("burn must not wrap magick in sh -c")
	}

	burnProcIdx := bytes.Index(service, []byte("id: burnProc"))
	if burnProcIdx < 0 {
		t.Fatal("missing burnProc")
	}
	burnTail := service[burnProcIdx:]
	failIdx := bytes.Index(burnTail, []byte("exitCode !== 0"))
	if failIdx < 0 {
		t.Fatal("burnProc missing error path")
	}
	failReturn := bytes.Index(burnTail[failIdx:], []byte("return"))
	if failReturn < 0 {
		t.Fatal("burnProc fail path missing return")
	}
	failBody := burnTail[failIdx : failIdx+failReturn]
	if !bytes.Contains(failBody, []byte("root.error")) {
		t.Error("burnProc fail path must set root.error")
	}
	if bytes.Contains(failBody, []byte("markAnnotated")) || bytes.Contains(failBody, []byte("hasMarks = true")) {
		t.Error("burnProc fail path must not markAnnotated")
	}
	if !bytes.Contains(failBody, []byte("annotatedTmpPath")) && !bytes.Contains(failBody, []byte("rmTmpProc")) {
		t.Error("magick fail must unlink annotated.png.tmp")
	}
	if bytes.Contains(failBody, []byte("flushPendingBurn")) || bytes.Contains(failBody, []byte("startBurn")) {
		t.Error("magick fail must not startBurn while tmp unlink is running")
	}
	if !bytes.Contains(burnTail, []byte("markAnnotated")) {
		t.Error("successful burn must still markAnnotated")
	}
	mvIdx := bytes.Index(service, []byte("id: mvBurnProc"))
	if mvIdx < 0 {
		t.Fatal("missing mvBurnProc")
	}
	mvTail := service[mvIdx:]
	mvFail := bytes.Index(mvTail, []byte("exitCode !== 0"))
	if mvFail < 0 {
		t.Fatal("mvBurnProc missing error path")
	}
	mvFailRet := bytes.Index(mvTail[mvFail:], []byte("return"))
	if mvFailRet < 0 {
		t.Fatal("mvBurnProc fail path missing return")
	}
	mvFailBody := mvTail[mvFail : mvFail+mvFailRet]
	if !bytes.Contains(mvFailBody, []byte("root.error")) {
		t.Error("mv fail must set root.error")
	}
	if bytes.Contains(mvFailBody, []byte("markAnnotated")) || bytes.Contains(mvFailBody, []byte("hasMarks = true")) {
		t.Error("mv fail must not markAnnotated")
	}

	seeIdx := bytes.Index(service, []byte("id: seeProc"))
	if seeIdx < 0 {
		t.Fatal("missing seeProc")
	}
	seeEnd := bytes.Index(service[seeIdx:], []byte("id: recProc"))
	if seeEnd < 0 {
		t.Fatal("seeProc not followed by recProc")
	}
	seeBody := service[seeIdx : seeIdx+seeEnd]
	if !bytes.Contains(seeBody, []byte("unlinkAnnotated()")) {
		t.Error("successful grab must unlink leftover annotated.png")
	}
	u := bytes.Index(seeBody, []byte("unlinkAnnotated()"))
	if bytes.Contains(seeBody[u:], []byte("grabFinished")) {
		t.Error("seeProc must not grabFinished until annotated unlink exits")
	}
	rmAnn := bytes.Index(service, []byte("id: rmAnnotatedProc"))
	if rmAnn < 0 {
		t.Fatal("missing rmAnnotatedProc")
	}
	rmTail := service[rmAnn:]
	rmEnd := bytes.Index(rmTail[1:], []byte("id: "))
	if rmEnd < 0 {
		rmEnd = len(rmTail) - 1
	}
	rmBody := rmTail[:rmEnd+1]
	if !bytes.Contains(rmBody, []byte("grabFinished")) {
		t.Error("rmAnnotatedProc onExited must grabFinished after unlink")
	}
	unlinkFn := bytes.Index(service, []byte("function unlinkAnnotated()"))
	if unlinkFn < 0 {
		t.Fatal("missing unlinkAnnotated")
	}
	unlinkEnd := bytes.Index(service[unlinkFn:], []byte("function strokeWidth"))
	if unlinkEnd < 0 {
		t.Fatal("unlinkAnnotated bounds")
	}
	unlinkBody := service[unlinkFn : unlinkFn+unlinkEnd]
	if bytes.Contains(unlinkBody, []byte("rmAnnotatedProc.running = false")) {
		t.Error("unlinkAnnotated must not kill in-flight rm")
	}
	if !bytes.Contains(unlinkBody, []byte("unlinkJobGen")) {
		t.Error("unlink job id must be assigned when unlink starts")
	}
	rmTmpIdx := bytes.Index(service, []byte("id: rmTmpProc"))
	if rmTmpIdx < 0 {
		t.Fatal("missing rmTmpProc")
	}
	rmTmpTail := service[rmTmpIdx:]
	rmTmpEnd := bytes.Index(rmTmpTail[1:], []byte("id: "))
	if rmTmpEnd < 0 {
		rmTmpEnd = len(rmTmpTail) - 1
	}
	if !bytes.Contains(rmTmpTail[:rmTmpEnd+1], []byte("flushPendingBurn")) {
		t.Error("rmTmpProc onExited must flushPendingBurn")
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
