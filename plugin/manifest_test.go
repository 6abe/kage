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
	if !bytes.Contains(rmBody, []byte("captureDone")) && !bytes.Contains(rmBody, []byte("grabFinished")) {
		t.Error("rmAnnotatedProc onExited must finish the grab after unlink")
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

func TestA3Contracts(t *testing.T) {
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
		"function send()",
		"function startGrok(",
		"function handleStreamLine(",
		"function payloadImagePath()",
		"function persistSessionId(",
		"function grabReady()",
		"function burnBusy()",
		"function isUuid(",
		"function launchSend()",
		"composerMaxChars",
		"ask-session",
		"--prompt-json",
		"--output-format",
		"streaming-json",
		"--session-id",
		"--resume",
		"--cwd",
		"homeDir",
		"--permission-mode",
		"dontAsk",
		"--deny",
		"mcp__kage__kage_click",
		"kage click",
		"kage type",
		"kage press",
		"kage hotkey",
		"--disallowed-tools",
		"kage__kage_click",
		"prompt.txt",
		"promptTxtPath",
		"uuidgen",
		"id: grokProc",
		"id: uuidProc",
		"SplitParser",
		"chatMessages",
		"sending",
		"streamBuf",
		"setLastAssistant",
		"appendChat",
	} {
		if !bytes.Contains(service, []byte(want)) {
			t.Errorf("Service.qml missing %q", want)
		}
	}
	for _, banned := range []string{
		"--always-approve",
		"--yolo",
		"always-approve",
		"base64",
		"kage agent",
	} {
		if bytes.Contains(service, []byte(banned)) {
			t.Errorf("Service.qml contains %q", banned)
		}
	}
	if bytes.Contains(overlay, []byte("--always-approve")) || bytes.Contains(overlay, []byte("--yolo")) {
		t.Error("Overlay.qml must not yolo desktop input")
	}

	sendIdx := bytes.Index(service, []byte("function send()"))
	if sendIdx < 0 {
		t.Fatal("missing send")
	}
	sendEnd := bytes.Index(service[sendIdx:], []byte("function launchSend()"))
	if sendEnd < 0 {
		t.Fatal("send not followed by launchSend")
	}
	sendBody := service[sendIdx : sendIdx+sendEnd]
	if !bytes.Contains(sendBody, []byte("stopMic()")) {
		t.Error("send must stopMic")
	}
	if !bytes.Contains(sendBody, []byte("grabReady()")) {
		t.Error("send must require a finished grab")
	}
	if !bytes.Contains(sendBody, []byte("composerMaxChars")) {
		t.Error("send must cap composer length")
	}
	if bytes.Contains(sendBody, []byte("appendChat")) || bytes.Contains(sendBody, []byte("uuidProc")) || bytes.Contains(sendBody, []byte("startGrok(")) {
		t.Error("send error/wait path must not appendChat, uuidgen, or startGrok")
	}
	if !bytes.Contains(sendBody, []byte("burnBusy()")) || !bytes.Contains(sendBody, []byte("sendQueued")) {
		t.Error("send must wait for in-flight burn")
	}
	if bytes.Contains(sendBody, []byte("grokProc.running = false")) {
		t.Error("send must not kill an in-flight grok")
	}
	failNothing := bytes.Index(sendBody, []byte("nothing to send"))
	if failNothing < 0 {
		t.Fatal("send missing nothing-to-send path")
	}
	failRet := bytes.Index(sendBody[failNothing:], []byte("return"))
	if failRet < 0 {
		t.Fatal("nothing-to-send missing return")
	}
	nothingBody := sendBody[failNothing : failNothing+failRet]
	if bytes.Contains(nothingBody, []byte("appendChat")) || bytes.Contains(nothingBody, []byte("uuidProc")) || bytes.Contains(nothingBody, []byte("startGrok")) {
		t.Error("nothing-to-send must not appendChat, uuidgen, or startGrok")
	}

	launchIdx := bytes.Index(service, []byte("function launchSend()"))
	launchEnd := bytes.Index(service[launchIdx:], []byte("function maybeLaunchQueuedSend()"))
	if launchIdx < 0 || launchEnd < 0 {
		t.Fatal("launchSend bounds")
	}
	launchBody := service[launchIdx : launchIdx+launchEnd]
	if !bytes.Contains(launchBody, []byte("promptFile.setText")) {
		t.Error("launchSend must write prompt.txt")
	}
	if !bytes.Contains(launchBody, []byte("appendChat")) {
		t.Error("launchSend must append the user turn")
	}
	if !bytes.Contains(launchBody, []byte("uuidProc")) {
		t.Error("first send must uuidgen when no session")
	}
	if !bytes.Contains(launchBody, []byte("pendingResume")) {
		t.Error("send must resume when sessionId is set")
	}

	uuidIdx := bytes.Index(service, []byte("id: uuidProc"))
	if uuidIdx < 0 {
		t.Fatal("missing uuidProc")
	}
	uuidTail := service[uuidIdx:]
	failIdx := bytes.Index(uuidTail, []byte("exitCode !== 0"))
	if failIdx < 0 {
		t.Fatal("uuidProc missing error path")
	}
	uuidFailRet := bytes.Index(uuidTail[failIdx:], []byte("return"))
	if uuidFailRet < 0 {
		t.Fatal("uuidProc fail path missing return")
	}
	uuidFail := uuidTail[failIdx : failIdx+uuidFailRet]
	if !bytes.Contains(uuidFail, []byte("root.error")) {
		t.Error("uuidgen fail must set root.error")
	}
	if !bytes.Contains(uuidFail, []byte("sending = false")) {
		t.Error("uuidgen fail must clear sending")
	}
	if bytes.Contains(uuidFail, []byte("startGrok")) || bytes.Contains(uuidFail, []byte("persistSessionId")) {
		t.Error("uuidgen fail must not start grok or persist a session")
	}

	cfgIdx := bytes.Index(service, []byte("id: writeSessionProc"))
	if cfgIdx < 0 {
		t.Fatal("missing writeSessionProc")
	}
	cfgTail := service[cfgIdx:]
	cfgFail := bytes.Index(cfgTail, []byte("exitCode !== 0"))
	if cfgFail < 0 {
		t.Fatal("writeSessionProc missing error path")
	}
	cfgFailRet := bytes.Index(cfgTail[cfgFail:], []byte("return"))
	if cfgFailRet < 0 {
		t.Fatal("writeSessionProc fail missing return")
	}
	cfgFailBody := cfgTail[cfgFail : cfgFail+cfgFailRet]
	if !bytes.Contains(cfgFailBody, []byte("root.error")) && !bytes.Contains(cfgFailBody, []byte("abortSend")) {
		t.Error("session write fail must set root.error")
	}
	if bytes.Contains(cfgFailBody, []byte("setText")) {
		t.Error("session write fail must not FileView.setText ask-session")
	}
	if bytes.Contains(cfgFailBody, []byte("startGrok(")) {
		t.Error("session write fail must not start grok")
	}
	persistIdx := bytes.Index(service, []byte("function persistSessionId("))
	persistEnd := bytes.Index(service[persistIdx:], []byte("function abortSend("))
	if persistIdx < 0 || persistEnd < 0 {
		t.Fatal("persistSessionId bounds")
	}
	persistBody := service[persistIdx : persistIdx+persistEnd]
	if !bytes.Contains(persistBody, []byte("chmod 0600")) && !bytes.Contains(persistBody, []byte(`"0600"`)) {
		t.Error("ask-session write must set mode 0600 in the same process")
	}
	if bytes.Contains(persistBody, []byte("sessionFile.setText")) {
		t.Error("must not chmod ask-session before the write process exits")
	}

	grokIdx := bytes.Index(service, []byte("id: grokProc"))
	if grokIdx < 0 {
		t.Fatal("missing grokProc")
	}
	grokTail := service[grokIdx:]
	if !bytes.Contains(grokTail, []byte("SplitParser")) {
		t.Error("grok stdout must stream via SplitParser")
	}
	stdoutSlice := grokTail
	if sp := bytes.Index(stdoutSlice, []byte("stdout:")); sp >= 0 {
		end := bytes.Index(stdoutSlice[sp:], []byte("stderr:"))
		if end < 0 {
			end = 400
		}
		stdoutBody := stdoutSlice[sp : sp+end]
		if bytes.Contains(stdoutBody, []byte("waitForEnd: true")) {
			t.Error("grok stdout must not waitForEnd before streaming")
		}
		if bytes.Contains(stdoutBody, []byte("StdioCollector")) {
			t.Error("grok stdout must stream lines, not collect until exit")
		}
	}
	grokFail := bytes.Index(grokTail, []byte("exitCode !== 0"))
	if grokFail < 0 {
		t.Fatal("grokProc missing error path")
	}
	if !bytes.Contains(grokTail, []byte("sending = false")) {
		t.Error("grok exit must clear sending")
	}
	if !bytes.Contains(grokTail, []byte("sendFinished")) {
		t.Error("grok exit must sendFinished")
	}
	if bytes.Contains(grokTail, []byte("grokErr.text")) && bytes.Contains(grokTail, []byte("root.error = msg")) {
		t.Error("must not dump grok stderr into the overlay")
	}
	if !bytes.Contains(grokTail, []byte("grok failed (")) {
		t.Error("grok fail with empty streamBuf must set a short local error")
	}
	if bytes.Contains(grokTail, []byte("startGrok(")) {
		t.Error("grok fail must not spawn another grok")
	}

	startIdx := bytes.Index(service, []byte("function startGrok("))
	startEnd := bytes.Index(service[startIdx:], []byte("function handleStreamLine("))
	if startIdx < 0 || startEnd < 0 {
		t.Fatal("startGrok bounds")
	}
	startBody := service[startIdx : startIdx+startEnd]
	if !bytes.Contains(startBody, []byte(`"--cwd"`)) || !bytes.Contains(startBody, []byte("root.homeDir")) {
		t.Error("grok cwd must be $HOME")
	}
	if bytes.Contains(startBody, []byte("git")) {
		t.Error("must not guess a repo cwd")
	}
	if bytes.Contains(startBody, []byte(`"-p"`)) || bytes.Contains(startBody, []byte(`"."`)) {
		t.Error("must not pass -p with a dummy \".\" prompt; --prompt-json is headless")
	}
	if bytes.Contains(startBody, []byte("bypassPermissions")) {
		t.Error("Ask mode must not auto-approve tools")
	}

	hsIdx := bytes.Index(service, []byte("function handleStreamLine("))
	hsEnd := bytes.Index(service[hsIdx:], []byte("function burnMarks("))
	if hsIdx < 0 || hsEnd < 0 {
		t.Fatal("handleStreamLine bounds")
	}
	hsBody := service[hsIdx : hsIdx+hsEnd]
	if bytes.Contains(hsBody, []byte("persistSessionId")) || bytes.Contains(hsBody, []byte("sessionId")) {
		t.Error("stream lines must not clobber ask-session")
	}

	payloadIdx := bytes.Index(service, []byte("function payloadImagePath()"))
	payloadEnd := bytes.Index(service[payloadIdx:], []byte("function isUuid("))
	if payloadIdx < 0 || payloadEnd < 0 {
		t.Fatal("payloadImagePath bounds")
	}
	payloadBody := service[payloadIdx : payloadIdx+payloadEnd]
	if !bytes.Contains(payloadBody, []byte("hasMarks")) || !bytes.Contains(payloadBody, []byte("annotatedPath")) {
		t.Error("send must prefer annotated.png when marks exist")
	}
	if !bytes.Contains(payloadBody, []byte("rawPath")) {
		t.Error("send must fall back to raw.png")
	}

	buildIdx := bytes.Index(service, []byte("function buildPromptJson()"))
	if buildIdx < 0 {
		t.Fatal("missing buildPromptJson")
	}
	buildEnd := bytes.Index(service[buildIdx:], []byte("function send()"))
	if buildEnd < 0 {
		t.Fatal("buildPromptJson bounds")
	}
	buildBody := service[buildIdx : buildIdx+buildEnd]
	if !bytes.Contains(buildBody, []byte("snapshotJsonPath")) {
		t.Error("prompt-json must include snapshot.json path")
	}
	if bytes.Contains(buildBody, []byte("readFile")) || bytes.Contains(bytes.ToLower(buildBody), []byte("base64")) {
		t.Error("must not base64 the screenshot")
	}

	closeIdx := bytes.Index(overlay, []byte("function close()"))
	closeEnd := bytes.Index(overlay[closeIdx:], []byte("function dismiss()"))
	if closeIdx < 0 || closeEnd < 0 {
		t.Fatal("close bounds")
	}
	closeBody := overlay[closeIdx : closeIdx+closeEnd]
	if bytes.Contains(closeBody, []byte("grokProc.running = false")) {
		t.Error("Esc/close must not kill the grok process")
	}
	if !bytes.Contains(closeBody, []byte("abortQueuedSend")) {
		t.Error("close must drop a queued send that has not started grok")
	}

	grabIdx := bytes.Index(service, []byte("function grab("))
	grabEnd := bytes.Index(service[grabIdx:], []byte("function startRecording()"))
	if grabIdx < 0 || grabEnd < 0 {
		t.Fatal("grab bounds")
	}
	grabBody := service[grabIdx : grabIdx+grabEnd]
	if !bytes.Contains(grabBody, []byte("abortQueuedSend")) {
		t.Error("grab must drop a queued send so sending cannot stick")
	}
	if bytes.Contains(grabBody, []byte("grokProc.running = false")) {
		t.Error("grab must not kill an in-flight grok process")
	}

	abortQ := bytes.Index(service, []byte("function abortQueuedSend()"))
	if abortQ < 0 {
		t.Fatal("missing abortQueuedSend")
	}
	abortQEnd := bytes.Index(service[abortQ:], []byte("function buildPromptJson()"))
	if abortQEnd < 0 {
		t.Fatal("abortQueuedSend bounds")
	}
	abortQBody := service[abortQ : abortQ+abortQEnd]
	if !bytes.Contains(abortQBody, []byte("grokProc.running")) {
		t.Error("abortQueuedSend must leave a running grok process")
	}
	if bytes.Contains(abortQBody, []byte("grokProc.running = false")) {
		t.Error("abortQueuedSend must not kill grok")
	}

	for _, want := range []string{
		"function onSendClicked()",
		"objectName: \"send\"",
		"objectName: \"chatPane\"",
		"ControlModifier",
		"chatMessages",
	} {
		if !bytes.Contains(overlay, []byte(want)) {
			t.Errorf("Overlay.qml missing %q", want)
		}
	}
	composerKeys := bytes.Index(overlay, []byte("id: composer"))
	if composerKeys < 0 {
		t.Fatal("missing composer")
	}
	keys := overlay[composerKeys:]
	ret := bytes.Index(keys, []byte("Qt.Key_Return"))
	if ret < 0 {
		t.Fatal("composer must handle Return")
	}
	retBody := keys[ret : ret+400]
	if !bytes.Contains(retBody, []byte("ControlModifier")) {
		t.Error("Send on Return must require Ctrl")
	}

	recIdx := bytes.Index(overlay, []byte("function onRecClicked()"))
	recEnd := bytes.Index(overlay[recIdx:], []byte("function onStopClicked()"))
	stopEnd := bytes.Index(overlay[recIdx+recEnd:], []byte("function onSendClicked()"))
	if recIdx < 0 || recEnd < 0 || stopEnd < 0 {
		t.Fatal("rec/stop/send handler bounds")
	}
	recBody := overlay[recIdx : recIdx+recEnd]
	stopBody := overlay[recIdx+recEnd : recIdx+recEnd+stopEnd]
	if bytes.Contains(recBody, []byte(".send(")) || bytes.Contains(stopBody, []byte(".send(")) {
		t.Error("Rec/Stop must not send")
	}
	trIdx := bytes.Index(overlay, []byte("function onTranscriptReady"))
	trEnd := bytes.Index(overlay[trIdx:], []byte("function onChatUpdated"))
	if trIdx < 0 || trEnd < 0 {
		t.Fatal("onTranscriptReady bounds")
	}
	trBody := overlay[trIdx : trIdx+trEnd]
	if bytes.Contains(trBody, []byte(".send(")) {
		t.Error("transcript must not auto-send")
	}
	sendClick := bytes.Index(overlay, []byte("function onSendClicked()"))
	sendClickEnd := bytes.Index(overlay[sendClick:], []byte("function fittedRect"))
	if sendClick < 0 || sendClickEnd < 0 {
		t.Fatal("onSendClicked bounds")
	}
	sendClickBody := overlay[sendClick : sendClick+sendClickEnd]
	if !bytes.Contains(sendClickBody, []byte("service.send()")) {
		t.Error("onSendClicked must call send")
	}
	if !bytes.Contains(sendClickBody, []byte("burnMarks()")) {
		t.Error("onSendClicked must burn marks before send")
	}
	if !bytes.Contains(sendClickBody, []byte("imagePath")) {
		t.Error("onSendClicked must require a grab")
	}

	if !bytes.Contains(service, []byte("chmod 0600")) {
		t.Error("ask-session must be chmod 0600")
	}
	sessIdx := bytes.Index(service, []byte("id: sessionFile"))
	if sessIdx < 0 {
		t.Fatal("missing sessionFile")
	}
	sessTail := service[sessIdx:]
	sessEnd := bytes.Index(sessTail[1:], []byte("id: "))
	if sessEnd < 0 {
		sessEnd = 800
	}
	sessBody := sessTail[:sessEnd+1]
	if !bytes.Contains(sessBody, []byte("isUuid")) {
		t.Error("session file load must reject non-UUIDs")
	}
	if !bytes.Contains(sessBody, []byte("root.sending")) {
		t.Error("must not reload ask-session while sending")
	}
}

