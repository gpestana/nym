// deliver.go - state manipulation logic for Tendermint ABCI for Nym
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

package nymapplication

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	tmconst "github.com/nymtech/nym/tendermint/nymabci/constants"
	"github.com/tendermint/iavl"
)

var (
	// ErrKeyDoesNotExist represents error thrown when trying to look-up non-existent key
	ErrKeyDoesNotExist = errors.New("the specified key does not exist in the database")
)

// State defines ABCI app state. Currently it is a iavl tree. Reason for the choice: it was recurring case in examples.
// It provides height (changes after each save -> perfect for blockchain) + fast hash which is also needed.
type State struct {
	db *iavl.MutableTree // hash and height (version) are obtained from the tree methods

	watcherThreshold  uint32
	verifierThreshold uint32
	pipeAccount       ethcommon.Address
}

func (app *NymApplication) storeWatcherThreshold() {
	thrb := make([]byte, 4)
	binary.BigEndian.PutUint32(thrb, app.state.watcherThreshold)
	app.state.db.Set(tmconst.WatcherThresholdKey, thrb)
}

func (app *NymApplication) loadWatcherThreshold() error {
	_, val := app.state.db.Get(tmconst.WatcherThresholdKey)
	if val == nil {
		return ErrKeyDoesNotExist
	}
	app.state.watcherThreshold = binary.BigEndian.Uint32(val)
	app.log.Info(fmt.Sprintf("Loaded watcher threshold: %v", app.state.watcherThreshold))
	return nil
}

func (app *NymApplication) storeVerifierThreshold() {
	thrb := make([]byte, 4)
	binary.BigEndian.PutUint32(thrb, app.state.verifierThreshold)
	app.state.db.Set(tmconst.VerifierThresholdKey, thrb)
}

func (app *NymApplication) loadVerifierThreshold() error {
	_, val := app.state.db.Get(tmconst.VerifierThresholdKey)
	if val == nil {
		return ErrKeyDoesNotExist
	}
	app.state.verifierThreshold = binary.BigEndian.Uint32(val)
	app.log.Info(fmt.Sprintf("Loaded verifier threshold: %v", app.state.verifierThreshold))
	return nil
}

func (app *NymApplication) storePipeAccountAddress() {
	app.state.db.Set(tmconst.PipeContractKey, app.state.pipeAccount[:])
}

func (app *NymApplication) loadPipeAccountAddress() error {
	_, val := app.state.db.Get(tmconst.PipeContractKey)
	if val == nil {
		return ErrKeyDoesNotExist
	}
	app.state.pipeAccount = ethcommon.BytesToAddress(val)
	app.log.Info(fmt.Sprintf("Loaded pipe account address: %v", app.state.pipeAccount.Hex()))
	return nil
}

// TODO: will we still need it?
// we will need to have access to g1, g2 and hs in order to verify credentials
// while we can get g1 and g2 from curve params, hs depends on number of attributes
// so store them; the points are always compressed
func (app *NymApplication) storeHs(hs []*Curve.ECP) {
	hsb := coconut.ECPSliceToCompressedBytes(hs)
	app.state.db.Set(tmconst.CoconutHsKey, hsb)
	app.log.Info(fmt.Sprintf("Stored hs in DB"))
}

func (app *NymApplication) retrieveHs() ([]*Curve.ECP, error) {
	_, hsb := app.state.db.Get(tmconst.CoconutHsKey)
	if hsb == nil {
		return nil, ErrKeyDoesNotExist
	}
	return coconut.CompressedBytesToECPSlice(hsb), nil
}

// TODO: will we still need it?
func (app *NymApplication) storeAggregateVerificationKey(avk *coconut.VerificationKey) {
	avkb, err := avk.MarshalBinary()
	if err != nil {
		panic(err)
	}

	app.state.db.Set(tmconst.AggregateVkKey, avkb)
	app.log.Info(fmt.Sprintf("Stored Aggregate Verification Key in DB"))
}

func (app *NymApplication) retrieveAggregateVerificationKey() (*coconut.VerificationKey, error) {
	_, avkb := app.state.db.Get(tmconst.AggregateVkKey)
	if avkb == nil {
		return nil, ErrKeyDoesNotExist
	}
	avk := &coconut.VerificationKey{}
	err := avk.UnmarshalBinary(avkb)
	if err != nil {
		app.log.Error("failed to unmarshal stored aggregated verification key")
		return nil, errors.New("failed to unmarshal stored aggregated verification key")
	}
	return avk, nil
}

// TODO: will we still need it?
func (app *NymApplication) storeIssuerKey(key []byte, id int64) {
	idb := make([]byte, 8)
	binary.BigEndian.PutUint64(idb, uint64(id))

	dbEntry := prefixKey(tmconst.IaKeyPrefix, idb)
	app.state.db.Set(dbEntry, key)
}

