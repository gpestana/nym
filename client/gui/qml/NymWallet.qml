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

import QtQuick 2.12
import QtQuick.Controls 2.5
import QtQuick.Layouts 1.12
import Qt.labs.platform 1.1 as QtLabs
import CustomQmlTypes 1.0

Flickable {
    id: walletPage
    anchors.fill: parent

    ScrollIndicator.vertical: ScrollIndicator { }

    ColumnLayout {
        spacing: 5
        anchors.fill: parent
        anchors.margins: 30
        Label {
            id: label3
            text: qsTr("Nym Wallet Demo")
            horizontalAlignment: Text.AlignHCenter
            Layout.fillWidth: true
            Layout.alignment: Qt.AlignHCenter | Qt.AlignTop
            font.weight: Font.DemiBold
            font.pointSize: 16
        }

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

		ConfigSummary{
			id: configView
			visible: true
		}

		RowLayout {
			Layout.fillHeight: false
			Layout.alignment: Qt.AlignBottom | Qt.AlignRight

			Layout.preferredHeight: 40
			Layout.fillWidth: true

			Button {
				Layout.alignment: Qt.AlignRight | Qt.AlignBottom
				id: configConfirmBtn
				enabled: false
				text: "confirm"
				Layout.fillHeight: false
				Layout.fillWidth: false
				onClicked: console.log("Confirmed config")
			}
		}
    }

    QtLabs.FileDialog {
        id: fileDialog
        folder: QtLabs.StandardPaths.standardLocations(QtLabs.StandardPaths.HomeLocation)[0]
        nameFilters: [ "Config files (*.toml)", "All files (*)" ]
        // onFolderChanged: {
        //     folderModel.folder = folder;
        // }
        onAccepted: {
            console.log("You chose: " + fileDialog.file)
            QmlBridge.loadConfig(fileDialog.file)
			configConfirmBtn.enabled = true
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



    Dialog {
        id: notificationDialog
        parent: ApplicationWindow.contentItem
        anchors.centerIn: ApplicationWindow.contentItem

        height: 200
        width: Math.min(ApplicationWindow.contentItem.width * 2/3, 800)

        modal: true

        closePolicy: Popup.CloseOnPressOutside | Popup.CloseOnEscape
        standardButtons: Dialog.Ok
        title: qsTr("Notification box")

        Label {
            id: notificationText
            // maximumWidth: notificationDialog.contentWidth
        }

        onAccepted: console.log("Ok clicked")
        onRejected: console.log("Cancel clicked")
    }

    Connections {
        target: QmlBridge
        onDisplayNotification: {
            notificationText.text = message
            // notificationDialog.implicitHeight = notificationText.height
            // notificationDialog.implicitWidth = notificationText.width
            notificationDialog.open()
        }
		
		// onNewNymValidator: {
		// 	nymValidatorsListModel.append({Identifier: "foo", Address: "bar"})
		// }
    }

}

























/*##^## Designer {
    D{i:0;autoSize:true;height:768;width:1024}D{i:28;anchors_height:200;anchors_width:200}
}
 ##^##*/
