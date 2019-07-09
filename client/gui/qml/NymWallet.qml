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

import QtQuick 2.9
import QtQuick.Controls 2.5
import QtQuick.Layouts 1.13
import QtQuick.Dialogs 1.3
import Qt.labs.platform 1.1

Rectangle {
    id: walletPage
    anchors.fill: parent

   ColumnLayout {
        anchors.fill: parent
        anchors.margins: 10
        RowLayout {
			Layout.alignment: Qt.AlignTop
            Layout.preferredHeight: 40
            Layout.fillWidth: true
            TextField {
                id: path
                enabled: false
                text: "Please load Nym Client configuration file"
                Layout.fillWidth: true
            }
            Button {
                text: "open config"
                onClicked: fileDialog.open();
            }
        }
    }

    FileDialog {
        id: fileDialog
        folder: StandardPaths.standardLocations(StandardPaths.HomeLocation)[0]
        nameFilters: [ "Config files (*.toml)", "All files (*)" ]
        // onFolderChanged: {
        //     folderModel.folder = folder;
        // }
        onAccepted: {
			console.log("You chose: " + fileDialog.file)
			QmlBridge.loadConfig(fileDialog.file)
			path.text = fileDialog.file
        }
    }
  //   ProgressBar {
  //     id: loading
  //     anchors.horizontalCenter: parent.horizontalCenter
  //     visible: false
  //     indeterminate: true
  //   }
  // }



  MessageDialog {
    id: notificationDialog
    // modality: Qt.ApplicationModal	

    buttons: MessageDialog.Ok | MessageDialog.Cancel
    title: "Notification box"
  }

  Connections {
    target: QmlBridge
    onDisplayNotification: {
        notificationDialog.text = message
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