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
import QtQuick.Controls.Material 2.12
import CustomQmlTypes 1.0

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
                    QmlBridge.forceUpdateBalances(balanceUpdateIndicator, mainColumn)
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
            text: qsTr("Send to Pipe Account")
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        TextField {
            // inputMethodHints: Qt.ImhDigitsOnly
            id: sendToPipeAccountAmount
            placeholderText: "enter amount"
            Layout.fillWidth: false
        }


        Button {
            text: "Confirm"
            onClicked: {
                QmlBridge.sendToPipeAccount(sendToPipeAccountAmount.text, sendToPipeAccountIndicator, mainColumn)
            }
        }

        BusyIndicator {
            id: sendToPipeAccountIndicator
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
            id: redeemTokensAmount
            placeholderText: "enter amount"
            Layout.fillWidth: false
        }

        Button {
            text: "Confirm"
            onClicked: {
                QmlBridge.redeemTokens(redeemTokensAmount.text, redeemTokensIndicator, mainColumn)
            }
        }

        BusyIndicator {
            id: redeemTokensIndicator
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

        TooltipTextField {
            id: secretField
            textFieldText: ""
            textFieldPlaceholderText: "N/A"
            tooltipText: ""

            Layout.columnSpan: 3
            Layout.fillWidth: true
        }


        Label {
            text: "Get credential"
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        ComboBox {
            id: credentialValueBox
            currentIndex: -1
            displayText: "Value"

            onActivated: displayText = model[index]
        }

        Button {
            text: "Confirm"
            onClicked: {
                QmlBridge.getCredential(credentialValueBox.currentText, getCredentialIndicator, mainColumn)
            }
        }

        BusyIndicator {
            id: getCredentialIndicator
            running: false
            width: 60
            Layout.preferredHeight: 50
            Layout.preferredWidth: 50
        }


    }

    CredentialListModel {
        id: credentialListModel
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
                id: scrollView
                x: -12
                y: -7
                anchors.topMargin: 5
                anchors.fill: parent
                anchors.bottomMargin: 5

                Component {
                    id: highlight
                    Rectangle {
                        width: credentialList.width
                        height: 30
                        id: highlightRectangle
                        color: "lightsteelblue"; radius: 5
                        y: credentialList.currentItem.y - 5
                        Behavior on y {
                            SmoothedAnimation {
                                velocity: 300
                            }
                        }
                    }
                }

                ListView {
                    id: credentialList
                    anchors.fill: parent
                    clip: true
                    keyNavigationWraps: true
                    
                    model: credentialListModel
                    
                    highlight: highlight
                    highlightFollowsCurrentItem: false
                    focus: true

                    delegate: Item {
                        x: 5
                        width: 500
                        height: 30

                        property string displayCredential: Credential.substr(0,8) + " ... " + Credential.substr(-8)
                        property string displaySequence: Sequence.substr(0,8) + " ... " + Sequence.substr(-16)
                        property string credential: Credential
                        property string sequence: Sequence
                        property string value: Value

                        Row {
                            spacing: 5
                            Text {
                                text: "(" + Value + "Nym) "
                            }
                            Label {
                                text: displayCredential
                                font.weight: Font.DemiBold
                            }
                            Text {
                                text: "sequence: " + displaySequence
                            }
                            
                        }
                        MouseArea {
                            anchors.fill: parent
                            onClicked: {
                                credentialList.currentIndex = index
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
        spacing: 5

        Label {
            text: qsTr("Selected Credential:")
            horizontalAlignment: Text.AlignRight
            font.weight: Font.DemiBold
        }

        TooltipTextField {
            id: selectedCredentialField
            textFieldText: credentialList.currentItem != null ? credentialList.currentItem.displayCredential : ""
            textFieldPlaceholderText: "N/A"
            tooltipText: credentialList.currentItem != null ? credentialList.currentItem.credential : ""
        }

        ToolSeparator {
            opacity: 0
        }

        Label {
            text: qsTr("value:")
            font.weight: Font.DemiBold
        }

        TextField {
            enabled: false
            width: 30
            id: selectedCredentialValueField
            placeholderText: "N/A"
            text: credentialList.currentItem != null ? credentialList.currentItem.value + " Nym" : ""
        }

         ToolSeparator {
            opacity: 0
        }
        
        Label {
            id: label1
            text: qsTr("sequence:")
            font.weight: Font.DemiBold
        }

        TooltipTextField {
            id: selectedCredentialSequenceField
            textFieldText: credentialList.currentItem != null ? credentialList.currentItem.displaySequence : ""
            textFieldPlaceholderText: "N/A"
            tooltipText: credentialList.currentItem != null ? credentialList.currentItem.sequence : ""
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
            secretField.tooltipText = strigifiedSecret
            secretField.textFieldText = strigifiedSecret
        }

        onPopulateValueComboBox: {
            credentialValueBox.model = values
        }
        
        onAddCredentialListItem: {
            credentialListModel.addItem(item)
        }
    }
}
























/*##^## Designer {
    D{i:0;height:1000;width:1000}
}
 ##^##*/
