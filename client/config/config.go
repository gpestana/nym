// config.go - config for coconut client
// Copyright (C) 2018  Jedrzej Stuczynski.
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

// Package config defines configuration used by coconut client.
package config

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// TODO: refactor config structure, change section names, move attributes around, etc.

const (
	defaultLogLevel = "NOTICE"

	defaultConnectTimeout    = 15 * 1000 // 15 sec.
	defaultRequestTimeout    = 30 * 1000 // 30 sec.
	defaultMaxRequests       = 3
	noLimitMaxRequests       = 16
	defaultMaximumAttributes = 5

	defaultLookUpBackoff         = 10 * 1000 // 10 sec.
	defaultNumberOfLookUpRetries = 5
)

// nolint: gochecknoglobals
var defaultLogging = Logging{
	Disable: false,
	File:    "",
	Level:   defaultLogLevel,
}

// Client is the Coconut Client configuration.
type Client struct {
	// Identifier is the human readable identifier for the instance.
	Identifier string

	// IAAddresses are the IP address:port combinations of Issuing Authority Servers.
	IAAddresses []string

	// UseGRPC specifies whether to use gRPC for sending server requests or TCP sockets.
	UseGRPC bool

	// IAAddresses are the gRPC IP address:port combinations of Issuing Authority Servers.
	IAgRPCAddresses []string

	// MaxRequests defines maximum number of concurrent requests each client can make.
	// -1 indicates no limit
	MaxRequests int

	// Threshold defines minimum number of signatures client needs to obtain. Default = len(IAAddresses).
	// 0 = no threshold
	Threshold int

	// MaximumAttributes specifies the maximum number of attributes the client will want to have signed.
	MaximumAttributes int
}

// Nym defines Nym-specific configuration options.
type Nym struct {
	// NymContract defined address of the ERC20 token Nym contract. It is expected to be provided in hex format.
	NymContract ethcommon.Address

	// PipeAccount defines address of Ethereum account that pipes Nym ERC20 into Nym Tendermint coins.
	// It is expected to be provided in hex format.
	PipeAccount ethcommon.Address

	// AccountKeysFile specifies the file containing keys used for the accounts on the Nym Blockchain.
	AccountKeysFile string

	// BlockchainNodeAddresses specifies addresses of blockchain nodes
	// to which the client should send all relevant requests.
	// Note that only a single request will ever be sent, but multiple addresses are provided in case
	// the particular node was unavailable.
	BlockchainNodeAddresses []string

	// EthereumNodeAddresses specifies addresses of Ethereum nodes
	// to which the client should send all relevant requests.
	// Note that only a single request will ever be sent, but multiple addresses are provided in case
	// the particular node was unavailable. (TODO: implement this functionality)
	EthereumNodeAddresses []string
}

// Debug is the Coconut Client debug configuration.
type Debug struct {
	// NumJobWorkers specifies the number of worker instances to use for jobpacket processing.
	NumJobWorkers int

	// ConnectTimeout specifies the maximum time a connection can take to establish a TCP/IP connection in milliseconds.
	ConnectTimeout int

	// RequestTimeout specifies the maximum time a client is going to wait for its request to resolve.
	RequestTimeout int

	// RegenerateKeys specifies whether to generate new Coconut-specific ElGamal keypair and overwrite existing files.
	RegenerateKeys bool

	// NumberOfLookUpRetries specifies maximum number of retries to call issuer to look up the credentials.
	NumberOfLookUpRetries int

	// LookUpBackoff specifies the backoff duration after failing to look up credential
	// (assuming it was due to not being processed yet).
	LookUpBackoff int
}

func (dCfg *Debug) applyDefaults() {
	if dCfg.NumJobWorkers <= 0 {
		dCfg.NumJobWorkers = runtime.NumCPU()
	}
	if dCfg.ConnectTimeout <= 0 {
		dCfg.ConnectTimeout = defaultConnectTimeout
	}
	if dCfg.RequestTimeout <= 0 {
		dCfg.RequestTimeout = defaultRequestTimeout
	}
	if dCfg.NumberOfLookUpRetries <= 0 {
		dCfg.NumberOfLookUpRetries = defaultNumberOfLookUpRetries
	}
	if dCfg.LookUpBackoff <= 0 {
		dCfg.LookUpBackoff = defaultLookUpBackoff
	}
}

// Logging is the Coconut Client logging configuration.
type Logging struct {
	// Disable disables logging entirely.
	Disable bool

	// File specifies the log file, if omitted stdout will be used.
	File string

	// Level specifies the log level.
	Level string
}

// Config is the top level Coconut Client configuration.
type Config struct {
	Client  *Client
	Nym     *Nym
	Logging *Logging

	Debug *Debug
}

// nolint: gocyclo
func (cfg *Config) validateAndApplyDefaults() error {
	if cfg.Client == nil {
		return errors.New("config: No Client block was present")
	}

	if cfg.Client.MaxRequests == 0 {
		cfg.Client.MaxRequests = defaultMaxRequests
	} else if cfg.Client.MaxRequests < 0 {
		cfg.Client.MaxRequests = noLimitMaxRequests
	}

	if cfg.Client.MaximumAttributes == 0 {
		cfg.Client.MaximumAttributes = defaultMaximumAttributes
	}

	if len(cfg.Client.IAAddresses) == 0 && !cfg.Client.UseGRPC {
		return errors.New("config: No server addresses provided")
	}

	if len(cfg.Client.IAgRPCAddresses) == 0 && cfg.Client.UseGRPC {
		return errors.New("config: No server gRPC addresses provided")
	}

	if cfg.Debug == nil {
		cfg.Debug = &Debug{}
	}
	cfg.Debug.applyDefaults()

	if cfg.Logging == nil {
		cfg.Logging = &defaultLogging
	}

	if cfg.Nym == nil {
		return errors.New("config: No Nym block was present")
	}
	if len(cfg.Nym.AccountKeysFile) == 0 {
		return errors.New("config: No key file provided")
	}
	if len(cfg.Nym.BlockchainNodeAddresses) == 0 {
		return errors.New("config: No node addresses provided")
	}
	if len(cfg.Nym.EthereumNodeAddresses) == 0 {
		return errors.New("config: No ethereum node addresses provider")
	}
	if len(cfg.Nym.NymContract) == 0 {
		return errors.New("config: Unspecified address of the Nym contract")
	}
	if len(cfg.Nym.PipeAccount) == 0 {
		return errors.New("config: Unspecified address of the Pipe account")
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
