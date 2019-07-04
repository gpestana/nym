// main_sample.go - sample usage for coconut/tendermint client
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
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	cclient "github.com/nymtech/nym/client"
	"github.com/nymtech/nym/client/config"
	"github.com/nymtech/nym/crypto/bpgroup"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/logger"
	"github.com/nymtech/nym/nym/token"
	tmclient "github.com/nymtech/nym/tendermint/client"
	"github.com/nymtech/nym/tendermint/nymabci/code"
	"github.com/nymtech/nym/tendermint/nymabci/query"
	"github.com/nymtech/nym/tendermint/nymabci/transaction"
	"gopkg.in/op/go-logging.v1"
)

const provider1IP = "127.0.0.1:4100"
const provider1Address = "0x5F828924E58f98f3dA07596F392fCB094aC818ad"
const provider2IP = "127.0.0.1:4101"
const provider2Address = "0xEe45d746721633f37142EDa6bd99F115aEb2Ff2D"

//nolint: gochecknoglobals
var (
	tendermintABCIAddresses = []string{
		// "tcp://0.0.0.0:12345", // does not exist
		"tcp://0.0.0.0:26657",
		"tcp://0.0.0.0:26659",
		"tcp://0.0.0.0:26661",
		"tcp://0.0.0.0:26663",
	}
)

// const tendermintABCIAddress = "tcp://0.0.0.0:26657"

func getRandomAttributes(G *bpgroup.BpGroup, n int) []*Curve.BIG {
	attrs := make([]*Curve.BIG, n)
	for i := 0; i < n; i++ {
		attrs[i] = Curve.Randomnum(G.Order(), G.Rng())
	}
	return attrs
}

// nolint: gosec, lll, errcheck
func main() {
	cfgFile := flag.String("f", "config.toml", "Path to the client config file.")
	flag.Parse()

	syscall.Umask(0077)

	haltCh := make(chan os.Signal)
	signal.Notify(haltCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			<-haltCh
			fmt.Println("Received SIGTERM...")
			os.Exit(0)
		}
	}()

	cfg, err := config.LoadFile(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config file '%v': %v\n", *cfgFile, err)
		os.Exit(-1)
	}

	// Start up the coconut client.
	cc, err := cclient.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to spawn client instance: %v\n", err)
		os.Exit(-1)
	}
	// testRedeem(cc)
	nymFlow(cc)
}

func testRedeem(cc *cclient.Client) {
	log, err := logger.New("", "DEBUG", false)
	if err != nil {
		panic(fmt.Sprintf("Failed to create a logger: %v", err))
	}

	tmclient, err := tmclient.New(tendermintABCIAddresses, log)
	if err != nil {
		panic(fmt.Sprintf("Failed to create a tmclient: %v", err))
	}

	pk, err := ethcrypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	newAccReq, err := transaction.CreateNewAccountRequest(pk, []byte("foo"))
	if err != nil {
		panic(err)
	}

	res, err := tmclient.Broadcast(newAccReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created new account. Code: %v, additional data: %v\n",
		code.ToString(res.DeliverTx.Code),
		string(res.DeliverTx.Data),
	)

	debugAcc, lerr := ethcrypto.LoadECDSA("tendermint/debugAccount.key")
	if lerr != nil {
		panic(lerr)
	}

	newAccAddress := ethcrypto.PubkeyToAddress(*pk.Public().(*ecdsa.PublicKey))
	debugAccAddress := ethcrypto.PubkeyToAddress(*debugAcc.Public().(*ecdsa.PublicKey))

	queryRes, err := tmclient.Query(query.QueryCheckBalancePath, debugAccAddress[:])
	if err != nil {
		panic(err)
	}

	fmt.Println("Debug Account Balance: ", binary.BigEndian.Uint64(queryRes.Response.Value))

	// transfer some funds to the new account
	transferReq, err := transaction.CreateNewTransferRequest(debugAcc, newAccAddress, 42)
	if err != nil {
		panic(err)
	}

	// add some funds
	res, err = tmclient.Broadcast(transferReq)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Transferred funds from debug to new account. Code: %v, additional data: %v\n",
		code.ToString(res.DeliverTx.Code),
		string(res.DeliverTx.Data),
	)

	tx, err := transaction.CreateNewTokenRedemptionRequest(pk, 2)
	if err != nil {
		panic(err)
	}

	res, err = tmclient.Broadcast(tx)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}

func checkNymBalance(cc *cclient.Client, log *logging.Logger) uint64 {
	log.Info("Querying for our current balance on the Nym chain")

	currentBalance, err := cc.GetCurrentNymBalance()
	if err != nil {
		panic(err)
	}

	log.Noticef("Our current balance is %vNyms", currentBalance)
	return currentBalance
}

func checkERC20NymBalance(cc *cclient.Client, log *logging.Logger) (uint64, uint64) {
	log.Info("Querying for our current balance of ERC20 Nyms on Ethereum chain")
	currentERC20Balance, err := cc.GetCurrentERC20Balance()
	if err != nil {
		panic(err)
	}
	pending, err := cc.GetCurrentERC20PendingBalance()
	if err != nil {
		panic(err)
	}
	log.Noticef("Our ERC20 balance is %v (pending %v)", currentERC20Balance, pending)
	return currentERC20Balance, pending
}

func nymFlow(cc *cclient.Client) {
	logger, err := logger.New("", "INFO", false)
	if err != nil {
		panic(err)
	}
	log := logger.GetLogger("SampleClientDemo")

	// We get our current balances
	currentNymBalance := checkNymBalance(cc, log)
	currentERC20Balance, _ := checkERC20NymBalance(cc, log)

	var tokenValue int64 = 1

	// we send some nyms to the pipe account
	log.Infof("Going to send %v Nyms from our account to the pipe account", tokenValue)
	if err := cc.SendToPipeAccount(context.TODO(), tokenValue); err != nil {
		panic(err)
	}

	// we wait for both operations to get finalized
	cc.WaitForERC20BalanceChangeWrapper(context.TODO(), currentERC20Balance-uint64(tokenValue))
	cc.WaitForBalanceIncrease(context.TODO(), currentNymBalance+uint64(tokenValue))
	log.Noticef("We sent %v to the pipe account", tokenValue)

	// and we see both balances changed accordingly
	checkNymBalance(cc, log)
	checkERC20NymBalance(cc, log)

	// generate materials for a credential
	params, err := coconut.Setup(1)
	if err != nil {
		panic(err)
	}
	s := Curve.Randomnum(params.P(), params.G.Rng())
	k := Curve.Randomnum(params.P(), params.G.Rng())
	token, err := token.New(s, k, tokenValue)
	if err != nil {
		panic(err)
	}

	log.Infof("Going to get a credential for value of %v Nyms", tokenValue)

	cred, err := cc.GetCredential(token)
	if err != nil {
		panic(err)
	}

	log.Noticef("Obtained Credential: %v %v\n", cred.Sig1().ToString(), cred.Sig2().ToString())

	// see that our balance changed
	checkNymBalance(cc, log)

	log.Info("Going to spend the obtained credential at some service provider")
	didSucceed, err := cc.SpendCredential(token, cred, provider1IP, ethcommon.HexToAddress(provider1Address), nil)
	if err != nil {
		panic(err)
	}
	if didSucceed {
		log.Notice("We managed to spend the credential successfully")
	} else {
		log.Error("For some reason, we failed to spend the credential - please refer to the provider's logs for details")
	}

	log.Warning("Going to test token redemption back to ERC20 (temporarily on completely new and fresh account until properly implemented")
	testRedeem(cc)
}
