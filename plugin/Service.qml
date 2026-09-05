import QtQuick
import Quickshell
import Quickshell.Io

Item {
  id: root

  property var shell: null
  property var manifest: null
  property string omarchyPath: Quickshell.env("OMARCHY_PATH")

  readonly property string kageDir: {
    var dir = Quickshell.env("XDG_RUNTIME_DIR")
    if (dir && dir.length)
      return dir + "/kage"
    var uid = Quickshell.env("UID")
    if (!uid || !uid.length)
      uid = Quickshell.env("EUID")
    if (!uid || !uid.length)
      uid = "0"
    return "/tmp/kage-" + uid
  }
  readonly property string askRoot: kageDir + "/ask"
  property string issueId: "current"
  readonly property string issueDir: askRoot + "/" + issueId
  property int grabSeq: 0
  readonly property string grabTag: grabSeq <= 1 ? "" : ("-" + grabSeq)
  readonly property string rawPath: issueDir + "/raw" + grabTag + ".png"
  readonly property string annotatedPath: issueDir + "/annotated" + grabTag + ".png"
  readonly property string annotatedTmpPath: issueDir + "/annotated" + grabTag + ".png.tmp"
  readonly property string snapshotJsonPath: issueDir + "/snapshot" + grabTag + ".json"
  readonly property string promptTxtPath: issueDir + "/prompt.txt"
  readonly property string statusJsonPath: issueDir + "/status.json"
  readonly property string grokErrPath: issueDir + "/grok.err"
  readonly property string wavPath: issueDir + "/rec.wav"
  readonly property int composerMaxChars: 32000
  readonly property string homeDir: {
    var h = Quickshell.env("HOME")
    return h && h.length ? h : "/home"
  }
  readonly property string configDir: homeDir + "/.config/kage"
  readonly property string sessionFilePath: configDir + "/ask-session"

  property bool grabbing: false
  property bool recording: false
  property bool transcribing: false
  property bool transcribeWhenRecEnds: false
  property bool recAfterTranscribe: false
  property int wavGen: 0
  property int transcribeGen: 0
  property int burnGen: 0
  property int burnJobGen: 0
  property int grabGen: 0
  property int dirJobGen: 0
  property int windowsJobGen: 0
  property int seeJobGen: 0
  property int unlinkJobGen: 0
  property string captureMode: "monitor"
  readonly property int payloadMaxChars: 4096
  property var pendingStrokes: null
  property bool hasMarks: false
  property string imagePath: ""
  property string imageToken: ""
  property var snapshot: null
  property string error: ""
  property string composerText: ""
  property bool sending: false
  property string sessionId: ""
  property var chatMessages: []
  property string streamBuf: ""
  property int sendGen: 0
  property int grokJobGen: 0
  property string pendingPromptJson: ""
  property bool pendingResume: false
  property bool sendQueued: false
  property bool startGrokAfterPersist: false
  property int persistJobGen: 0
  property int persistGen: 0
  property int uuidJobGen: 0
  property bool pendingGrab: false
  property bool pendingActRecapture: false
  property string pendingStatusText: ""
  property string mode: "ask"
  property string configText: ""
  property int freshGen: 0
  property bool uuidForFresh: false
  property bool pendingFresh: false
  property string pendingInputIds: ""
  property bool updatedCue: false

  readonly property string configFilePath: configDir + "/config.toml"

  signal grabFinished()
  signal transcriptReady(string text)
  signal chatUpdated()
  signal sendFinished()
  signal actRecaptureRequested()

  function parseCapture(payloadJson) {
    var raw = String(payloadJson || "")
    if (raw.length > root.payloadMaxChars)
      raw = raw.substring(0, root.payloadMaxChars)
    raw = raw.trim()
    if (!raw.length)
      return "monitor"
    var obj = null
    try {
      obj = JSON.parse(raw)
    } catch (e) {
      return "monitor"
    }
    if (!obj || typeof obj !== "object")
      return "monitor"
    if (obj.capture === "window")
      return "window"
    return "monitor"
  }

  function focusedAddress(text) {
    var obj = null
    try {
      obj = JSON.parse(String(text || "").trim())
    } catch (e) {
      return ""
    }
    var list = obj && obj.windows
    if (!Array.isArray(list))
      return ""
    for (var i = 0; i < list.length; i++) {
      var w = list[i]
      if (!w || !w.focus)
        continue
      var addr = String(w.address || "")
      if (root.isWindowAddress(addr))
        return addr
    }
    return ""
  }

  function isWindowAddress(addr) {
    if (addr.length < 3 || addr.substring(0, 2) !== "0x")
      return false
    for (var i = 2; i < addr.length; i++) {
      var c = addr.charAt(i)
      if (!((c >= "0" && c <= "9") || (c >= "a" && c <= "f") || (c >= "A" && c <= "F")))
        return false
    }
    return true
  }

  function startSee(windowAddr) {
    var cmd = ["kage", "see"]
    if (windowAddr && windowAddr.length) {
      cmd.push("--window")
      cmd.push(windowAddr)
    }
    cmd.push("--path")
    cmd.push(root.rawPath)
    root.seeJobGen = root.grabGen
    seeProc.command = cmd
    seeProc.running = true
  }

  function captureBusy() {
    return ensureDirProc.running || windowsProc.running || seeProc.running || rmAnnotatedProc.running
  }

  function startGrabPipeline() {
    root.grabGen += 1
    root.grabSeq += 1
    root.dirJobGen = root.grabGen
    ensureDirProc.command = ["install", "-d", "-m", "0700", root.kageDir, root.askRoot, root.issueDir]
    ensureDirProc.running = true
  }

  function captureDone() {
    if (root.pendingGrab) {
      root.pendingGrab = false
      root.error = ""
      root.hasMarks = false
      root.pendingStrokes = null
      root.burnGen += 1
      root.startGrabPipeline()
      return
    }
    if (root.pendingActRecapture && root.mode === "do") {
      root.pendingActRecapture = false
      root.updatedCue = true
      root.error = ""
      root.hasMarks = false
      root.pendingStrokes = null
      root.burnGen += 1
      root.startGrabPipeline()
      return
    }
    root.grabbing = false
    root.grabFinished()
  }

  function grab(payloadJson) {
    abortQueuedSend()
    root.grabbing = true
    root.error = ""
    root.hasMarks = false
    root.pendingStrokes = null
    root.burnGen += 1
    root.captureMode = root.parseCapture(payloadJson)
    stopMic()
    if (burnProc.running)
      burnProc.running = false
    if (mvBurnProc.running)
      mvBurnProc.running = false
    if (root.captureBusy()) {
      root.pendingGrab = true
      return
    }
    root.startGrabPipeline()
  }

  function startRecording() {
    if (root.recording || recProc.running)
      return
    if (transcribeProc.running) {
      root.recAfterTranscribe = true
      return
    }
    root.recAfterTranscribe = false
    root.transcribeWhenRecEnds = false
    root.wavGen += 1
    recProc.command = ["pw-record", "--rate", "16000", "--channels", "1", "--format", "s16", root.wavPath]
    recProc.running = true
    root.recording = true
  }

  function unlinkWav() {
    if (rmWavProc.running)
      rmWavProc.running = false
    rmWavProc.command = ["rm", "-f", root.wavPath]
    rmWavProc.running = true
  }

  function cancelRecording() {
    root.transcribeWhenRecEnds = false
    root.recAfterTranscribe = false
    root.recording = false
    root.transcribing = false
    if (transcribeProc.running)
      transcribeProc.running = false
    if (recProc.running)
      recProc.running = false
    else
      unlinkWav()
  }

  function stopMic() {
    cancelRecording()
  }

  function stopAndTranscribe() {
    if (!root.recording && !recProc.running) {
      if (!root.transcribing)
        unlinkWav()
      return
    }
    root.recording = false
    root.transcribeWhenRecEnds = true
    if (recProc.running)
      recProc.running = false
    else
      beginTranscribe()
  }

  function beginTranscribe() {
    root.transcribeWhenRecEnds = false
    root.transcribing = true
    root.transcribeGen = root.wavGen
    transcribeProc.command = ["voxtype", "transcribe", root.wavPath]
    transcribeProc.running = true
  }

  function markAnnotated() {
    root.hasMarks = true
  }

  function unlinkAnnotated() {
    if (rmAnnotatedProc.running)
      return
    root.unlinkJobGen = root.grabGen
    rmAnnotatedProc.command = ["rm", "-f", root.annotatedPath, root.annotatedTmpPath]
    rmAnnotatedProc.running = true
  }

  function strokeWidth(w) {
    return Math.max(3, Math.round(w / 400))
  }

  function payloadImagePath() {
    if (root.hasMarks && root.annotatedPath.length)
      return root.annotatedPath
    return root.rawPath
  }

  function isUuid(id) {
    var s = String(id || "")
    return /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(s)
  }

  function burnBusy() {
    return burnProc.running || mvBurnProc.running || rmTmpProc.running || !!root.pendingStrokes
  }

  function grabReady() {
    return !root.grabbing && !!root.snapshot && !!root.imagePath
  }

  function appendChat(role, text) {
    var next = root.chatMessages.slice()
    next.push({ role: role, text: text })
    root.chatMessages = next
    root.chatUpdated()
  }

  function setLastAssistant(text) {
    var next = root.chatMessages.slice()
    if (next.length && next[next.length - 1].role === "assistant")
      next[next.length - 1] = { role: "assistant", text: text }
    else
      next.push({ role: "assistant", text: text })
    root.chatMessages = next
    root.chatUpdated()
  }

  function persistSessionId(id) {
    if (!root.isUuid(id))
      return
    root.sessionId = id
    root.persistGen += 1
    root.persistJobGen = root.persistGen
    if (writeSessionProc.running)
      writeSessionProc.running = false
    writeSessionProc.command = [
      "sh", "-c",
      "install -d -m 0700 \"$1\" && umask 077 && printf '%s\\n' \"$2\" > \"$3\" && chmod 0600 \"$3\"",
      "kage-ask-session",
      root.configDir,
      id,
      root.sessionFilePath
    ]
    writeSessionProc.running = true
  }

  function abortSend(msg) {
    root.sendQueued = false
    root.startGrokAfterPersist = false
    root.sending = false
    root.error = msg
    root.sendFinished()
  }

  function abortQueuedSend() {
    if (grokProc.running)
      return
    if (!root.sending && !root.sendQueued && !root.startGrokAfterPersist)
      return
    root.sendGen += 1
    if (uuidProc.running)
      uuidProc.running = false
    if (writeSessionProc.running)
      writeSessionProc.running = false
    abortSend("send cancelled")
  }

  function buildPromptJson() {
    var img = root.payloadImagePath()
    var body = "Screenshot: " + img + "\nSnapshot JSON: " + root.snapshotJsonPath + "\n\n" + String(root.composerText || "")
    return JSON.stringify([{ type: "text", text: body }])
  }

  function send() {
    if (root.sending || grokProc.running)
      return
    if (!root.grabReady()) {
      root.error = "nothing to send"
      return
    }
    if (String(root.composerText || "").length > root.composerMaxChars) {
      root.error = "message too long"
      return
    }
    stopMic()
    root.error = ""
    root.sending = true
    root.sendGen += 1
    root.streamBuf = ""
    if (root.burnBusy()) {
      root.sendQueued = true
      return
    }
    launchSend()
  }

  function launchSend() {
    promptFile.setText(String(root.composerText || ""))
    var userText = String(root.composerText || "")
    appendChat("user", userText)
    root.pendingPromptJson = root.buildPromptJson()
    root.pendingResume = root.isUuid(root.sessionId)
    if (root.pendingResume) {
      startGrok(root.sessionId, true)
      return
    }
    root.uuidJobGen = root.sendGen
    uuidProc.command = ["uuidgen"]
    uuidProc.running = true
  }

  function maybeLaunchQueuedSend() {
    if (!root.sendQueued)
      return
    if (root.burnJobGen !== root.burnGen)
      return
    if (root.burnBusy())
      return
    root.sendQueued = false
    if (!root.grabReady()) {
      abortSend("nothing to send")
      return
    }
    launchSend()
  }

  function startGrok(id, resume) {
    var doInput = root.mode === "do" && root.inputAllowed()
    if (root.mode === "do" && !doInput) {
      abortSend("input not allowed; need KAGE_ALLOW_INPUT=1 or allow_input = true in config")
      return
    }
    var cmd = [
      "grok",
      "--prompt-json", root.pendingPromptJson,
      "--output-format", "streaming-json",
      "--cwd", root.homeDir,
      "--permission-mode", "dontAsk"
    ]
    var names = ["kage click", "kage type", "kage press", "kage hotkey"]
    var mcp = ["mcp__kage__kage_click", "mcp__kage__kage_type", "mcp__kage__kage_press", "mcp__kage__kage_hotkey"]
    var bash = ["Bash(kage click*)", "Bash(kage type*)", "Bash(kage press*)", "Bash(kage hotkey*)"]
    var i
    if (doInput) {
      for (i = 0; i < names.length; i++) {
        cmd.push("--allow")
        cmd.push(names[i])
      }
      for (i = 0; i < mcp.length; i++) {
        cmd.push("--allow")
        cmd.push(mcp[i])
      }
      for (i = 0; i < bash.length; i++) {
        cmd.push("--allow")
        cmd.push(bash[i])
      }
    } else {
      for (i = 0; i < names.length; i++) {
        cmd.push("--deny")
        cmd.push(names[i])
      }
      for (i = 0; i < mcp.length; i++) {
        cmd.push("--deny")
        cmd.push(mcp[i])
      }
      for (i = 0; i < bash.length; i++) {
        cmd.push("--deny")
        cmd.push(bash[i])
      }
      cmd.push("--disallowed-tools")
      cmd.push("kage__kage_click,kage__kage_type,kage__kage_press,kage__kage_hotkey")
    }
    if (resume) {
      cmd.push("--resume")
      cmd.push(id)
    } else {
      cmd.push("--session-id")
      cmd.push(id)
    }
    root.grokJobGen = root.sendGen
    grokProc.workingDirectory = root.homeDir
    grokProc.command = cmd
    grokProc.running = true
  }

  function handleStreamLine(line) {
    var s = String(line || "").trim()
    if (!s.length)
      return
    var obj = null
    try {
      obj = JSON.parse(s)
    } catch (e) {
      return
    }
    if (!obj || typeof obj !== "object")
      return
    var upd = root.streamUpdate(obj)
    root.noteToolEvent(upd)
    if (obj.type === "text" && obj.data != null)
      root.streamBuf += String(obj.data)
    else if (obj.type === "assistant" && obj.text)
      root.streamBuf += String(obj.text)
    else if (obj.delta && obj.delta.text)
      root.streamBuf += String(obj.delta.text)
    else if (upd && upd.sessionUpdate === "agent_message_chunk" && upd.content && upd.content.text)
      root.streamBuf += String(upd.content.text)
    if (root.streamBuf.length)
      setLastAssistant(root.streamBuf)
  }

  function burnMarks(strokes) {
    if (!root.snapshot || !root.rawPath.length)
      return
    var w = root.snapshot.width || 0
    var h = root.snapshot.height || 0
    if (w <= 0 || h <= 0)
      return
    var list = strokes || []
    if (burnProc.running || mvBurnProc.running || rmAnnotatedProc.running || rmTmpProc.running) {
      root.pendingStrokes = list
      return
    }
    startBurn(list)
  }

  function startBurn(list) {
    var w = root.snapshot.width || 0
    var h = root.snapshot.height || 0
    var strokeW = root.strokeWidth(w)
    var cmd = ["magick", root.rawPath, "-stroke", "#e23d28", "-strokewidth", String(strokeW), "-fill", "none"]
    var drew = false
    for (var s = 0; s < list.length; s++) {
      var pts = list[s]
      if (!pts || pts.length < 2)
        continue
      var parts = []
      for (var i = 0; i < pts.length; i++) {
        var x = Math.round(Number(pts[i].x) * w)
        var y = Math.round(Number(pts[i].y) * h)
        parts.push(x + "," + y)
      }
      cmd.push("-draw")
      cmd.push("polyline " + parts.join(" "))
      drew = true
    }
    if (!drew)
      return
    cmd.push(root.annotatedTmpPath)
    root.burnJobGen = root.burnGen
    burnProc.command = cmd
    burnProc.running = true
  }

  function flushPendingBurn() {
    var pending = root.pendingStrokes
    root.pendingStrokes = null
    if (pending)
      startBurn(pending)
  }

  function parseFresh(payloadJson) {
    var raw = String(payloadJson || "")
    if (raw.length > root.payloadMaxChars)
      raw = raw.substring(0, root.payloadMaxChars)
    raw = raw.trim()
    if (!raw.length)
      return false
    var obj = null
    try {
      obj = JSON.parse(raw)
    } catch (e) {
      return false
    }
    if (!obj || typeof obj !== "object")
      return false
    return obj.fresh === true
  }

  function parseAllowInput(text) {
    var lines = String(text || "").split("\n")
    var allow = false
    for (var i = 0; i < lines.length; i++) {
      var line = String(lines[i] || "").trim()
      if (!line.length || line.charAt(0) === "#")
        continue
      var eq = line.indexOf("=")
      if (eq < 0)
        continue
      var key = line.substring(0, eq).trim()
      if (key !== "allow_input")
        continue
      var v = line.substring(eq + 1).trim()
      if (v.length >= 2) {
        var q = v.charAt(0)
        if ((q === "\"" || q === "'") && v.charAt(v.length - 1) === q)
          v = v.substring(1, v.length - 1)
      }
      allow = v === "true" || v === "1"
    }
    return allow
  }

  function inputAllowed() {
    if (Quickshell.env("KAGE_ALLOW_INPUT") === "1")
      return true
    return root.parseAllowInput(root.configText)
  }

  function setMode(next) {
    var want = String(next || "") === "do" ? "do" : "ask"
    if (want === "do" && !root.inputAllowed()) {
      root.error = "input not allowed; need KAGE_ALLOW_INPUT=1 or allow_input = true in config"
      root.mode = "ask"
      writeModeStatus()
      return
    }
    root.error = ""
    root.mode = want
    writeModeStatus()
  }

  function writeModeStatus() {
    root.pendingStatusText = JSON.stringify({
      mode: root.mode,
      error: String(root.error || ""),
      inputAllowed: root.inputAllowed()
    }) + "\n"
    if (statusDirProc.running)
      statusDirProc.running = false
    statusDirProc.command = ["install", "-d", "-m", "0700", root.kageDir, root.askRoot, root.issueDir]
    statusDirProc.running = true
  }

  function startFresh() {
    abortQueuedSend()
    root.chatMessages = []
    root.chatUpdated()
    root.grabSeq = 0
    root.mode = "ask"
    root.pendingInputIds = ""
    root.updatedCue = false
    if (root.sending || grokProc.running || uuidProc.running) {
      root.pendingFresh = true
      return
    }
    beginFreshUuid()
  }

  function beginFreshUuid() {
    root.pendingFresh = false
    root.freshGen += 1
    root.uuidForFresh = true
    root.uuidJobGen = root.freshGen
    uuidProc.command = ["uuidgen"]
    uuidProc.running = true
  }

  function isKageInputTool(upd) {
    var parts = []
    parts.push(String(upd && upd.title || ""))
    var meta = upd && upd._meta && upd._meta["x.ai/tool"]
    if (meta && meta.name)
      parts.push(String(meta.name))
    var rawIn = upd && upd.rawInput
    if (rawIn) {
      parts.push(String(rawIn.command || ""))
      parts.push(String(rawIn.tool || ""))
    }
    var s = parts.join(" ").toLowerCase()
    if (s.indexOf("kage_see") >= 0 || s.indexOf("kage see") >= 0)
      return false
    return s.indexOf("kage click") >= 0 || s.indexOf("kage_click") >= 0
      || s.indexOf("kage type") >= 0 || s.indexOf("kage_type") >= 0
      || s.indexOf("kage press") >= 0 || s.indexOf("kage_press") >= 0
      || s.indexOf("kage hotkey") >= 0 || s.indexOf("kage_hotkey") >= 0
  }

  function dropInputId(id) {
    var want = String(id || "")
    if (!want.length)
      return
    var cur = " " + String(root.pendingInputIds || "").trim() + " "
    var next = cur.split(" " + want + " ").join(" ")
    root.pendingInputIds = next.replace(/ +/g, " ").trim()
  }

  function streamUpdate(obj) {
    if (!obj || typeof obj !== "object")
      return obj
    if (obj.params && obj.params.update && typeof obj.params.update === "object")
      return obj.params.update
    if (obj.update && typeof obj.update === "object" && obj.update.sessionUpdate)
      return obj.update
    return obj
  }

  function noteToolEvent(upd) {
    if (!upd || typeof upd !== "object")
      return
    var kind = String(upd.sessionUpdate || upd.type || "")
    var id = String(upd.toolCallId || "")
    if (!id.length)
      return
    if (kind === "tool_call" || kind === "tool_call_update") {
      var cur = " " + String(root.pendingInputIds || "").trim() + " "
      if (root.isKageInputTool(upd) && cur.indexOf(" " + id + " ") < 0)
        root.pendingInputIds = (String(root.pendingInputIds || "").trim() + " " + id).trim()
    }
    var st = String(upd.status || "").toLowerCase()
    if (st === "completed")
      root.maybeRecaptureAfterAct(id)
    else if (st === "failed" || st === "cancelled")
      root.dropInputId(id)
  }

  function maybeRecaptureAfterAct(id) {
    var cur = " " + String(root.pendingInputIds || "").trim() + " "
    if (cur.indexOf(" " + id + " ") < 0)
      return
    root.dropInputId(id)
    if (root.mode !== "do")
      return
    if (root.grabbing) {
      root.pendingActRecapture = true
      return
    }
    root.updatedCue = true
    root.actRecaptureRequested()
  }

  Process {
    id: ensureDirProc
    onExited: function(exitCode) {
      if (root.dirJobGen !== root.grabGen)
        return
      if (exitCode !== 0) {
        root.error = "could not create " + root.askRoot
        root.captureDone()
        return
      }
      if (root.captureMode === "window") {
        root.windowsJobGen = root.grabGen
        windowsProc.command = ["kage", "windows"]
        windowsProc.running = true
        return
      }
      root.startSee("")
    }
  }

  Process {
    id: windowsProc
    stdout: StdioCollector {
      id: windowsOut
      waitForEnd: true
    }
    stderr: StdioCollector {
      id: windowsErr
      waitForEnd: true
    }
    onExited: function(exitCode) {
      if (root.windowsJobGen !== root.grabGen)
        return
      if (exitCode !== 0) {
        var wmsg = String(windowsErr.text || windowsOut.text || "").trim()
        root.error = wmsg.length ? wmsg : ("kage windows failed (" + exitCode + ")")
        root.captureDone()
        return
      }
      var addr = root.focusedAddress(windowsOut.text)
      if (!addr.length) {
        root.error = "no focused window"
        root.captureDone()
        return
      }
      root.startSee(addr)
    }
  }

  Process {
    id: seeProc
    stdout: StdioCollector {
      id: seeOut
      waitForEnd: true
    }
    stderr: StdioCollector {
      id: seeErr
      waitForEnd: true
    }
    onExited: function(exitCode) {
      if (root.seeJobGen !== root.grabGen)
        return
      if (exitCode !== 0) {
        var msg = String(seeErr.text || seeOut.text || "").trim()
        root.error = msg.length ? msg : ("kage see failed (" + exitCode + ")")
        root.captureDone()
        return
      }
      var snap = null
      try {
        snap = JSON.parse(String(seeOut.text || "").trim())
      } catch (e) {
        snap = null
      }
      if (!snap || !snap.path) {
        root.error = "kage see returned no path"
        root.captureDone()
        return
      }
      root.snapshot = snap
      root.imagePath = snap.path
      root.imageToken = Date.now().toString()
      snapshotFile.setText(JSON.stringify(snap, null, 2) + "\n")
      unlinkAnnotated()
    }
  }

  Process {
    id: recProc
    onExited: function() {
      root.recording = false
      if (root.transcribeWhenRecEnds) {
        beginTranscribe()
        return
      }
      unlinkWav()
    }
  }

  Process {
    id: transcribeProc
    stdout: StdioCollector {
      id: transcribeOut
      waitForEnd: true
    }
    stderr: StdioCollector {
      id: transcribeErr
      waitForEnd: true
    }
    onExited: function(exitCode) {
      root.transcribing = false
      var ownWav = root.wavGen === root.transcribeGen && !recProc.running
      if (ownWav)
        unlinkWav()
      if (exitCode !== 0) {
        var msg = String(transcribeErr.text || transcribeOut.text || "").trim()
        root.error = msg.length ? msg : ("voxtype transcribe failed (" + exitCode + ")")
      } else {
        var text = String(transcribeOut.text || "").trim()
        if (text.length) {
          if (root.composerText.length)
            root.composerText = root.composerText + (root.composerText.endsWith(" ") ? "" : " ") + text
          else
            root.composerText = text
          root.transcriptReady(text)
        }
      }
      if (root.recAfterTranscribe) {
        root.recAfterTranscribe = false
        startRecording()
      }
    }
  }

  Process {
    id: rmWavProc
  }

  Process {
    id: rmAnnotatedProc
    onExited: function() {
      if (root.unlinkJobGen !== root.grabGen)
        return
      root.captureDone()
      flushPendingBurn()
    }
  }

  Process {
    id: burnProc
    stderr: StdioCollector {
      id: burnErr
      waitForEnd: true
    }
    onExited: function(exitCode) {
      if (root.burnJobGen !== root.burnGen) {
        flushPendingBurn()
        return
      }
      if (exitCode !== 0) {
        var msg = String(burnErr.text || "").trim()
        root.error = msg.length ? msg : ("magick burn failed (" + exitCode + ")")
        if (root.sendQueued)
          abortSend(root.error)
        if (!rmTmpProc.running) {
          rmTmpProc.command = ["rm", "-f", root.annotatedTmpPath]
          rmTmpProc.running = true
        }
        return
      }
      mvBurnProc.command = ["mv", "-f", root.annotatedTmpPath, root.annotatedPath]
      mvBurnProc.running = true
    }
  }

  Process {
    id: rmTmpProc
    onExited: function() {
      flushPendingBurn()
      maybeLaunchQueuedSend()
    }
  }

  Process {
    id: mvBurnProc
    onExited: function(exitCode) {
      if (root.burnJobGen !== root.burnGen) {
        flushPendingBurn()
        return
      }
      if (exitCode !== 0) {
        root.error = "could not install annotated.png"
        if (root.sendQueued)
          abortSend(root.error)
        flushPendingBurn()
        return
      }
      root.markAnnotated()
      flushPendingBurn()
      maybeLaunchQueuedSend()
    }
  }

  FileView {
    id: snapshotFile
    path: root.snapshotJsonPath
    printErrors: false
  }

  FileView {
    id: promptFile
    path: root.promptTxtPath
    printErrors: false
  }

  FileView {
    id: statusFile
    path: root.statusJsonPath
    printErrors: false
  }

  Process {
    id: statusDirProc
    onExited: function(exitCode) {
      if (exitCode !== 0 || !root.pendingStatusText.length)
        return
      statusFile.setText(root.pendingStatusText)
    }
  }

  FileView {
    id: sessionFile
    path: root.sessionFilePath
    printErrors: false
    watchChanges: true
    onLoaded: function() {
      var id = String(text() || "").trim()
      if (root.isUuid(id))
        root.sessionId = id
    }
    onFileChanged: {
      if (root.sending)
        return
      reload()
    }
    Component.onCompleted: reload()
  }

  Process {
    id: writeSessionProc
    onExited: function(exitCode) {
      if (root.persistJobGen !== root.persistGen)
        return
      if (exitCode !== 0) {
        if (root.startGrokAfterPersist)
          abortSend("could not write ask-session")
        else
          root.error = "could not write ask-session"
        return
      }
      if (!root.startGrokAfterPersist)
        return
      root.startGrokAfterPersist = false
      startGrok(root.sessionId, false)
    }
  }

  Process {
    id: uuidProc
    stdout: StdioCollector {
      id: uuidOut
      waitForEnd: true
    }
    onExited: function(exitCode) {
      if (root.uuidForFresh) {
        if (root.uuidJobGen !== root.freshGen)
          return
        root.uuidForFresh = false
        if (exitCode !== 0) {
          root.sending = false
          root.error = "uuidgen failed"
          return
        }
        var freshId = String(uuidOut.text || "").trim()
        if (!root.isUuid(freshId)) {
          root.error = "uuidgen returned invalid id"
          return
        }
        persistSessionId(freshId)
        return
      }
      if (root.uuidJobGen !== root.sendGen)
        return
      if (exitCode !== 0) {
        root.sending = false
        root.error = "uuidgen failed"
        root.sendFinished()
        return
      }
      var id = String(uuidOut.text || "").trim()
      if (!id.length) {
        root.sending = false
        root.error = "uuidgen returned empty"
        root.sendFinished()
        return
      }
      if (!root.isUuid(id)) {
        abortSend("uuidgen returned invalid id")
        return
      }
      root.startGrokAfterPersist = true
      persistSessionId(id)
    }
  }

  Process {
    id: writeGrokErrProc
  }

  Process {
    id: grokProc
    stdout: SplitParser {
      onRead: function(line) {
        if (root.grokJobGen !== root.sendGen)
          return
        root.handleStreamLine(line)
      }
    }
    stderr: StdioCollector {
      id: grokErr
      waitForEnd: true
    }
    onExited: function(exitCode) {
      if (root.grokJobGen !== root.sendGen)
        return
      root.sending = false
      var errText = String(grokErr.text || "")
      if (errText.length) {
        writeGrokErrProc.command = [
          "sh", "-c",
          "umask 077 && printf '%s' \"$1\" > \"$2\" && chmod 0600 \"$2\"",
          "kage-ask-grok-err",
          errText,
          root.grokErrPath
        ]
        writeGrokErrProc.running = true
      }
      if (root.pendingFresh) {
        root.streamBuf = ""
      } else if (exitCode !== 0 && !root.streamBuf.length) {
        root.error = "grok failed (" + exitCode + ")"
      } else if (root.streamBuf.length) {
        setLastAssistant(root.streamBuf)
      }
      root.sendFinished()
      if (root.pendingFresh && !uuidProc.running)
        beginFreshUuid()
    }
  }

  FileView {
    id: configFile
    path: root.configFilePath
    printErrors: false
    watchChanges: true
    onLoaded: function() {
      root.configText = String(text() || "")
    }
    onFileChanged: reload()
    Component.onCompleted: reload()
  }
}