func TestA4Contracts(t *testing.T) {
	dir := pluginDir(t)
	service, err := os.ReadFile(filepath.Join(dir, "Service.qml"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile(filepath.Join(dir, "Overlay.qml"))
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"function parseCapture(",
		"function focusedAddress(",
		"function startSee(",
		"captureMode",
		"grabSeq",
		"grabTag",
		"payloadMaxChars",
		`"kage", "windows"`,
		`"--window"`,
		`"/raw"`,
		"id: windowsProc",
		"no focused window",
		"dirJobGen",
		"windowsJobGen",
		"seeJobGen",
		"function isWindowAddress(",
		"function captureBusy(",
		"function startGrabPipeline(",
		"function captureDone(",
		"pendingGrab",
	} {
		if !bytes.Contains(service, []byte(want)) {
			t.Errorf("Service.qml missing %q", want)
		}
	}
	for _, want := range []string{
		"function grab(",
		"function beginCapture(",
		"function recapturePayload(",
		"function bindService(",
		"property bool capturing",
		"text: \"Recapture\"",
		"root.beginCapture(payloadJson)",
	} {
		if !bytes.Contains(overlay, []byte(want)) {
			t.Errorf("Overlay.qml missing %q", want)
		}
	}
	if bytes.Contains(overlay, []byte("function toggle(")) {
		t.Error("Overlay.qml must not toggle; summon recaptures")
	}
	if bytes.Contains(overlay, []byte("shell.toggle")) {
		t.Error("Overlay.qml must not call shell.toggle")
	}

	parseIdx := bytes.Index(service, []byte("function parseCapture("))
	parseEnd := bytes.Index(service[parseIdx:], []byte("function focusedAddress("))
	if parseIdx < 0 || parseEnd < 0 {
		t.Fatal("parseCapture bounds")
	}
	parseBody := service[parseIdx : parseIdx+parseEnd]
	if !bytes.Contains(parseBody, []byte(`obj.capture === "window"`)) {
		t.Error("parseCapture must honor capture window")
	}
	if !bytes.Contains(parseBody, []byte(`return "monitor"`)) {
		t.Error("parseCapture default is monitor")
	}
	if bytes.Contains(parseBody, []byte("fresh")) {
		t.Error("parseCapture must not implement fresh:true")
	}
	if !bytes.Contains(parseBody, []byte("payloadMaxChars")) {
		t.Error("parseCapture must bound payload size")
	}

	addrIdx := bytes.Index(service, []byte("function focusedAddress("))
	addrEnd := bytes.Index(service[addrIdx:], []byte("function startSee("))
	if addrIdx < 0 || addrEnd < 0 {
		t.Fatal("focusedAddress bounds")
	}
	addrBody := service[addrIdx : addrIdx+addrEnd]
	if !bytes.Contains(addrBody, []byte("w.focus")) {
		t.Error("focusedAddress must use the focused client")
	}
	if !bytes.Contains(addrBody, []byte("isWindowAddress")) {
		t.Error("focusedAddress must accept only 0x addresses from kage windows")
	}

	seeFn := bytes.Index(service, []byte("function startSee("))
	seeFnEnd := bytes.Index(service[seeFn:], []byte("function grab("))
	if seeFn < 0 || seeFnEnd < 0 {
		t.Fatal("startSee bounds")
	}
	startSeeBody := service[seeFn : seeFn+seeFnEnd]
	if !bytes.Contains(startSeeBody, []byte(`["kage", "see"]`)) {
		t.Error("startSee must invoke kage see as argv")
	}
	if bytes.Contains(startSeeBody, []byte(`"sh"`)) || bytes.Contains(startSeeBody, []byte(`"-c"`)) {
		t.Error("startSee must not wrap kage see in sh -c")
	}
	if !bytes.Contains(startSeeBody, []byte(`"--window"`)) {
		t.Error("startSee must pass --window as its own argv")
	}
	if !bytes.Contains(startSeeBody, []byte("windowAddr")) {
		t.Error("startSee must take the focused address")
	}

	grabIdx := bytes.Index(service, []byte("function grab("))
	grabEnd := bytes.Index(service[grabIdx:], []byte("function startRecording()"))
	if grabIdx < 0 || grabEnd < 0 {
		t.Fatal("grab bounds")
	}
	grabBody := service[grabIdx : grabIdx+grabEnd]
	if !bytes.Contains(grabBody, []byte("parseCapture")) {
		t.Error("grab must parse the capture payload")
	}
	pipeIdx := bytes.Index(service, []byte("function startGrabPipeline("))
	pipeEnd := bytes.Index(service[pipeIdx:], []byte("function captureDone("))
	if pipeIdx < 0 || pipeEnd < 0 {
		t.Fatal("startGrabPipeline bounds")
	}
	pipeBody := service[pipeIdx : pipeIdx+pipeEnd]
	if !bytes.Contains(pipeBody, []byte("grabSeq")) {
		t.Error("startGrabPipeline must bump grabSeq so later PNGs are not overwritten")
	}
	if bytes.Contains(grabBody, []byte("sessionId")) {
		t.Error("grab must not touch the Grok session id")
	}
	if bytes.Contains(grabBody, []byte("chatMessages")) {
		t.Error("grab must leave old thread messages in place")
	}
	if bytes.Contains(grabBody, []byte("hyprctl")) {
		t.Error("grab must not call hyprctl; use kage windows")
	}
	if bytes.Contains(grabBody, []byte("seeProc.running = false")) {
		t.Error("grab must not kill an in-flight kage see; grim children wedge screencopy")
	}
	if !bytes.Contains(grabBody, []byte("pendingGrab")) {
		t.Error("grab must queue a recapture while the current see is still running")
	}

	winIdx := bytes.Index(service, []byte("id: windowsProc"))
	if winIdx < 0 {
		t.Fatal("missing windowsProc")
	}
	winEnd := bytes.Index(service[winIdx:], []byte("id: seeProc"))
	if winEnd < 0 {
		t.Fatal("windowsProc not followed by seeProc")
	}
	winBody := service[winIdx : winIdx+winEnd]
	if !bytes.Contains(winBody, []byte("focusedAddress")) {
		t.Error("windowsProc must parse the focused client")
	}
	if !bytes.Contains(winBody, []byte("startSee(addr)")) {
		t.Error("windowsProc must pass the focused address to kage see --window")
	}
	if bytes.Contains(winBody, []byte("startSee(\"\")")) {
		t.Error("window capture must not call kage see without --window")
	}

	seeProcIdx := bytes.Index(service, []byte("id: seeProc"))
	seeProcEnd := bytes.Index(service[seeProcIdx:], []byte("id: recProc"))
	if seeProcIdx < 0 || seeProcEnd < 0 {
		t.Fatal("seeProc bounds")
	}
	seeProcBody := service[seeProcIdx : seeProcIdx+seeProcEnd]
	if !bytes.Contains(seeProcBody, []byte("seeJobGen")) {
		t.Error("seeProc must ignore superseded kage see jobs")
	}
	if !bytes.Contains(seeProcBody, []byte("unlinkAnnotated()")) {
		t.Error("successful grab must still unlink leftover annotated.png")
	}

	dirIdx := bytes.Index(service, []byte("id: ensureDirProc"))
	dirEnd := bytes.Index(service[dirIdx:], []byte("id: windowsProc"))
	if dirIdx < 0 || dirEnd < 0 {
		t.Fatal("ensureDirProc bounds")
	}
	dirBody := service[dirIdx : dirIdx+dirEnd]
	if !bytes.Contains(dirBody, []byte(`captureMode === "window"`)) {
		t.Error("ensureDir must branch to kage windows for capture window")
	}
	if !bytes.Contains(dirBody, []byte(`startSee("")`)) {
		t.Error("monitor capture must call kage see without --window")
	}

	openIdx := bytes.Index(overlay, []byte("function open("))
	openEnd := bytes.Index(overlay[openIdx:], []byte("function grab("))
	if openIdx < 0 || openEnd < 0 {
		t.Fatal("open bounds")
	}
	openBody := overlay[openIdx : openIdx+openEnd]
	if !bytes.Contains(openBody, []byte("beginCapture")) {
		t.Error("open must recapture rather than toggle")
	}
	if bytes.Contains(openBody, []byte("shell.hide")) {
		t.Error("open/summon must not hide via shell.hide")
	}

	grabOv := bytes.Index(overlay, []byte("function grab("))
	grabOvEnd := bytes.Index(overlay[grabOv:], []byte("function close()"))
	if grabOv < 0 || grabOvEnd < 0 {
		t.Fatal("overlay grab bounds")
	}
	grabOvBody := overlay[grabOv : grabOv+grabOvEnd]
	if !bytes.Contains(grabOvBody, []byte("beginCapture")) {
		t.Error("overlay grab must recapture through beginCapture")
	}

	finIdx := bytes.Index(overlay, []byte("function onGrabFinished()"))
	finEnd := bytes.Index(overlay[finIdx:], []byte("function onTranscriptReady"))
	if finIdx < 0 || finEnd < 0 {
		t.Fatal("onGrabFinished bounds")
	}
	finBody := overlay[finIdx : finIdx+finEnd]
	if !bytes.Contains(finBody, []byte("startRecording")) {
		t.Error("recapture must start the mic again")
	}
	if !bytes.Contains(finBody, []byte("!root.service.error")) {
		t.Error("recapture must skip the mic when grab failed")
	}

	recap := bytes.Index(overlay, []byte("text: \"Recapture\""))
	if recap < 0 {
		t.Fatal("missing Recapture control")
	}
	recapBody := overlay[recap : recap+400]
	if !bytes.Contains(recapBody, []byte("root.grab(")) {
		t.Error("Recapture must call overlay grab")
	}
	if bytes.Contains(recapBody, []byte(".send(")) {
		t.Error("Recapture must not send")
	}

	for _, want := range []string{
		"SUPER + SHIFT + W",
		`{"capture":"window"}`,
		"shell call kage.ask grab",
		"raw-2.png",
	} {
		if !bytes.Contains(readme, []byte(want)) {
			t.Errorf("README.md missing %q", want)
		}
	}
	if bytes.Contains(readme, []byte("shell toggle kage.ask")) {
		t.Error("README.md must not suggest toggle")
	}
	for name, body := range map[string][]byte{"Service.qml": service, "Overlay.qml": overlay} {
		for _, banned := range [][]byte{
			[]byte("grim"),
			[]byte("slurp"),
			[]byte("omarchy screenshot"),
			[]byte("tensaku"),
			[]byte("hyprctl"),
		} {
			if bytes.Contains(body, banned) {
				t.Errorf("%s contains %q", name, banned)
			}
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
		"SUPER + SHIFT + W",
		"SUPER + SHIFT + N",
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

func TestA5Contracts(t *testing.T) {
	dir := pluginDir(t)
	service, err := os.ReadFile(filepath.Join(dir, "Service.qml"))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := os.ReadFile(filepath.Join(dir, "Overlay.qml"))
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"function parseFresh(",
		"function startFresh(",
		"function beginFreshUuid(",
		"function setMode(",
		"function inputAllowed(",
		"function parseAllowInput(",
		"function isKageInputTool(",
		"function maybeRecaptureAfterAct(",
		"function noteToolEvent(",
		"actRecaptureRequested",
		"updatedCue",
		`property string mode: "ask"`,
		"KAGE_ALLOW_INPUT",
		"allow_input",
		"config.toml",
		"--allow",
		"uuidForFresh",
		"pendingFresh",
		"tool_call",
		"completed",
	} {
		if !bytes.Contains(service, []byte(want)) {
			t.Errorf("Service.qml missing %q", want)
		}
	}
	for _, want := range []string{
		"parseFresh",
		"startFresh",
		`objectName: "modeAsk"`,
		`objectName: "modeDo"`,
		`text: "Ask"`,
		`text: "Do"`,
		"setMode(\"ask\")",
		"setMode(\"do\")",
		"quietRecapture",
		`"updated"`,
		"onActRecaptureRequested",
		"skipMic",
	} {
		if !bytes.Contains(overlay, []byte(want)) {
			t.Errorf("Overlay.qml missing %q", want)
		}
	}
	for _, want := range []string{
		"SUPER + SHIFT + N",
		`{"fresh":true,"capture":"monitor"}`,
		"ask-session",
		"Ask is the default",
		"KAGE_ALLOW_INPUT=1",
		"allow_input = true",
		"updated",
	} {
		if !bytes.Contains(readme, []byte(want)) {
			t.Errorf("README.md missing %q", want)
		}
	}

	parseIdx := bytes.Index(service, []byte("function parseCapture("))
	parseEnd := bytes.Index(service[parseIdx:], []byte("function focusedAddress("))
	if parseIdx < 0 || parseEnd < 0 {
		t.Fatal("parseCapture bounds")
	}
	parseBody := service[parseIdx : parseIdx+parseEnd]
	if bytes.Contains(parseBody, []byte("fresh")) {
		t.Error("parseCapture must stay capture-only; parseFresh is separate")
	}

	freshIdx := bytes.Index(service, []byte("function parseFresh("))
	freshEnd := bytes.Index(service[freshIdx:], []byte("function parseAllowInput("))
	if freshIdx < 0 || freshEnd < 0 {
		t.Fatal("parseFresh bounds")
	}
	freshBody := service[freshIdx : freshIdx+freshEnd]
	if !bytes.Contains(freshBody, []byte("obj.fresh === true")) {
		t.Error("parseFresh must require fresh:true")
	}
	if !bytes.Contains(freshBody, []byte("payloadMaxChars")) {
		t.Error("parseFresh must bound payload size")
	}

	startFreshIdx := bytes.Index(service, []byte("function startFresh("))
	startFreshEnd := bytes.Index(service[startFreshIdx:], []byte("function beginFreshUuid("))
	if startFreshIdx < 0 || startFreshEnd < 0 {
		t.Fatal("startFresh bounds")
	}
	startFreshBody := service[startFreshIdx : startFreshIdx+startFreshEnd]
	if !bytes.Contains(startFreshBody, []byte("chatMessages = []")) {
		t.Error("fresh must start a new overlay thread")
	}
	if !bytes.Contains(startFreshBody, []byte("grabSeq = 0")) {
		t.Error("fresh must reset grab numbering")
	}
	if !bytes.Contains(startFreshBody, []byte(`mode = "ask"`)) {
		t.Error("new issue defaults to Ask")
	}
	if bytes.Contains(startFreshBody, []byte(`"rm"`)) || bytes.Contains(startFreshBody, []byte("RemoveAll")) {
		t.Error("fresh must not delete the old session")
	}
	if !bytes.Contains(startFreshBody, []byte("pendingFresh")) {
		t.Error("fresh must wait if grok is already running")
	}
	if !bytes.Contains(service, []byte("pendingActRecapture")) {
		t.Error("Do recapture must queue if a grab is already running")
	}

	beginIdx := bytes.Index(service, []byte("function beginFreshUuid("))
	beginEnd := bytes.Index(service[beginIdx:], []byte("function isKageInputTool("))
	if beginIdx < 0 || beginEnd < 0 {
		t.Fatal("beginFreshUuid bounds")
	}
	beginBody := service[beginIdx : beginIdx+beginEnd]
	if !bytes.Contains(beginBody, []byte("uuidgen")) {
		t.Error("fresh must uuidgen a new session id")
	}
	if !bytes.Contains(beginBody, []byte("uuidForFresh")) {
		t.Error("fresh uuidgen must not share the send path")
	}

	uuidIdx := bytes.Index(service, []byte("id: uuidProc"))
	if uuidIdx < 0 {
		t.Fatal("missing uuidProc")
	}
	uuidTail := service[uuidIdx:]
	if !bytes.Contains(uuidTail, []byte("uuidForFresh")) {
		t.Error("uuidProc must persist a fresh session without starting grok")
	}
	freshPersist := bytes.Index(uuidTail, []byte("uuidForFresh"))
	freshPersistEnd := bytes.Index(uuidTail[freshPersist:], []byte("uuidJobGen !== root.sendGen"))
	if freshPersist < 0 || freshPersistEnd < 0 {
		t.Fatal("fresh uuidProc bounds")
	}
	freshUuidBody := uuidTail[freshPersist : freshPersist+freshPersistEnd]
	if !bytes.Contains(freshUuidBody, []byte("persistSessionId")) {
		t.Error("fresh uuidgen must write ask-session")
	}
	if bytes.Contains(freshUuidBody, []byte("startGrokAfterPersist = true")) || bytes.Contains(freshUuidBody, []byte("startGrok(")) {
		t.Error("fresh uuidgen must not start grok")
	}

	modeIdx := bytes.Index(service, []byte("function setMode("))
	modeEnd := bytes.Index(service[modeIdx:], []byte("function startFresh("))
	if modeIdx < 0 || modeEnd < 0 {
		t.Fatal("setMode bounds")
	}
	modeBody := service[modeIdx : modeIdx+modeEnd]
	if !bytes.Contains(modeBody, []byte("inputAllowed()")) {
		t.Error("Do must check the kage input gate")
	}
	if !bytes.Contains(modeBody, []byte(`mode = "ask"`)) {
		t.Error("missing gate must stay on Ask")
	}
	if !bytes.Contains(modeBody, []byte("input not allowed")) {
		t.Error("missing gate must set a clear overlay error")
	}
	if bytes.Contains(modeBody, []byte("--yes")) || bytes.Contains(modeBody, []byte("KAGE_ALLOW_INPUT=1\"")) {
		t.Error("setMode must not yolo --yes")
	}

	allowIdx := bytes.Index(service, []byte("function inputAllowed("))
	allowEnd := bytes.Index(service[allowIdx:], []byte("function setMode("))
	if allowIdx < 0 || allowEnd < 0 {
		t.Fatal("inputAllowed bounds")
	}
	allowBody := service[allowIdx : allowIdx+allowEnd]
	if !bytes.Contains(allowBody, []byte("KAGE_ALLOW_INPUT")) || !bytes.Contains(allowBody, []byte(`=== "1"`)) {
		t.Error("inputAllowed must honor KAGE_ALLOW_INPUT=1")
	}
	if !bytes.Contains(allowBody, []byte("parseAllowInput")) {
		t.Error("inputAllowed must read allow_input from config")
	}

	cfgIdx := bytes.Index(service, []byte("function parseAllowInput("))
	cfgEnd := bytes.Index(service[cfgIdx:], []byte("function inputAllowed("))
	if cfgIdx < 0 || cfgEnd < 0 {
		t.Fatal("parseAllowInput bounds")
	}
	cfgBody := service[cfgIdx : cfgIdx+cfgEnd]
	if !bytes.Contains(cfgBody, []byte(`charAt(0) === "#"`)) {
		t.Error("parseAllowInput must ignore comments")
	}
	if !bytes.Contains(cfgBody, []byte(`v === "true"`)) || !bytes.Contains(cfgBody, []byte(`v === "1"`)) {
		t.Error("parseAllowInput must accept true and 1")
	}

	startIdx := bytes.Index(service, []byte("function startGrok("))
	startEnd := bytes.Index(service[startIdx:], []byte("function handleStreamLine("))
	if startIdx < 0 || startEnd < 0 {
		t.Fatal("startGrok bounds")
	}
	startBody := service[startIdx : startIdx+startEnd]
	if !bytes.Contains(startBody, []byte("doInput")) {
		t.Error("startGrok must branch Ask vs Do")
	}
	denyIdx := bytes.Index(startBody, []byte("} else {"))
	if denyIdx < 0 {
		t.Fatal("startGrok missing Ask else")
	}
	askBody := startBody[denyIdx:]
	if !bytes.Contains(askBody, []byte("--deny")) || !bytes.Contains(askBody, []byte("names[i]")) {
		t.Error("Ask must still deny kage click/type/press/hotkey")
	}
	if bytes.Contains(askBody, []byte("--allow")) {
		t.Error("Ask else branch must not --allow input tools")
	}
	doBody := startBody[:denyIdx]
	if !bytes.Contains(doBody, []byte("--allow")) || !bytes.Contains(doBody, []byte("kage click")) {
		t.Error("Do must --allow kage input tools when the gate is open")
	}
	if bytes.Contains(startBody, []byte("bypassPermissions")) || bytes.Contains(startBody, []byte("--always-approve")) || bytes.Contains(startBody, []byte("--yolo")) || bytes.Contains(startBody, []byte("--yes")) {
		t.Error("Do must not yolo desktop input")
	}
	if !bytes.Contains(startBody, []byte("inputAllowed()")) {
		t.Error("Do send must re-check the gate")
	}

	recapIdx := bytes.Index(service, []byte("function maybeRecaptureAfterAct("))
	recapEnd := bytes.Index(service[recapIdx:], []byte("Process {"))
	if recapIdx < 0 || recapEnd < 0 {
		t.Fatal("maybeRecaptureAfterAct bounds")
	}
	recapBody := service[recapIdx : recapIdx+recapEnd]
	if !bytes.Contains(recapBody, []byte(`mode !== "do"`)) {
		t.Error("recapture-after-act is Do only")
	}
	if !bytes.Contains(recapBody, []byte("updatedCue = true")) {
		t.Error("successful Do input must set the updated cue")
	}
	if !bytes.Contains(recapBody, []byte("actRecaptureRequested")) {
		t.Error("successful Do input must recapture through the overlay")
	}

	toolIdx := bytes.Index(service, []byte("function isKageInputTool("))
	toolEnd := bytes.Index(service[toolIdx:], []byte("function dropInputId("))
	if toolIdx < 0 || toolEnd < 0 {
		t.Fatal("isKageInputTool bounds")
	}
	toolBody := service[toolIdx : toolIdx+toolEnd]
	if !bytes.Contains(toolBody, []byte("kage click")) || !bytes.Contains(toolBody, []byte("kage_click")) {
		t.Error("must detect kage click/type/press/hotkey tools")
	}
	if !bytes.Contains(toolBody, []byte("kage see")) || !bytes.Contains(toolBody, []byte("return false")) {
		t.Error("kage see must not count as an input tool")
	}

	noteIdx := bytes.Index(service, []byte("function noteToolEvent("))
	noteEnd := bytes.Index(service[noteIdx:], []byte("function maybeRecaptureAfterAct("))
	if noteIdx < 0 || noteEnd < 0 {
		t.Fatal("noteToolEvent bounds")
	}
	noteBody := service[noteIdx : noteIdx+noteEnd]
	if !bytes.Contains(noteBody, []byte(`st === "completed"`)) {
		t.Error("recapture only after a successful tool")
	}
	if !bytes.Contains(noteBody, []byte("failed")) {
		t.Error("failed input tools must not recapture")
	}

	grabIdx := bytes.Index(service, []byte("function grab("))
	grabEnd := bytes.Index(service[grabIdx:], []byte("function startRecording()"))
	if grabIdx < 0 || grabEnd < 0 {
		t.Fatal("grab bounds")
	}
	grabBody := service[grabIdx : grabIdx+grabEnd]
	if bytes.Contains(grabBody, []byte("startFresh")) || bytes.Contains(grabBody, []byte("sessionId")) {
		t.Error("grab must stay recapture-only; Overlay calls startFresh")
	}

	beginCap := bytes.Index(overlay, []byte("function beginCapture("))
	beginCapEnd := bytes.Index(overlay[beginCap:], []byte("function open("))
	if beginCap < 0 || beginCapEnd < 0 {
		t.Fatal("beginCapture bounds")
	}
	beginCapBody := overlay[beginCap : beginCap+beginCapEnd]
	if !bytes.Contains(beginCapBody, []byte("parseFresh")) || !bytes.Contains(beginCapBody, []byte("startFresh")) {
		t.Error("summon with fresh:true must start a new session before grab")
	}

	finIdx := bytes.Index(overlay, []byte("function onGrabFinished()"))
	finEnd := bytes.Index(overlay[finIdx:], []byte("function onTranscriptReady"))
	if finIdx < 0 || finEnd < 0 {
		t.Fatal("onGrabFinished bounds")
	}
	finBody := overlay[finIdx : finIdx+finEnd]
	if !bytes.Contains(finBody, []byte("skipMic")) {
		t.Error("Do recapture-after-act must not auto-start the mic")
	}
	if !bytes.Contains(finBody, []byte("startRecording")) {
		t.Error("user recapture must still start the mic")
	}

	for _, banned := range []string{"--always-approve", "--yolo", "bypassPermissions", "kage agent", "bar-widget", "this project"} {
		if bytes.Contains(service, []byte(banned)) || bytes.Contains(overlay, []byte(banned)) {
			t.Errorf("plugin contains banned %q", banned)
		}
	}
	if bytes.Contains(service, []byte(`"--yes"`)) {
		t.Error("plugin must not pass --yes to bypass the input gate")
	}
}