func (app *NymApplication) setAccountBalance(address []byte, value uint64) {
	balance := make([]byte, 8)
	binary.BigEndian.PutUint64(balance, value)

	dbEntry := prefixKey(tmconst.AccountsPrefix, address)
	app.state.db.Set(dbEntry, balance)

	app.log.Debug(fmt.Sprintf("Set balance of: %v to: %v", ethcommon.BytesToAddress(address).Hex(), value))
}

func (app *NymApplication) retrieveAccountBalance(address []byte) (uint64, error) {
	if len(address) != ethcommon.AddressLength {
		return 0, errors.New("invalid address length")
	}

	app.log.Debug(fmt.Sprintf("Checking balance for: %v", ethcommon.BytesToAddress(address).Hex()))
	dbEntry := prefixKey(tmconst.AccountsPrefix, address)

	_, val := app.state.db.Get(dbEntry)
	if val == nil {
		return 0, ErrKeyDoesNotExist
	}

	return binary.BigEndian.Uint64(val), nil
}

func (app *NymApplication) storeWatcherKey(watcher Watcher) {
	pubB64 := base64.StdEncoding.EncodeToString(watcher.PublicKey)
	app.log.Debug(fmt.Sprintf("Adding to the trusted set watcher with public key: %v", pubB64))
	dbEntry := prefixKey(tmconst.EthereumWatcherKeyPrefix, watcher.PublicKey)
	// TODO: do we even need to set any meaningful value here?
	app.state.db.Set(dbEntry, tmconst.EthereumWatcherKeyPrefix)
}

func (app *NymApplication) checkWatcherKey(publicKey []byte) bool {
	dbEntry := prefixKey(tmconst.EthereumWatcherKeyPrefix, publicKey)
	return app.state.db.Has(dbEntry)
}

func (app *NymApplication) storeVerifierKey(verifier Verifier) {
	pubB64 := base64.StdEncoding.EncodeToString(verifier.PublicKey)
	app.log.Debug(fmt.Sprintf("Adding to the trusted set verifier with public key: %v", pubB64))
	dbEntry := prefixKey(tmconst.CredentialVerifierKeyPrefix, verifier.PublicKey)
	// TODO: do we even need to set any meaningful value here?
	app.state.db.Set(dbEntry, tmconst.CredentialVerifierKeyPrefix)
}

func (app *NymApplication) checkVerifierKey(publicKey []byte) bool {
	dbEntry := prefixKey(tmconst.CredentialVerifierKeyPrefix, publicKey)
	return app.state.db.Has(dbEntry)
}

// checks if given (random) nonce was already seen before for the particular address
func (app *NymApplication) checkNonce(nonce, address []byte) bool {
	if len(nonce) != tmconst.NonceLength || len(address) != ethcommon.AddressLength {
		return true
	}

	// [PREFIX || NONCE || ADDRESS]
	key := prefixKey(tmconst.SeenNoncePrefix, prefixKey(nonce, address))
	return app.state.db.Has(key)
}

func (app *NymApplication) setNonce(nonce, address []byte) {
	key := prefixKey(tmconst.SeenNoncePrefix, prefixKey(nonce, address))
	// [PREFIX || NONCE || ADDRESS]
	app.state.db.Set(key, tmconst.SeenNoncePrefix)
}

// returns new number of notifications received for this transaction
func (app *NymApplication) storeWatcherNotification(watcherKey, txHash []byte) uint32 {
	// first set this watcher
	key := prefixKey(tmconst.EthereumWatcherNotificationPrefix, prefixKey(txHash, watcherKey))
	// [PREFIX || TXHASH || WATCHER ]
	// again, does the value matter here? we could just set an empty array to save on space
	// now increase notification count for this transaction
	app.state.db.Set(key, tmconst.EthereumWatcherNotificationPrefix)
	// and update total count
	newCount := app.getPipeTransferNotificationCount(txHash) + 1
	app.updatePipeTransferNotificationCount(txHash, newCount)
	return newCount
}

// checks if this watcher has already sent notification regarding this transaction
func (app *NymApplication) checkWatcherNotification(watcherKey, txHash []byte) bool {
	key := prefixKey(tmconst.EthereumWatcherNotificationPrefix, prefixKey(txHash, watcherKey))
	return app.state.db.Has(key)
}

func (app *NymApplication) getPipeTransferNotificationCount(txHash []byte) uint32 {
	key := prefixKey(tmconst.PipeAccountTransferNotificationCountKeyPrefix, txHash)

	_, val := app.state.db.Get(key)
	if val == nil {
		return 0
	}
	return binary.BigEndian.Uint32(val)
}

