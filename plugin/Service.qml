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
  readonly property string rawPath: issueDir + "/raw.png"
  readonly property string annotatedPath: issueDir + "/annotated.png"
  readonly property string annotatedTmpPath: issueDir + "/annotated.png.tmp"
  readonly property string snapshotJsonPath: issueDir + "/snapshot.json"
  readonly property string promptTxtPath: issueDir + "/prompt.txt"
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
  property int unlinkJobGen: 0
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

  signal grabFinished()
  signal transcriptReady(string text)
  signal chatUpdated()
  signal sendFinished()

  function grab(payloadJson) {
    root.grabbing = true
    root.error = ""
    root.hasMarks = false
    root.pendingStrokes = null
    root.burnGen += 1
    root.grabGen += 1
    stopMic()
    if (burnProc.running)
      burnProc.running = false
    if (mvBurnProc.running)
      mvBurnProc.running = false
    if (ensureDirProc.running)
      ensureDirProc.running = false
    if (seeProc.running)
      seeProc.running = false
    ensureDirProc.command = ["install", "-d", "-m", "0700", root.kageDir, root.askRoot, root.issueDir]
    ensureDirProc.running = true
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
    root.unlinkJobGen = root.grabGen
    if (rmAnnotatedProc.running)
      return
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
    ensureConfigProc.command = ["install", "-d", "-m", "0700", root.configDir]
    ensureConfigProc.running = true
  }

  function abortSend(msg) {
    root.sendQueued = false
    root.startGrokAfterPersist = false
    root.sending = false
    root.error = msg
    root.sendFinished()
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
    uuidProc.command = ["uuidgen"]
    uuidProc.running = true
  }

  function maybeLaunchQueuedSend() {
    if (!root.sendQueued)
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
    var cmd = [
      "grok",
      "--prompt-json", root.pendingPromptJson,
      "--output-format", "streaming-json",
      "--cwd", root.homeDir,
      "--permission-mode", "dontAsk",
      "--deny", "kage click",
      "--deny", "kage type",
      "--deny", "kage press",
      "--deny", "kage hotkey",
      "--deny", "mcp__kage__kage_click",
      "--deny", "mcp__kage__kage_type",
      "--deny", "mcp__kage__kage_press",
      "--deny", "mcp__kage__kage_hotkey",
      "--deny", "Bash(kage click*)",
      "--deny", "Bash(kage type*)",
      "--deny", "Bash(kage press*)",
      "--deny", "Bash(kage hotkey*)",
      "--disallowed-tools", "kage__kage_click,kage__kage_type,kage__kage_press,kage__kage_hotkey"
    ]
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
    if (obj.type === "text" && obj.data != null)
      root.streamBuf += String(obj.data)
    else if (obj.type === "assistant" && obj.text)
      root.streamBuf += String(obj.text)
    else if (obj.delta && obj.delta.text)
      root.streamBuf += String(obj.delta.text)
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

  Process {
    id: ensureDirProc
    onExited: function(exitCode) {
      if (exitCode !== 0) {
        root.grabbing = false
        root.error = "could not create " + root.askRoot
        root.grabFinished()
        return
      }
      seeProc.command = ["kage", "see", "--path", root.rawPath]
      seeProc.running = true
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
      if (exitCode !== 0) {
        root.grabbing = false
        var msg = String(seeErr.text || seeOut.text || "").trim()
        root.error = msg.length ? msg : ("kage see failed (" + exitCode + ")")
        root.grabFinished()
        return
      }
      var snap = null
      try {
        snap = JSON.parse(String(seeOut.text || "").trim())
      } catch (e) {
        snap = null
      }
      if (!snap || !snap.path) {
        root.grabbing = false
        root.error = "kage see returned no path"
        root.grabFinished()
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
      root.grabbing = false
      root.grabFinished()
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

  FileView {
    id: grokErrFile
    path: root.grokErrPath
    printErrors: false
  }

  Process {
    id: ensureConfigProc
    onExited: function(exitCode) {
      if (exitCode !== 0) {
        if (root.startGrokAfterPersist)
          abortSend("could not create " + root.configDir)
        else
          root.error = "could not create " + root.configDir
        return
      }
      if (!root.isUuid(root.sessionId))
        return
      sessionFile.setText(root.sessionId + "\n")
      chmodSessionProc.command = ["chmod", "0600", root.sessionFilePath]
      chmodSessionProc.running = true
    }
  }

  Process {
    id: chmodSessionProc
    onExited: function(exitCode) {
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
    id: chmodGrokErrProc
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
        grokErrFile.setText(errText)
        chmodGrokErrProc.command = ["chmod", "0600", root.grokErrPath]
        chmodGrokErrProc.running = true
      }
      if (exitCode !== 0 && !root.streamBuf.length)
        root.error = "grok failed (" + exitCode + ")"
      else if (root.streamBuf.length)
        setLastAssistant(root.streamBuf)
      root.sendFinished()
    }
  }
}
