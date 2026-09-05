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
  readonly property string snapshotJsonPath: issueDir + "/snapshot.json"

  property bool grabbing: false
  property string imagePath: ""
  property string imageToken: ""
  property var snapshot: null
  property string error: ""

  signal grabFinished()

  function grab(payloadJson) {
    root.grabbing = true
    root.error = ""
    if (ensureDirProc.running)
      ensureDirProc.running = false
    if (seeProc.running)
      seeProc.running = false
    ensureDirProc.command = ["install", "-d", "-m", "0700", root.askRoot, root.issueDir]
    ensureDirProc.running = true
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

  FileView {
    id: snapshotFile
    path: root.snapshotJsonPath
    printErrors: false
  }
}
