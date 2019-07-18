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
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/client"
	"github.com/nymtech/nym/client/config"
	"github.com/nymtech/nym/nym/token"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/qml"
	"github.com/therecipe/qt/quickcontrols2"
)

// TODO: once basic structure is figured out + got a good hang of how it works
// split the file into separate packages, etc.
// also cleanup it.... there's a lot of messy code here right now...

var (
	// qmlObjects = make(map[string]*core.QObject)

	qmlBridge    *QmlBridge
	configBridge *ConfigBridge
)

//go:generate qtmoc
type ConfigBridge struct {
	core.QObject

	_ string `property:"identifier"`
	_ string `property:"address"`
	_ string `property:"keyfile"`
	_ string `property:"ethereumNode"`
	_ string `property:"nymERC20"`
	_ string `property:"pipeAccount"`
}

//go:generate qtmoc
type QmlBridge struct {
	core.QObject
	cfg            *config.Config
	clientInstance *client.Client
	longtermSecret *Curve.BIG

	_ func() `constructor:"init"`

	_ func(file string)                                                 `slot:"loadConfig"`
	_ func()                                                            `slot:"confirmConfig"`
	_ func(message string)                                              `signal:"displayNotification"`
	_ func(identifier, address string)                                  `signal:"newNymValidator"`
	_ func(identifier, address string)                                  `signal:"newTendermintValidator"`
	_ func(amount string)                                               `signal:"updateERC20NymBalance"`
	_ func(amount string)                                               `signal:"updateERC20NymBalancePending"`
	_ func(amount string)                                               `signal:"updateNymTokenBalance"`
	_ func(strigifiedSecret string)                                     `signal:"updateSecret"`
	_ func(values []string)                                             `signal:"populateValueComboBox"`
	_ func(busyIndicator *core.QObject, mainLayoutObject *core.QObject) `slot:"forceUpdateBalances"`

	_ func(amount string, busyIndicator *core.QObject, mainLayoutObject *core.QObject) `slot:"sendToPipeAccount"`
	_ func(amount string, busyIndicator *core.QObject, mainLayoutObject *core.QObject) `slot:"redeemTokens"`
	_ func(value string, busyIndicator *core.QObject, mainLayoutObject *core.QObject)  `slot:"getCredential"`
	_ func()                                                                           `slot:"spendCredential"`
}

func setIndicatorAndObjects(indicator *core.QObject, objs []*core.QObject, run bool) {
	if run {
		if indicator != nil {
			indicator.SetProperty("running", core.NewQVariant1(true))
		}
		if len(objs) > 0 {
			disableAllObjects(objs)
		}
	} else {
		if indicator != nil {
			indicator.SetProperty("running", core.NewQVariant1(false))
		}
		if len(objs) > 0 {
			enableAllObjects(objs)
		}
	}
}

func enableAllObjects(objs []*core.QObject) {
	for _, obj := range objs {
		obj.SetProperty("enabled", core.NewQVariant1(true))
	}
}

func disableAllObjects(objs []*core.QObject) {
	for _, obj := range objs {
		obj.SetProperty("enabled", core.NewQVariant1(false))
	}
}

// TODO: FIXME: a temporary solution. copied from client file just to update 'live' erc20 balances
func (qb *QmlBridge) waitForERC20BalanceChange(ctx context.Context, expectedBalance uint64) {
	retryTicker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-retryTicker.C:
			currentBalance, err := qb.clientInstance.GetCurrentERC20Balance()
			qb.displayErrorDialogOnErr("failed to query for ERC20 Nym Balance", err)
			qb.UpdateERC20NymBalance(strconv.FormatUint(currentBalance, 10))

			pendingBalance, err := qb.clientInstance.GetCurrentERC20PendingBalance()
			qb.displayErrorDialogOnErr("failed to query for ERC20 Nym Balance (pending)", err)
			qb.UpdateERC20NymBalancePending(strconv.FormatUint(pendingBalance, 10))

			if currentBalance == expectedBalance {
				return
			}
		case <-ctx.Done():
			qb.displayErrorDialogOnErr("failed to query for obtain current ERC20 balances", errors.New("ctx timeout"))
			return
		}
	}
}

