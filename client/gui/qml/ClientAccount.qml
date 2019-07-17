// ClientAccount.qml - blockchains interactions
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

ColumnLayout {
    id: mainColumn
    spacing: 20
    Layout.fillWidth: true

    ColumnLayout {
        id: columnLayout
        width: 100
        height: 100
        Layout.alignment: Qt.AlignHCenter | Qt.AlignVCenter
        Layout.fillHeight: false
        Layout.fillWidth: true

        RowLayout {
            id: rowLayout
            width: 100
            height: 100
            spacing: 5
            Layout.alignment: Qt.AlignHCenter | Qt.AlignVCenter
            Layout.fillWidth: true


            Label {
                text: "ERC20 Nym Balance:"
                horizontalAlignment: Text.AlignRight
                font.weight: Font.DemiBold
            }

            TextField {
                enabled: false
                id: erc20BalanceField
                Layout.maximumWidth: 100
                Layout.minimumWidth: 30
                Layout.preferredWidth: 50
                Layout.fillWidth: false
                placeholderText: "-1"
            }

            ToolSeparator {
                id: toolSeparator
                opacity: 0
            }

            Label {
                text: "ERC20 Nym Balance (pending):"
                horizontalAlignment: Text.AlignRight
                font.weight: Font.DemiBold
            }

            TextField {
                enabled: false
                id: erc20BalancePendingField
                Layout.maximumWidth: 100
                Layout.minimumWidth: 30
                Layout.preferredWidth: 50
                Layout.fillWidth: false
                placeholderText: "-1"
            }

            ToolSeparator {
                id: toolSeparator1
                opacity: 0
            }

            Label {
                text: "Nym Token Balance:"
                horizontalAlignment: Text.AlignRight
                font.weight: Font.DemiBold
            }

            TextField {
                enabled: false
                id: nymTokenBalanceField
                Layout.maximumWidth: 100
                Layout.minimumWidth: 30
                Layout.preferredWidth: 50
                Layout.fillWidth: false
                placeholderText: "-1"
            }



        }

        RowLayout {
            id: rowLayout4
            width: 100
            height: 100
            Layout.alignment: Qt.AlignHCenter | Qt.AlignVCenter
            Layout.fillWidth: true

            Button {
                id: updateBalancesBtn
                text: qsTr("Force update")
                Layout.alignment: Qt.AlignHCenter | Qt.AlignVCenter
                onClicked: {
                    // busyIndicator5.running = !busyIndicator5.running
                    QmlBridge.forceUpdateBalances(balanceUpdateIndicator, mainColumn)
                    // busyIndicator5.running = false
                }
            }

            BusyIndicator {
                id: balanceUpdateIndicator
                running: false
                width: 60
                Layout.preferredHeight: 50
                Layout.preferredWidth: 50
            }
        }
    }

    GridLayout {
        id: actionGrid
        width: 100
        height: 100
        columnSpacing: 10
        rowSpacing: 20
        rows: 4
        columns: 4
        Layout.fillHeight: true
        Layout.fillWidth: true


        Label {
            text: "Send to Pipe Account"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        TextField {
            // inputMethodHints: Qt.ImhDigitsOnly
            id: textField3
            text: qsTr(" ")
            placeholderText: "amount"
            Layout.fillWidth: false
        }


        Button {
            id: button
            text: "Confirm"
        }

        BusyIndicator {
            id: busyIndicator1
            running: false
            width: 60
            Layout.preferredHeight: 50
            Layout.preferredWidth: 50
        }


        Label {
            text: "Redeem Tokens"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        TextField {
            // inputMethodHints: Qt.ImhDigitsOnly
            id: textField4
            text: qsTr(" ")
            placeholderText: "amount"
            Layout.fillWidth: false
        }

        Button {
            id: button1
            text: "Confirm"
        }

        BusyIndicator {
            id: busyIndicator2
            running: false
            width: 60
            Layout.preferredHeight: 50
            Layout.preferredWidth: 50
        }

        Label {
            text: "Long term secret (TEMPORARY!)"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        TextField {
            enabled: false
            id: secretField
            Layout.columnSpan: 3
            placeholderText: "N/A"
            Layout.fillWidth: true
        }


        Label {
            text: "Get credential"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        ComboBox {
            id: comboBox
            currentIndex: 1
            displayText: "Value"
        }

        Button {
            id: button2
            text: "Confirm"
        }

        BusyIndicator {
            id: busyIndicator3
            running: false
            width: 60
            Layout.preferredHeight: 50
            Layout.preferredWidth: 50
        }


    }


    RowLayout {
        id: credentialDisplayRow
        width: 100
        height: 100
        Layout.fillHeight: false
        Layout.fillWidth: true

        GroupBox {
            id: groupBox1
            width: 200
            height: 200
            Layout.fillWidth: true
            Layout.minimumHeight: 200
            Layout.preferredHeight: 200
            Layout.maximumHeight: 300
            title: qsTr("Nym Credential List")
            Layout.preferredWidth: parent.width/2

            ScrollView {
                id: scrollView2
                x: -12
                y: -7
                anchors.topMargin: 5
                anchors.fill: parent
                anchors.bottomMargin: 5

                ListView {
                    id: listView
                    anchors.fill: parent
                    clip: true
                    keyNavigationWraps: true
                    model: ListModel {
                        ListElement {
                            name: "Grey"
                            colorCode: "grey"
                        }

                        ListElement {
                            name: "Red"
                            colorCode: "red"
                        }

                        ListElement {
                            name: "Blue"
                            colorCode: "blue"
                        }

                        ListElement {
                            name: "Green"
                            colorCode: "green"
                        }
                    }
                    delegate: Item {
                        x: 5
                        width: 80
                        height: 40
                        Row {
                            id: row1
                            spacing: 10
                            Rectangle {
                                width: 40
                                height: 40
                                color: colorCode
                            }

                            Text {
                                text: name
                                font.bold: true
                                anchors.verticalCenter: parent.verticalCenter
                            }
                        }
                    }
                }
            }
        }

    }

    RowLayout {
        id: selectedCredentialRow
        width: 100
        height: 100
        Layout.fillWidth: false

        Label {
            text: "Selected Credential:"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        ToolSeparator {
            id: toolSeparator2
            opacity: 0
        }

        Label {
            id: label
            text: qsTr("value:")
        }

        TextField {
            enabled: false
            id: textField6
            text: qsTr(" ")
            placeholderText: "-1"
            Layout.fillWidth: false
        }

        Label {
            id: label1
            text: qsTr("sequence:")
        }

        TextField {
            enabled: false
            id: textField8
            text: qsTr(" ")
            placeholderText: "N/A"
            Layout.fillWidth: false
        }


    }

    RowLayout {
        id: rowLayout3
        width: 100
        height: 100
        spacing: 15

        Label {
            text: "Spend the Credential"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        ComboBox {
            id: comboBox1
            Layout.preferredWidth: 250
            displayText: "Choose Service Provider"
            currentIndex: 1
        }

        ToolSeparator {
            id: toolSeparator3
            opacity: 0
        }

        Button {
            id: button3
            text: "Confirm"
        }

        BusyIndicator {
            id: busyIndicator4
            running: false
            width: 60
            Layout.preferredHeight: 50
            Layout.preferredWidth: 50
        }


    }
    Connections {
        target: QmlBridge
        onUpdateERC20NymBalance: {
            erc20BalanceField.text = amount
        }

        onUpdateERC20NymBalancePending: {
            erc20BalancePendingField.text = amount
        }

        onUpdateNymTokenBalance: {
            nymTokenBalanceField.text = amount
        }

        onUpdateSecret: {
            secretField.text = strigifiedSecret
        }

        
    }



}
























/*##^## Designer {
    D{i:0;height:1000;width:1000}
}
 ##^##*/
