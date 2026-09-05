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
  readonly property string snapshotJsonPath: issueDir + "/snapshot.json"
  readonly property string wavPath: issueDir + "/rec.wav"

  property bool grabbing: false
  property bool recording: false
  property bool transcribing: false
  property bool transcribeWhenRecEnds: false
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
    ensureDirProc.command = ["install", "-d", "-m", "0700", root.kageDir, root.askRoot, root.issueDir]
    ensureDirProc.running = true
  }

  function startRecording() {
    if (root.recording || recProc.running)
      return
    if (transcribeProc.running)
      transcribeProc.running = false
    root.transcribing = false
    root.transcribeWhenRecEnds = false
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
    transcribeProc.command = ["voxtype", "transcribe", root.wavPath]
    transcribeProc.running = true
  }

  function markAnnotated() {
    root.hasMarks = true
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
      unlinkWav()
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

  Process {
    id: rmWavProc
  }

  FileView {
    id: snapshotFile
    path: root.snapshotJsonPath
    printErrors: false
  }
}