// this function will be automatically called, when you use the `NewQmlBridge` function
func (qb *QmlBridge) init() {
	qb.ConnectLoadConfig(func(file string) {
		// TODO: is that prefix always added?
		file = strings.TrimPrefix(file, "file://")

		cfg, err := config.LoadFile(file)
		if err != nil {
			errStr := fmt.Sprintf("Failed to load config file '%v': %v\n", file, err)
			qmlBridge.DisplayNotification(errStr)
			return
		} else {
			fmt.Println("loaded config!")
		}

		configBridge.SetIdentifier(cfg.Client.Identifier)
		configBridge.SetKeyfile(cfg.Nym.AccountKeysFile)

		// TODO: later remove it, but for now it's temporary for demo sake
		privateKey, loadErr := ethcrypto.LoadECDSA(cfg.Nym.AccountKeysFile)
		if loadErr != nil {
			errStr := fmt.Sprintf("Failed to load Nym keys: %v", loadErr)
			fmt.Println(errStr)
			configBridge.SetAddress(errStr)
		} else {
			address := ethcrypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey)).Hex()
			configBridge.SetAddress(address)
		}

		// should have been detected during validation...
		if len(cfg.Nym.EthereumNodeAddresses) > 0 {
			configBridge.SetEthereumNode(cfg.Nym.EthereumNodeAddresses[0])
		} else {
			configBridge.SetEthereumNode("none specified")
		}
		configBridge.SetNymERC20(cfg.Nym.NymContract.Hex())
		configBridge.SetPipeAccount(cfg.Nym.PipeAccount.Hex())

		for i, addr := range cfg.Client.IAAddresses {
			qb.NewNymValidator(fmt.Sprintf("nymnode%v", i), addr)
		}

		for i, addr := range cfg.Nym.BlockchainNodeAddresses {
			qb.NewTendermintValidator(fmt.Sprintf("tendermintnode%v", i), addr)
		}

		qb.cfg = cfg
	})

	qb.ConnectConfirmConfig(func() {
		if qb.clientInstance == nil {
			client, err := client.New(qb.cfg)
			if err != nil {
				errStr := fmt.Sprintf("Could not use the config to create client instance: %v\n", err)
				qmlBridge.DisplayNotification(errStr)
				return
			}
			qb.clientInstance = client
		}

		if qb.longtermSecret == nil {
			qb.longtermSecret = qb.clientInstance.RandomBIG()
			qb.UpdateSecret(qb.longtermSecret.ToString())
		}
		valueList := make([]string, len(token.AllowedValues))
		for i, val := range token.AllowedValues {
			valueList[i] = strconv.FormatInt(val, 10) + "Nym"
		}
		qb.PopulateValueComboBox(valueList)
	})

	qb.ConnectForceUpdateBalances(func(busyIndicator *core.QObject, mainLayoutObject *core.QObject) {
		go func() {
			setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, true)
			defer setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, false)

			qb.updateBalances()
		}()
	})

	qb.ConnectSendToPipeAccount(func(amount string, busyIndicator *core.QObject, mainLayoutObject *core.QObject) {
		go func() {
			setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, true)
			defer setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, false)

			amountInt64, err := strconv.ParseInt(amount, 10, 64)
			qb.displayErrorDialogOnErr("could not parse value", err)

			currentERC20Balance, err := qb.clientInstance.GetCurrentERC20Balance()
			qb.displayErrorDialogOnErr("failed to query for ERC20 Nym Balance", err)
			currentNymBalance, err := qb.clientInstance.GetCurrentNymBalance()
			qb.displayErrorDialogOnErr("failed to query for Nym Token Balance", err)

			// TODO:
			ctx := context.TODO()
			err = qb.clientInstance.SendToPipeAccount(ctx, amountInt64)
			qb.displayErrorDialogOnErr(fmt.Sprintf("failed to send %v to the pipe account", amountInt64), err)
			if err != nil {
				return
			}

			qb.waitForERC20BalanceChange(ctx, currentERC20Balance-uint64(amountInt64))

			err = qb.clientInstance.WaitForBalanceChange(ctx, currentNymBalance+uint64(amountInt64))
			qb.displayErrorDialogOnErr("could not query for the Nym token balance", err)

			qb.UpdateNymTokenBalance(strconv.FormatUint(currentNymBalance+uint64(amountInt64), 10))
		}()
	})

	qb.ConnectRedeemTokens(func(amount string, busyIndicator *core.QObject, mainLayoutObject *core.QObject) {
		go func() {
			setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, true)
			defer setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, false)

			amountInt64, err := strconv.ParseInt(amount, 10, 64)
			qb.displayErrorDialogOnErr("could not parse value", err)

			currentERC20Balance, err := qb.clientInstance.GetCurrentERC20Balance()
			qb.displayErrorDialogOnErr("failed to query for ERC20 Nym Balance", err)
			currentNymBalance, err := qb.clientInstance.GetCurrentNymBalance()
			qb.displayErrorDialogOnErr("failed to query for Nym Token Balance", err)

			// TODO:
			ctx := context.TODO()

			err = qb.clientInstance.RedeemTokens(ctx, uint64(amountInt64))
			qb.displayErrorDialogOnErr(fmt.Sprintf("failed to redeem %v tokens", amountInt64), err)
			if err != nil {
				return
			}

			err = qb.clientInstance.WaitForBalanceChange(ctx, currentNymBalance-uint64(amountInt64))
			qb.displayErrorDialogOnErr("could not query for the Nym token balance", err)
			if err != nil {
				return
			}

			qb.UpdateNymTokenBalance(strconv.FormatUint(currentNymBalance-uint64(amountInt64), 10))
			qb.waitForERC20BalanceChange(ctx, currentERC20Balance+uint64(amountInt64))
		}()
	})

	qb.ConnectGetCredential(func(value string, busyIndicator *core.QObject, mainLayoutObject *core.QObject) {
		go func() {
			setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, true)
			defer setIndicatorAndObjects(busyIndicator, []*core.QObject{mainLayoutObject}, false)

			value = strings.TrimSuffix(value, "Nym")
			valueInt64, err := strconv.ParseInt(value, 10, 64)
			qb.displayErrorDialogOnErr("could not parse value", err)

			seq := qb.clientInstance.RandomBIG()

			token, err := token.New(seq, qb.longtermSecret, valueInt64)
			qb.displayErrorDialogOnErr(fmt.Sprintf("could not generate token for %v", value), err)

			cred, err := qb.clientInstance.GetCredential(token)
			qb.displayErrorDialogOnErr(fmt.Sprintf("could not obtain credential for %v", value), err)

			qb.updateBalances()

			if cred != nil && err == nil {
				qb.DisplayNotification(fmt.Sprintf("Obtained credential: \nsig1: %v\nsig2:%v\n", cred.Sig1().ToString(), cred.Sig2().ToString()))
				// TODO: add it to the list model that I will create
			}
			fmt.Printf("obtained cred: %+v\n", cred)
		}()
	})
}

