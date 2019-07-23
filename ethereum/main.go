package main

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/nymtech/nym/ethereum/client"
	"github.com/nymtech/nym/logger"
)

// just sends some tokens to the pipe account
func main() {
	// TODO: move all of those to some .toml file
	privateKey, err := crypto.LoadECDSA("tmpPrivate")
	if err != nil {
		panic(err)
	}
	pipeContract := common.HexToAddress("0xb749305b3293477b4d6b498b22db5353c9acb3f1")
	nymContract := common.HexToAddress("0xE80025228D5448A55B995c829B89567ECE5203d3")

	log, err := logger.New("", "DEBUG", false)
	if err != nil {
		panic(err)
	}

	cfg := client.NewConfig(privateKey,
		"https://ropsten.infura.io/v3/5607a6494adb4ad4be814ec20f46ec5b", nymContract, pipeContract, log)
	c, err := client.New(cfg)
	if err != nil {
		panic(err)
	}

	if _, err := c.TransferERC20Tokens(context.TODO(), 1, pipeContract); err != nil {
		panic(err)
	}
}