func (app *NymApplication) updatePipeTransferNotificationCount(txHash []byte, count uint32) {
	key := prefixKey(tmconst.PipeAccountTransferNotificationCountKeyPrefix, txHash)
	countb := make([]byte, 4)
	binary.BigEndian.PutUint32(countb, count)

	app.state.db.Set(key, countb)
}

// additional data is used as a suffix for zeta status
// currently only used for spent status to more easily query for whom redeemed given zeta
func (app *NymApplication) setZetaStatus(zeta []byte, status tmconst.ZetaStatus, additionalData ...byte) {
	key := prefixKey(tmconst.ZetaStatusPrefix, zeta)
	value := status.DbEntry()
	if len(additionalData) > 0 {
		value = prefixKey(value, additionalData)
	}
	app.state.db.Set(key, value)
}

func (app *NymApplication) checkIfZetaIsUnspent(zeta []byte) bool {
	key := prefixKey(tmconst.ZetaStatusPrefix, zeta)
	_, status := app.state.db.Get(key)
	if status == nil {
		return true
	}
	return bytes.HasPrefix(status, tmconst.ZetaStatusSpent.DbEntry())
}

func (app *NymApplication) checkZetaStatus(zeta []byte) tmconst.ZetaStatus {
	key := prefixKey(tmconst.ZetaStatusPrefix, zeta)
	_, status := app.state.db.Get(key)
	if status == nil {
		return tmconst.ZetaStatusUnspent
	}
	if bytes.HasPrefix(status, tmconst.ZetaStatusBeingVerified.DbEntry()) {
		return tmconst.ZetaStatusBeingVerified
	}
	if bytes.HasPrefix(status, tmconst.ZetaStatusSpent.DbEntry()) {
		return tmconst.ZetaStatusSpent
	}
	// should never happen, but if unsure, always assume it's already spent and gone
	return tmconst.ZetaStatusSpent
}

// returns new number of notifications received for this transaction
// TODO: rethink if we need to include value in the key field or even at all here, because in principle
// there can be no other credential of different value with the same zeta
func (app *NymApplication) storeVerifierNotification(verifierKey, zeta []byte, value int64, valid bool) uint32 {
	key := make([]byte, len(verifierKey)+len(zeta)+8)
	i := copy(key, verifierKey)
	i += copy(key[i:], zeta)
	binary.BigEndian.PutUint64(key[i:], uint64(value))

	// [PREFIX || VERIFIER || uint64(VALUE) || ZETA ]
	key = prefixKey(tmconst.CredentialVerifierNotificationPrefix, key)

	// again, does the value matter here? we could just set an empty array to save on space
	// first store that given verifier sent the notification
	app.state.db.Set(key, tmconst.CredentialVerifierNotificationPrefix)

	// then update the global count, but only if credential is valid
	currentCount := app.getCredentialVerificationCount(zeta, value)
	if valid {
		newCount := currentCount + 1
		app.updateCredentialVerificationNotificationCount(zeta, value, newCount)
		return newCount
	}
	return currentCount
}

// checks if this verifier has already sent notification regarding this credential
func (app *NymApplication) checkVerifierNotification(verifierKey, zeta []byte, value int64) bool {
	key := make([]byte, len(verifierKey)+len(zeta)+8)
	i := copy(key, verifierKey)
	i += copy(key[i:], zeta)
	binary.BigEndian.PutUint64(key[i:], uint64(value))

	// [PREFIX || VERIFIER || uint64(VALUE) || ZETA ]
	key = prefixKey(tmconst.CredentialVerifierNotificationPrefix, key)

	return app.state.db.Has(key)
}

func (app *NymApplication) getCredentialVerificationCount(zeta []byte, value int64) uint32 {
	key := make([]byte, len(zeta)+8)
	i := copy(key, zeta)
	binary.BigEndian.PutUint64(key[i:], uint64(value))

	// [PREFIX || uint64(VALUE) || ZETA ]
	key = prefixKey(tmconst.CredentialVerificationNotificationCountKeyPrefix, key)

	_, val := app.state.db.Get(key)
	if val == nil {
		return 0
	}
	return binary.BigEndian.Uint32(val)
}

func (app *NymApplication) updateCredentialVerificationNotificationCount(zeta []byte, value int64, count uint32) {
	key := make([]byte, len(zeta)+8)
	i := copy(key, zeta)
	binary.BigEndian.PutUint64(key[i:], uint64(value))

	// [PREFIX || uint64(VALUE) || ZETA ]
	key = prefixKey(tmconst.CredentialVerificationNotificationCountKeyPrefix, key)

	countb := make([]byte, 4)
	binary.BigEndian.PutUint32(countb, count)

	app.state.db.Set(key, countb)
}