func (qb *QmlBridge) updateBalances() {
	erc20balance, err := qb.clientInstance.GetCurrentERC20Balance()
	qb.displayErrorDialogOnErr("failed to query for ERC20 Nym Balance", err)
	pending, err := qb.clientInstance.GetCurrentERC20PendingBalance()
	qb.displayErrorDialogOnErr("failed to query for ERC20 Nym Balance (pending)", err)
	nymBalance, err := qb.clientInstance.GetCurrentNymBalance()
	qb.displayErrorDialogOnErr("failed to query for Nym Token Balance", err)

	qb.UpdateERC20NymBalance(strconv.FormatUint(erc20balance, 10))
	qb.UpdateERC20NymBalancePending(strconv.FormatUint(pending, 10))
	qb.UpdateNymTokenBalance(strconv.FormatUint(nymBalance, 10))
}

// TODO: redesign this....
func (qb *QmlBridge) displayErrorDialogOnErr(prefix string, err error) {
	if err != nil {
		qb.DisplayNotification(fmt.Sprintf("%v: %v", prefix, err))
	}
}

func main() {

	// enable high dpi scaling
	// useful for devices with high pixel density displays
	// such as smartphones, retina displays, ...
	core.QCoreApplication_SetAttribute(core.Qt__AA_EnableHighDpiScaling, true)

	// needs to be called once before you can start using QML
	gui.NewQGuiApplication(len(os.Args), os.Args)
	// widgets.NewQApplication(len(os.Args), os.Args)
	gui.QGuiApplication_SetApplicationDisplayName("Nym Demo")

	// use the material style
	// the other inbuild styles are:
	// Default, Fusion, Imagine, Universal
	quickcontrols2.QQuickStyle_SetStyle("Material")
	// quickcontrols2.QQuickStyle_SetStyle("Imagine")

	fntdb := gui.NewQFontDatabase()
	fntdb.AddApplicationFont(":/materialdesignicons-webfont.ttf")

	// create the qml application engine
	engine := qml.NewQQmlApplicationEngine(nil)

	// Create connector
	qmlBridge = NewQmlBridge(nil)
	configBridge = NewConfigBridge(nil)

	// Set up the connector
	engine.RootContext().SetContextProperty("QmlBridge", qmlBridge)

	engine.RootContext().SetContextProperty("ConfigBridge", configBridge)

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
