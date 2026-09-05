import Quickshell
import Quickshell.Wayland
import QtQuick
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

  property color background: Color.menu.background
  property color foreground: Color.menu.text
  property color border: Color.menu.border
  property var borderSpec: Border.surfaceSpec("menu", "border", border, Math.max(1, Style.space(2)))
  property color scrim: Color.menu.scrim
  readonly property int cornerRadius: Style.cornerRadius
  property string fontFamily: Style.font.menuFamily
  property int contentMargin: Style.spacing.panelPadding
  property int headerHeight: Math.max(Style.space(34), Style.font.title + Style.spacing.controlPaddingY * 2)
  property int contentSpacing: Style.spacing.md
  property int cardWidth: panel.width - Style.gapsOut * 2
  property int cardHeight: panel.height - Style.gapsOut * 2

  readonly property string imageUrl: {
    if (!root.service || !root.service.imagePath)
      return ""
    return Util.fileUrl(root.service.imagePath) + "?v=" + String(root.service.imageToken || "")
  }
  readonly property string statusText: {
    if (root.loadError.length)
      return root.loadError
    if (root.service && root.service.error)
      return root.service.error
    if (root.service && root.service.grabbing)
      return "Capturing…"
    return ""
  }

  function open(payloadJson) {
    // Hide first so kage see captures the screen, not this overlay.
    root.loadError = ""
    root.opened = false
    root.summoning = true
    if (!root.service || typeof root.service.grab !== "function") {
      root.loadError = "kage.ask service is not loaded"
      root.opened = true
      Qt.callLater(function() { keyCatcher.forceActiveFocus() })
      return
    }
    root.service.grab(payloadJson)
  }

  function close() {
    root.summoning = false
    root.opened = false
  }

  function dismiss() {
    root.close()
    if (root.shell && typeof root.shell.hide === "function")
      root.shell.hide((root.manifest && root.manifest.id) || "kage.ask")
  }

  Connections {
    target: root.service
    function onGrabFinished() {
      if (!root.summoning)
        return
      root.summoning = false
      root.opened = true
      Qt.callLater(function() { keyCatcher.forceActiveFocus() })
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
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            text: "Kage ask"
            color: root.foreground
            font.family: root.fontFamily
            font.pixelSize: Style.font.heading
            elide: Text.ElideRight
          }
        }

        Item {
          width: parent.width
          height: parent.height - root.headerHeight - root.contentSpacing

          Image {
            anchors.fill: parent
            visible: root.imageUrl.length > 0 && root.statusText.length === 0
            source: root.imageUrl
            fillMode: Image.PreserveAspectFit
            asynchronous: true
            cache: false
            smooth: true
          }

          Text {
            textFormat: Text.PlainText
            visible: root.statusText.length > 0
            anchors.centerIn: parent
            width: parent.width
            text: root.statusText
            color: root.foreground
            opacity: 0.7
            font.family: root.fontFamily
            font.pixelSize: Style.font.title
            wrapMode: Text.Wrap
            horizontalAlignment: Text.AlignHCenter
          }
        }
      }
    }
  }
}
