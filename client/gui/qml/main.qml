import QtQuick 2.7
import QtQuick.Controls 2.1

ApplicationWindow {
  id: window

  visible: true
  title: "Hello World Example"
  minimumWidth: 400
  minimumHeight: 400

  Column {
    anchors.centerIn: parent

    TextField {
      id: input

      anchors.horizontalCenter: parent.horizontalCenter

      placeholderText: "1. write something"
    }

    Button {
      anchors.horizontalCenter: parent.horizontalCenter

      text: "2. click me"
      onClicked: {
        QmlBridge.sendToGo(input.text)
      }
      
      enabled: true
    }

    Text {
      id: someText
      anchors.horizontalCenter: parent.horizontalCenter

      text: "3. foo"
    }
  }

  Connections {
    target: QmlBridge
    onSendToQml: {
        someText.text = name
      }
  }

  // Dialog {
  //   contentWidth: window.width / 2
  //   contentHeight: window.height / 4

  //   id: dialog
  //   standardButtons: Dialog.Ok

  //   contentItem: Label {
  //     text: input.text
  //   }
  // }
}