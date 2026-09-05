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
