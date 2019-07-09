// main.go - entry point for nym GUI application
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
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/qml"
	"github.com/therecipe/qt/quickcontrols2"
	"github.com/therecipe/qt/widgets"
)

var (
	qmlObjects = make(map[string]*core.QObject)

	qmlBridge          *QmlBridge
	manipulatedFromQml *widgets.QWidget
)

// type QmlBridge struct {
// 	core.QObject

// 	_ func() `constructor:"init"`
// 	_ func(source, action, data string) `signal:"sendToQml"`
// 	_ func(source, action, data string) `slot:"sendToGo"`

// 	_ func(object *core.QObject) `slot:"registerToGo"`
// 	_ func(objectName string)    `slot:"deregisterToGo"`
// }

//go:generate qtmoc
type QmlBridge struct {
	core.QObject

	_ func() `constructor:"init"`

	// Signal to make QML do something on 'sendToQml'
	_ func(name string) `signal:"sendToQml"`

	// Slot to make Go do something on 'sendToGo'
	_ func(name string) string `slot:"sendToGo"`

	_ func(file string) `slot:"loadConfig"`

	_ func(message string) `signal:"displayNotification"`

	_ func() `slot:"runPipeline"`
}

//this function will be automatically called, when you use the `NewQmlBridge` function
func (qb *QmlBridge) init() {
	//here you can do some initializing
	fmt.Println("init called on qmlbridge")
	qb.ConnectSendToGo(func(name string) string {
		fmt.Println("sent to go", name)
		qb.SendToQml(name + "foo")
		return "hello from go"
	})

	qb.ConnectRunPipeline(func() {
		fmt.Println("Called to run entire pipeline")
		qb.DisplayNotification("Sample notification text")
		fmt.Println("after notif")
		// runWhole()
	})

	qb.ConnectLoadConfig(func(file string) {
		// TODO: is that prefix always added?
		file = strings.TrimPrefix(file, "file://")

		fmt.Println("Want to load config", file)
		qb.DisplayNotification("File to load: " + file)
	})
}

func main() {

	// enable high dpi scaling
	// useful for devices with high pixel density displays
	// such as smartphones, retina displays, ...
	core.QCoreApplication_SetAttribute(core.Qt__AA_EnableHighDpiScaling, true)

	// needs to be called once before you can start using QML
	gui.NewQGuiApplication(len(os.Args), os.Args)

	// use the material style
	// the other inbuild styles are:
	// Default, Fusion, Imagine, Universal
	quickcontrols2.QQuickStyle_SetStyle("Material")

	fntdb := gui.NewQFontDatabase()
	fntdb.AddApplicationFont(":/materialdesignicons-webfont.ttf")

	// create the qml application engine
	engine := qml.NewQQmlApplicationEngine(nil)

	// Create connector
	var qmlBridge = NewQmlBridge(nil)

	// Set up the connector
	engine.RootContext().SetContextProperty("QmlBridge", qmlBridge)

	// load the embedded qml file
	// created by either qtrcc or qtdeploy
	engine.Load(core.NewQUrl3("qrc:/qml/main.qml", 0))
	// you can also load a local file like this instead:
	// engine.Load(core.QUrl_FromLocalFile("./qml/main.qml"))

	// start the main Qt event loop
	// and block until app.Exit() is called
	// or the window is closed by the user
	gui.QGuiApplication_Exec()
}
