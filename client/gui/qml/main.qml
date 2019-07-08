// main.qml - qml definition for the gui application
// Copyright (C) 2018-2019  Jedrzej Stuczynski.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

import QtQuick 2.7
import QtQuick.Controls 2.1

ApplicationWindow {
  id: window

  visible: true
  title: "Hello World Example"
  minimumWidth: 400
  minimumHeight: 400

  Button {
    anchors.centerIn: parent
    text: "Run whole pipeline"
    onClicked: {
      console.log("Qml says: we are going to run entire nym pipeline")
      QmlBridge.runPipeline()
    }
  }
}
//   Column {
//     anchors.centerIn: parent

//     TextField {
//       id: input

//       anchors.horizontalCenter: parent.horizontalCenter

//       placeholderText: "1. write something"
//     }

//     Button {
//       anchors.horizontalCenter: parent.horizontalCenter

//       text: "2. click me"
//       onClicked: {
//         console.log(QmlBridge.sendToGo(input.text))
        
//       }
      
//       enabled: true
//     }

//     Text {
//       id: someText
//       anchors.horizontalCenter: parent.horizontalCenter

//       text: "3. foo"
//     }
//   }

//   Connections {
//     target: QmlBridge
//     onSendToQml: {
//         someText.text = name
//       }
//   }

//   // Dialog {
//   //   contentWidth: window.width / 2
//   //   contentHeight: window.height / 4

//   //   id: dialog
//   //   standardButtons: Dialog.Ok

//   //   contentItem: Label {
//   //     text: input.text
//   //   }
//   // }
// }