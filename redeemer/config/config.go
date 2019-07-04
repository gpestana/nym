// config.go - config for Nym redeemer
// Copyright (C) 2019  Jedrzej Stuczynski.
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

// Package config defines configuration used by Nym redeemer.
package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	defaultLogLevel = "NOTICE"

	defaultEthereumCallTimeout            = 15 * 1000 // 15s
	defaultTransactionStatusQueryTimeout  = 90 * 1000 // 90s
	defaultTransactionStatusQueryInterval = 5 * 1000  // 5s
	// defaultNumServerWorkers = 1
)

// nolint: gochecknoglobals
var defaultLogging = Logging{
	Disable: false,
	File:    "",
	Level:   defaultLogLevel,
}

// Redeemer is the main Nym redeemer configuration.
type Redeemer struct {
	// Identifier is the human readable identifier for the node.
	Identifier string

	// KeyFile defines path to file containing ECDSA private key of the redeemer.
	KeyFile string

	// DataDir specifies path to a .db file holding relevant server-specific persistent data.
	DataDir string

	// BlockchainNodeAddresses specifies addresses of Tendermint blockchain nodes
	// to which the issuer should send all relevant requests.
	// Note that only a single request will ever be sent, but multiple addresses are provided in case
	// the particular node was unavailable.
	BlockchainNodeAddresses []string

	// PipeAccountKeyFile defines path to file containing ECDSA private key for the pipe account contract.
	PipeAccountKeyFile string

	// EthereumNodeAddress defines address of an Ethereum node to which transactions are sent.
	EthereumNodeAddress string

	// NymContract defined address of the ERC20 token Nym contract. It is expected to be provided in hex format.
	NymContract ethcommon.Address
}

// Debug is the Nym redeemer debug configuration.
type Debug struct {
	// // NumJobWorkers specifies the number of worker instances to use for jobpacket processing.
	// NumJobWorkers int

	// // NumServerWorkers specifies the number of concurrent worker instances
	// // to use when processing verification requests.
	// NumServerWorkers int

	// EthereumCallTimeout defines timeout for calling the Ethereum contract to transfer tokens back from the pipe account.
	EthereumCallTimeout int

	// TransactionStatusQueryTimeout defines timeout for waiting to obtain status of particular transaction.
	TransactionStatusQueryTimeout int

	// TransactionStatusQueryInterval defines interval for querying for status of particular transaction.
	TransactionStatusQueryInterval int
}

func (dCfg *Debug) applyDefaults() {
	// if dCfg.NumJobWorkers <= 0 {
	// 	dCfg.NumJobWorkers = runtime.NumCPU()
	// }
	// if dCfg.NumServerWorkers <= 0 {
	// 	dCfg.NumServerWorkers = defaultNumServerWorkers
	// }

	if dCfg.EthereumCallTimeout <= 0 {
		dCfg.EthereumCallTimeout = defaultEthereumCallTimeout
	}
	if dCfg.TransactionStatusQueryTimeout <= 0 {
		dCfg.TransactionStatusQueryTimeout = defaultTransactionStatusQueryTimeout
	}
	if dCfg.TransactionStatusQueryInterval <= 0 {
		dCfg.TransactionStatusQueryInterval = defaultTransactionStatusQueryInterval
	}
}

// Logging is the Nym redeemer logging configuration.
type Logging struct {
	// Disable disables logging entirely.
	Disable bool

	// File specifies the log file, if omitted stdout will be used.
	File string

	// Level specifies the log level.
	Level string
}

// Config is the top level Nym redeemer configuration.
type Config struct {
	Redeemer *Redeemer
	Logging  *Logging
	Debug    *Debug
}

// nolint: gocyclo
func (cfg *Config) validateAndApplyDefaults() error {
	if cfg.Redeemer == nil {
		return errors.New("config: No Redeemer block was present")
	}

	if _, err := os.Stat(cfg.Redeemer.KeyFile); err != nil {
		return fmt.Errorf("config: The specified key file does not seem to exist: %v", err)
	}

	if cfg.Debug == nil {
		cfg.Debug = &Debug{}
	}
	cfg.Debug.applyDefaults()

	if cfg.Logging == nil {
		cfg.Logging = &defaultLogging
	}

	return nil
}

// LoadBinary loads, parses and validates the provided buffer b (as a config)
// and returns the Config.
func LoadBinary(b []byte) (*Config, error) {
	cfg := new(Config)
	_, err := toml.Decode(string(b), cfg)
	if err != nil {
		return nil, err
	}
	if err := cfg.validateAndApplyDefaults(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadFile loads, parses and validates the provided file and returns the Config.
func LoadFile(f string) (*Config, error) {
	b, err := ioutil.ReadFile(filepath.Clean(f))
	if err != nil {
		return nil, err
	}
	return LoadBinary(b)
}
