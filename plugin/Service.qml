import QtQuick
import Quickshell
import Quickshell.Io

Item {
  id: root

  property var shell: null
  property var manifest: null
  property string omarchyPath: Quickshell.env("OMARCHY_PATH")

  readonly property string runtimeDir: {
    var dir = Quickshell.env("XDG_RUNTIME_DIR")
    if (dir && dir.length)
      return dir
    return "/tmp"
  }
  readonly property string askRoot: runtimeDir + "/kage/ask"
  property string issueId: "current"
  readonly property string issueDir: askRoot + "/" + issueId
  readonly property string rawPath: issueDir + "/raw.png"
  readonly property string annotatedPath: issueDir + "/annotated.png"
  readonly property string snapshotJsonPath: issueDir + "/snapshot.json"
  readonly property string wavPath: issueDir + "/rec.wav"

  property bool grabbing: false
  property bool recording: false
  property bool transcribing: false
  property bool hasMarks: false
  property string imagePath: ""
  property string imageToken: ""
  property var snapshot: null
  property string error: ""
  property string composerText: ""

  signal grabFinished()
  signal transcriptReady(string text)

  function grab(payloadJson) {
    root.grabbing = true
    root.error = ""
    root.hasMarks = false
    stopMic()
    if (ensureDirProc.running)
      ensureDirProc.running = false
    if (seeProc.running)
      seeProc.running = false
    ensureDirProc.command = ["install", "-d", "-m", "0700", root.askRoot, root.issueDir]
    ensureDirProc.running = true
  }

  function startRecording() {
    if (root.recording || recProc.running)
      return
    if (transcribeProc.running)
      transcribeProc.running = false
    root.transcribing = false
    recProc.command = ["pw-record", "--rate", "16000", "--channels", "1", "--format", "s16", root.wavPath]
    recProc.running = true
    root.recording = true
  }

  function cancelRecording() {
    if (recProc.running)
      recProc.running = false
    root.recording = false
    root.transcribing = false
    if (transcribeProc.running)
      transcribeProc.running = false
  }

  function stopMic() {
    cancelRecording()
  }

  function stopAndTranscribe() {
    if (!root.recording && !recProc.running)
      return
    root.recording = false
    if (recProc.running)
      recProc.running = false
    root.transcribing = true
    transcribeProc.command = ["voxtype", "transcribe", root.wavPath]
    transcribeProc.running = true
  }

  function markAnnotated() {
    root.hasMarks = true
    root.imagePath = root.annotatedPath
    root.imageToken = Date.now().toString()
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
      root.grabbing = false
      if (exitCode !== 0) {
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
        root.error = "kage see returned no path"
        root.grabFinished()
        return
      }
      root.snapshot = snap
      root.imagePath = snap.path
      root.imageToken = Date.now().toString()
      snapshotFile.setText(JSON.stringify(snap, null, 2) + "\n")
      root.grabFinished()
    }
  }

  Process {
    id: recProc
    onExited: function() {
      root.recording = false
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
      if (exitCode !== 0) {
        var msg = String(transcribeErr.text || transcribeOut.text || "").trim()
        root.error = msg.length ? msg : ("voxtype transcribe failed (" + exitCode + ")")
        return
      }
      var text = String(transcribeOut.text || "").trim()
      if (text.length) {
        if (root.composerText.length)
          root.composerText = root.composerText + (root.composerText.endsWith(" ") ? "" : " ") + text
        else
          root.composerText = text
        root.transcriptReady(text)
      }
    }
  }

  FileView {
    id: snapshotFile
    path: root.snapshotJsonPath
    printErrors: false
  }
}
