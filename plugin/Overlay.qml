import Quickshell
import Quickshell.Wayland
import QtQuick
import QtQuick.Controls
import qs.Commons
import qs.Ui

Item {
  id: root

  property string omarchyPath: Quickshell.env("OMARCHY_PATH")
  property var shell: null
  property var manifest: null
  property var service: null

  property bool opened: false
  property bool summoning: false
  property string loadError: ""
  property var strokes: []
  property var currentStroke: []
  property bool pendingBurn: false

  property color background: Color.menu.background
  property color foreground: Color.menu.text
  property color border: Color.menu.border
  property var borderSpec: Border.surfaceSpec("menu", "border", border, Math.max(1, Style.space(2)))
  property color scrim: Color.menu.scrim
  property color accent: Color.accent
  readonly property int cornerRadius: Style.cornerRadius
  property string fontFamily: Style.font.menuFamily
  property int contentMargin: Style.spacing.panelPadding
  property int headerHeight: Math.max(Style.space(34), Style.font.title + Style.spacing.controlPaddingY * 2)
  property int composerHeight: Math.max(Style.space(88), Style.font.body * 4)
  property int contentSpacing: Style.spacing.md
  property int cardWidth: panel.width - Style.gapsOut * 2
  property int cardHeight: panel.height - Style.gapsOut * 2

  readonly property string imageUrl: {
    if (!root.service || !root.service.rawPath || !root.service.imageToken)
      return ""
    if (!root.service.imagePath)
      return ""
    return Util.fileUrl(root.service.rawPath) + "?v=" + String(root.service.imageToken || "")
  }
  readonly property bool recording: root.service && root.service.recording
  readonly property bool inkLocked: {
    if (!root.service)
      return true
    return root.service.grabbing || root.service.transcribing
  }
  readonly property string statusText: {
    if (root.loadError.length)
      return root.loadError
    if (root.service && root.service.error)
      return root.service.error
    if (root.service && root.service.grabbing)
      return "Capturing…"
    if (root.service && root.service.transcribing)
      return "Transcribing…"
    return ""
  }

  function open(payloadJson) {
    // Hide first so kage see captures the screen, not this overlay.
    root.loadError = ""
    root.opened = false
    root.summoning = true
    root.strokes = []
    root.currentStroke = []
    if (!root.service || typeof root.service.grab !== "function") {
      root.loadError = "kage.ask service is not loaded"
      root.opened = true
      Qt.callLater(function() { keyCatcher.forceActiveFocus() })
      return
    }
    root.service.grab(payloadJson)
  }

  function close() {
    if (root.service && typeof root.service.stopMic === "function")
      root.service.stopMic()
    root.summoning = false
    root.opened = false
  }

  function dismiss() {
    if (root.service && typeof root.service.cancelRecording === "function")
      root.service.cancelRecording()
    root.close()
    if (root.shell && typeof root.shell.hide === "function")
      root.shell.hide((root.manifest && root.manifest.id) || "kage.ask")
  }

  function onRecClicked() {
    if (!root.service)
      return
    if (root.service.recording) {
      root.service.cancelRecording()
      return
    }
    root.service.startRecording()
  }

  function onStopClicked() {
    if (!root.service)
      return
    root.service.stopAndTranscribe()
  }

  function fittedRect(availW, availH, srcW, srcH) {
    if (srcW <= 0 || srcH <= 0)
      return { x: 0, y: 0, w: availW, h: availH }
    var s = Math.min(availW / srcW, availH / srcH)
    var w = Math.max(1, Math.floor(srcW * s))
    var h = Math.max(1, Math.floor(srcH * s))
    return { x: Math.floor((availW - w) / 2), y: Math.floor((availH - h) / 2), w: w, h: h }
  }

  function paintStrokes(ctx, w, h) {
    ctx.strokeStyle = "#e23d28"
    ctx.lineWidth = Math.max(3, w / 400)
    ctx.lineCap = "round"
    ctx.lineJoin = "round"
    var all = root.strokes.slice()
    if (root.currentStroke && root.currentStroke.length)
      all.push(root.currentStroke)
    for (var s = 0; s < all.length; s++) {
      var pts = all[s]
      if (!pts || pts.length < 2)
        continue
      ctx.beginPath()
      ctx.moveTo(pts[0].x * w, pts[0].y * h)
      for (var i = 1; i < pts.length; i++)
        ctx.lineTo(pts[i].x * w, pts[i].y * h)
      ctx.stroke()
    }
  }

  function burnMarks() {
    if (!root.service || !root.service.snapshot)
      return
    var snap = root.service.snapshot
    var w = snap.width || 0
    var h = snap.height || 0
    if (w <= 0 || h <= 0)
      return
    if (rawPreview.status !== Image.Ready)
      return
    root.pendingBurn = true
    burnCanvas.width = w
    burnCanvas.height = h
    burnCanvas.requestPaint()
  }

  Connections {
    target: root.service
    function onGrabFinished() {
      if (!root.summoning)
        return
      root.summoning = false
      root.opened = true
      Qt.callLater(function() {
        keyCatcher.forceActiveFocus()
        if (root.service && !root.service.error && typeof root.service.startRecording === "function")
          root.service.startRecording()
      })
    }
    function onTranscriptReady(text) {
      composer.text = root.service.composerText
    }
  }

  PanelWindow {
    id: panel
    visible: root.opened
    anchors { top: true; bottom: true; left: true; right: true }
    color: "transparent"
    WlrLayershell.namespace: "kage-ask"
    WlrLayershell.layer: WlrLayer.Overlay
    WlrLayershell.keyboardFocus: WlrKeyboardFocus.Exclusive
    exclusionMode: ExclusionMode.Ignore

    Rectangle {
      anchors.fill: parent
      color: root.scrim
    }

    MouseArea {
      anchors.fill: parent
      onClicked: root.dismiss()
    }

    BorderSurface {
      id: card
      width: root.cardWidth
      height: root.cardHeight
      radius: root.cornerRadius
      anchors.centerIn: parent
      color: root.background
      borderSpec: root.borderSpec
      padding: root.contentMargin

      MouseArea { anchors.fill: parent; onClicked: {} }

      Item {
        id: keyCatcher
        anchors.fill: parent
        focus: true

        Keys.priority: Keys.BeforeItem
        Keys.onPressed: function(event) {
          if (event.key === Qt.Key_Escape) {
            root.dismiss()
            event.accepted = true
          }
        }
      }

      Column {
        anchors.fill: parent
        anchors.topMargin: card.contentTopInset
        anchors.rightMargin: card.contentRightInset
        anchors.bottomMargin: card.contentBottomInset
        anchors.leftMargin: card.contentLeftInset
        spacing: root.contentSpacing

        Rectangle {
          width: parent.width
          height: root.headerHeight
          radius: root.cornerRadius
          color: "transparent"

          Text {
            textFormat: Text.PlainText
            anchors.left: parent.left
            anchors.right: recRow.left
            anchors.rightMargin: root.contentSpacing
            anchors.verticalCenter: parent.verticalCenter
            text: "Kage ask"
            color: root.foreground
            font.family: root.fontFamily
            font.pixelSize: Style.font.heading
            elide: Text.ElideRight
          }

          Row {
            id: recRow
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            spacing: root.contentSpacing

            Rectangle {
              id: recLight
              width: Style.space(12)
              height: Style.space(12)
              radius: width / 2
              anchors.verticalCenter: parent.verticalCenter
              color: root.recording ? "#e23d28" : Qt.rgba(root.foreground.r, root.foreground.g, root.foreground.b, 0.25)
              visible: true
            }

            Button {
              text: root.recording ? "Rec again" : "Rec"
              foreground: root.foreground
              accent: root.accent
              bordered: true
              onClicked: root.onRecClicked()
            }

            Button {
              text: "Stop"
              visible: root.recording
              foreground: root.foreground
              accent: root.accent
              bordered: true
              onClicked: root.onStopClicked()
            }
          }
        }

        Item {
          id: stageHost
          width: parent.width
          height: parent.height - root.headerHeight - root.composerHeight - root.contentSpacing * 2

          readonly property var srcSize: {
            var snap = root.service && root.service.snapshot
            if (snap && snap.width && snap.height)
              return { w: snap.width, h: snap.height }
            return { w: 16, h: 9 }
          }
          readonly property var fit: root.fittedRect(width, height, srcSize.w, srcSize.h)

          Item {
            id: viewLayer
            x: stageHost.fit.x
            y: stageHost.fit.y
            width: stageHost.fit.w
            height: stageHost.fit.h

            Image {
              id: rawPreview
              anchors.fill: parent
              visible: root.imageUrl.length > 0
              source: root.imageUrl
              fillMode: Image.Stretch
              asynchronous: true
              cache: false
              smooth: true
            }

            Canvas {
              id: ink
              anchors.fill: parent
              renderTarget: Canvas.Image
              renderStrategy: Canvas.Immediate
              onPaint: {
                var ctx = getContext("2d")
                ctx.clearRect(0, 0, width, height)
                root.paintStrokes(ctx, width, height)
              }
            }

            MouseArea {
              anchors.fill: parent
              acceptedButtons: Qt.LeftButton
              preventStealing: true
              enabled: !root.inkLocked && root.imageUrl.length > 0
              onPressed: function(mouse) {
                root.currentStroke = [{ x: mouse.x / Math.max(1, width), y: mouse.y / Math.max(1, height) }]
                ink.requestPaint()
              }
              onPositionChanged: function(mouse) {
                if (!pressed)
                  return
                var pts = root.currentStroke.slice()
                pts.push({ x: mouse.x / Math.max(1, width), y: mouse.y / Math.max(1, height) })
                root.currentStroke = pts
                ink.requestPaint()
              }
              onReleased: function() {
                if (root.currentStroke && root.currentStroke.length > 1) {
                  var next = root.strokes.slice()
                  next.push(root.currentStroke)
                  root.strokes = next
                }
                root.currentStroke = []
                ink.requestPaint()
                root.burnMarks()
              }
            }
          }

          Rectangle {
            visible: root.statusText.length > 0
            anchors.left: parent.left
            anchors.right: parent.right
            anchors.bottom: parent.bottom
            height: Math.max(Style.space(28), Style.font.body * 2)
            color: Qt.rgba(root.background.r, root.background.g, root.background.b, 0.85)

            Text {
              textFormat: Text.PlainText
              anchors.fill: parent
              anchors.margins: Style.spacing.controlPaddingX
              text: root.statusText
              color: root.foreground
              font.family: root.fontFamily
              font.pixelSize: Style.font.body
              wrapMode: Text.Wrap
              elide: Text.ElideRight
              verticalAlignment: Text.AlignVCenter
              horizontalAlignment: Text.AlignHCenter
            }
          }
        }

        Rectangle {
          width: parent.width
          height: root.composerHeight
          radius: root.cornerRadius
          color: Style.controlFill(composer.activeFocus, false, root.foreground, root.accent)
          border.color: root.border
          border.width: Math.max(1, Style.space(1))

          TextArea {
            id: composer
            objectName: "composer"
            anchors.fill: parent
            anchors.margins: Style.spacing.controlPaddingX
            wrapMode: TextEdit.Wrap
            textFormat: TextEdit.PlainText
            font.family: root.fontFamily
            font.pixelSize: Style.font.body
            color: root.foreground
            placeholderText: "Talk or type. Nothing is sent yet."
            onTextChanged: {
              if (root.service)
                root.service.composerText = text
            }
            background: Item {}
            Keys.onPressed: function(event) {
              if (event.key === Qt.Key_Escape) {
                root.dismiss()
                event.accepted = true
              }
            }
          }
        }
      }
    }
  }

  Canvas {
    id: burnCanvas
    visible: false
    width: 1
    height: 1
    renderTarget: Canvas.Image
    renderStrategy: Canvas.Immediate
    onPaint: {
      if (!root.pendingBurn)
        return
      var ctx = getContext("2d")
      ctx.clearRect(0, 0, width, height)
      ctx.drawImage(rawPreview, 0, 0, width, height)
      root.paintStrokes(ctx, width, height)
      var path = root.service ? root.service.annotatedPath : ""
      root.pendingBurn = false
      if (path.length && burnCanvas.save(path) && root.service)
        root.service.markAnnotated()
    }
  }
}
