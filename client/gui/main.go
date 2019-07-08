package main

import (
	"fmt"
	"os"

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

func doStuff(foo string) {
	bar := foo + "bar"
	fmt.Println(bar)
}

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
	_ func(name string) `slot:"sendToGo"`
}

//this function will be automatically called, when you use the `NewExampleStruct` function
func (qb *QmlBridge) init() {
	//here you can do some initializing
	fmt.Println("init called on qmlbridge")
	qb.ConnectSendToGo(func(name string) {
		fmt.Println("sent to go", name)
		qb.SendToQml(name + "foo")
		// return "hello from go"
	})
	// qb.ConnectSendToQml(func(name string) {
	// 	fmt.Println("connect to qml?")
	// })
	// s.SetFirstProperty("defaultString")
	// s.ConnectFirstSignal(func() { println("do something here") })
	// s.ConnectSecondSignal(s.secondSignal)
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

	// create the qml application engine
	engine := qml.NewQQmlApplicationEngine(nil)

	// Create connector
	var qmlBridge = NewQmlBridge(nil)

	// // Function to execute from QML
	// qmlBridge.ConnectRemoveItem(func(data int) string {
	// 	fmt.Println("go:", data)
	// 	return "hello from go"
	// })

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
