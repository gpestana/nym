// constants.go - Set of constants related to the blockchain application.
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

// Package constants declares system-wide constants.
package constants

import (
	"errors"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	// DebugMode is a flag to indicate whether the application is in debug mode.
	// If disabled some options won't be available
	DebugMode = true

	// NonceLength indicates number of bytes used for any nonces.
	NonceLength = 32
)

// TODO: requires major cleanup and removing unused entries
// TODO: change all prefixes to say only length of 8 bytes?

// nolint: gochecknoglobals
var (
	// DEPRECATED: replaced by Zeta Status
	// SpentZetaPrefix represents prefix for each zeta in the database to indicate it has been spent.
	SpentZetaPrefix = []byte("SpentZeta")

	// ZetaStatusPrefix represents prefix for each zeta in the database to indicate its status
	// (spent, being verified, unspent)
	ZetaStatusPrefix = []byte("ZetaStatus")

	// AggregateVkKey represents the database entry for the aggregate verification key of the threshold number
	// of issuing authorities of the system. It is used for credential verification.
	AggregateVkKey = []byte("AggregateVerificationKey")

	// IaKeyPrefix represents the prefix for particular issuing authority to store their keys.
	IaKeyPrefix = []byte("IssuingAuthority")

	// EthereumWatcherKeyPrefix represents the prefix for storing public keys of trusted watchers.
	EthereumWatcherKeyPrefix = []byte("EthereumWatcher")

	// CredentialVerifierKeyPrefix represents the prefix for storing public keys of trusted verifiers.
	CredentialVerifierKeyPrefix = []byte("CredentialVerifier")

	// TokenRedeemerKeyPrefix represents the prefix for storing public key of trusted redeemers.
	TokenRedeemerKeyPrefix = []byte("TokenRedeemer")

	// AccountsPrefix represents prefix for each account in the database to indicate amount of associated tokens.
	AccountsPrefix = []byte("account")

	// CoconutHsKey represents the database entry for the EC points of G1 as defined by
	// the public, system-wide coconut parameters.
	CoconutHsKey = []byte("coconutHs")

	// SeenNoncePrefix represents prefix for each seen nonce in the database.
	SeenNoncePrefix = []byte("NONCE")

	// CredentialRequestKeyPrefix represents prefix attached to key field of kvpair in the tags of response
	// to a successful request to transfer tokens to the pipe account.
	CredentialRequestKeyPrefix = []byte("GETCREDENTIAL")

	// RedeemCredentialRequestKeyPrefix represents prefix attached to key field of kvpair in the tags of response
	// to a successful request to move redeem attached credential and move tokens into corresponding Nym account.
	RedeemCredentialRequestKeyPrefix = []byte("REDEEMCREDENTIAL")

	// RedeemTokensRequestKeyPrefix represents prefix attached to key field of kvpair in the tags of response
	// to a successful request to move tokens to the corresponding ERC20 account.
	RedeemTokensRequestKeyPrefix = []byte("REDEEMTOKENS")

	// EthereumWatcherNotificationPrefix represents prefix for database entry
	// to indicate given watcher has already notified about particular transfer.
	EthereumWatcherNotificationPrefix = []byte("HOLDTRANSFNOTIF")

	// CredentialVerifierNotificationPrefix represents prefix for database entry
	// to indicate given verifier has already notified about particular credential status.
	CredentialVerifierNotificationPrefix = []byte("CREDVERIFNOTIF")

	// TokenRedeemerNotificationPrefix represents prefix for database entry
	// to indicate given redeemer has already notified and confirmed given user's intent to redeem tokens.
	TokenRedeemerNotificationPrefix = []byte("TOKENREDNOTIF")

	// PipeAccountTransferNotificationCountKeyPrefix represents prefix for the key for number of watchers
	// confirming given transfer
	PipeAccountTransferNotificationCountKeyPrefix = []byte("COUNT HODLTRANSFNOTIF")

	// CredentialVerificationNotificationCountKeyPrefix represents prefix for the key for number of verifiers
	// verifying given credential
	CredentialVerificationNotificationCountKeyPrefix = []byte("COUNT CREDVERIFNOTIF")

	// TokenRedemptionNotificationCountKeyPrefix represents prefix for the key for number of redeemers
	// confirming user's intent to redeem tokens
	TokenRedemptionNotificationCountKeyPrefix = []byte("COUNT TOKENREDNOTIF")

	// WatcherThresholdKey represents key under which watcher threshold as initially set in genesis state is stored.
	WatcherThresholdKey = []byte("WatcherThreshold")

	// VerifierThresholdKey represents key under which verifier threshold as initially set in genesis state is stored.
	VerifierThresholdKey = []byte("VerifierThreshold")

	// RedeemerThresholdKey represents key under which redeemer threshold as initially set in genesis state is stored.
	RedeemerThresholdKey = []byte("RedeemerThreshold")

	// PipeContractKey represents key under which address of the pipe account
	// as initially set in genesis state is stored.
	PipeContractKey = []byte("PipeContractAddress")

	// HashFunction defines a hash function used during signing and verification of messages sent to tendermint chain
	HashFunction = ethcrypto.Keccak256

	// ErrNotInDebug indicates error thrown when trying to access functionalities only available in debug mode
	ErrNotInDebug = errors.New("could not proceed with request. App is not in debug mode")
)

type ZetaStatus byte

const (
	// Given Zeta can have 3 states:
	// Unspent - when it was never sent to the chain before
	// Being Verified - SP sent deposit request but verifiers have not reached consensus on credential validity yet
	// Spent - SP has already been credited for credential value
	// Unspent status is never explicitly written to the database, it's being implied from lack of any entry
	ZetaStatusUnspent       ZetaStatus = 0
	ZetaStatusSpent         ZetaStatus = 1
	ZetaStatusBeingVerified ZetaStatus = 2
)

func (status ZetaStatus) DbEntry() []byte {
	return []byte{byte(status)}
}
