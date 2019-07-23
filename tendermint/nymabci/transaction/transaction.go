// transaction.go - tx logic
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

// Package transaction defines transaction logic for the Nym application.
package transaction

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	proto "github.com/golang/protobuf/proto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/common/utils"
	"github.com/nymtech/nym/constants"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	tmconst "github.com/nymtech/nym/tendermint/nymabci/constants"
)

const (
	// TxTypeLookUpZeta is byte prefix for transaction to check for presence of zeta.
	TxTypeLookUpZeta byte = 0x01
	// TxNewAccount is byte prefix for transaction to create new account.
	TxNewAccount byte = 0x02
	// TxTransferBetweenAccounts is byte prefix for transaction to transfer funds between 2 accounts. for debug
	TxTransferBetweenAccounts byte = 0x03
	// // TxTransferToPipeAccount is byte prefix for transaction to transfer client's funds to pipe account.
	// TxTransferToPipeAccount byte = 0x04
	// TxDepositCoconutCredential is byte prefix for transaction to deposit a coconut credential (+ transfer funds).
	TxDepositCoconutCredential byte = 0xa0
	// TxTransferToPipeAccountNotification is byte prefix for transaction notifying tendermint nodes about
	// transfer to pipe account that happened on ethereum chain
	TxTransferToPipeAccountNotification byte = 0xa1
	// TxCredentialRequest is byte prefix for transaction indicating client wanting to convert some of its tokens
	// into a credential
	TxCredentialRequest byte = 0xa2
	// TxCredentialVerificationNotification is byte prefix for transaction notifying tendermint nodes about
	// validity (or lack of therein) of a credential some service provider wanted to deposit.
	TxCredentialVerificationNotification byte = 0xa3
	// TxTokenRedemptionRequest is byte prefix for transaction to request transfer of tokens from the Nym system
	// back into ERC20 Nym tokens.
	TxTokenRedemptionRequest byte = 0xa4
	// TxTokenRedemptionConfirmationNotification is byte prefix for transaction notifying tendermint nodes about
	// redeemer seeing said token redemption request (so that threshold can be determined and Ethereum contract called)
	// Note that this is a dummy and quite naive implementation, but is only there to have 'a' solution as
	// in actual deployment there won't be pipe accounts as in here.
	TxTokenRedemptionConfirmationNotification byte = 0xa5
	// TxAdvanceBlock is byte prefix for transaction to store entire tx block in db to advance the blocks.
	TxAdvanceBlock byte = 0xff // entirely for debug purposes
)

func marshalRequest(req proto.Message, prefix byte) ([]byte, error) {
	protob, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	b := make([]byte, len(protob)+1)
	b[0] = prefix
	copy(b[1:], protob)
	return b, nil
}

// NewLookUpZetaTx creates new request for tx to lookup provided zeta.
func NewLookUpZetaTx(zeta *Curve.ECP) []byte {
	tx := make([]byte, 1+constants.ECPLen)
	zb := make([]byte, constants.ECPLen)
	zeta.ToBytes(zb, true)

	tx[0] = TxTypeLookUpZeta
	copy(tx[1:], zb)
	return tx
}

// CreateNewAccountRequest creates new request for tx for new account creation.
func CreateNewAccountRequest(privateKey *ecdsa.PrivateKey, credential []byte) ([]byte, error) {
	addr := ethcrypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))
	msg := make([]byte, len(addr)+len(credential))
	copy(msg, addr[:])
	copy(msg[len(addr):], credential)
	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), privateKey)
	if err != nil {
		return nil, err
	}

	req := &NewAccountRequest{
		Address:    addr[:],
		Credential: credential,
		Sig:        sig,
	}
	return marshalRequest(req, TxNewAccount)
}

// CreateNewTransferRequest creates new request for tx to transfer funds from one account to another.
// Currently and possibly only for debug purposes
// to freely transfer tokens between accounts to setup different scenarios.
func CreateNewTransferRequest(sourcePrivateKey *ecdsa.PrivateKey,
	targetAddress ethcommon.Address,
	amount uint64,
) ([]byte, error) {

	nonce, err := utils.GenerateRandomBytes(tmconst.NonceLength)
	if err != nil {
		return nil, err
	}

	sourceAddress := ethcrypto.PubkeyToAddress(*sourcePrivateKey.Public().(*ecdsa.PublicKey))

	// msg := make([]byte, 2*ethcommon.AddressLength+tmconst.NonceLength+8)
	// copy(msg, sourceAddress[:])
	// copy(msg[ethcommon.AddressLength:], targetAddress[:])
	// binary.BigEndian.PutUint64(msg[2*ethcommon.AddressLength:], amount)
	// copy(msg[2*ethcommon.AddressLength+8:], nonce)

	msg := make([]byte, 2*ethcommon.AddressLength+tmconst.NonceLength+8)
	i := copy(msg, sourceAddress[:])
	i += copy(msg[i:], targetAddress[:])
	binary.BigEndian.PutUint64(msg[i:], amount)
	i += 8
	copy(msg[i:], nonce)

	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), sourcePrivateKey)
	if err != nil {
		return nil, err
	}

	req := &AccountTransferRequest{
		SourceAddress: sourceAddress[:],
		TargetAddress: targetAddress[:],
		Amount:        amount,
		Nonce:         nonce,
		Sig:           sig,
	}
	return marshalRequest(req, TxTransferBetweenAccounts)
}

// CreateNewDepositCoconutCredentialRequest creates new request for tx to send credential created out of given token
// (that is bound to particular merchant address) to be spent.
func CreateNewDepositCoconutCredentialRequest(
	protoSig *coconut.ProtoSignature,
	pubMb [][]byte,
	protoThetaTumbler *coconut.ProtoThetaTumbler,
	value int64,
	address ethcommon.Address,
) ([]byte, error) {

	cryptoMaterials := &coconut.ProtoTumblerBlindVerifyMaterials{
		Sig:   protoSig,
		PubM:  pubMb,
		Theta: protoThetaTumbler,
	}

	req := &DepositCoconutCredentialRequest{
		CryptoMaterials: cryptoMaterials,
		Value:           value,
		ProviderAddress: address[:],
	}

	return marshalRequest(req, TxDepositCoconutCredential)
}

func CreateNewTransferToPipeAccountNotification(privateKey *ecdsa.PrivateKey,
	clientAddress ethcommon.Address,
	pipeAccountAddress ethcommon.Address,
	amount uint64,
	txHash ethcommon.Hash,
) ([]byte, error) {

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	publicKeyBytes := ethcrypto.FromECDSAPub(publicKey)

	// msg := make([]byte, len(publicKeyBytes)+2*ethcommon.AddressLength+8+ethcommon.HashLength)
	// copy(msg, publicKeyBytes)
	// copy(msg[len(publicKeyBytes):], clientAddress[:])
	// copy(msg[len(publicKeyBytes)+ethcommon.AddressLength:], pipeAccountAddress[:])
	// binary.BigEndian.PutUint64(msg[len(publicKeyBytes)+2*ethcommon.AddressLength:], amount)
	// copy(msg[len(publicKeyBytes)+2*ethcommon.AddressLength+8:], txHash[:])

	msg := make([]byte, len(publicKeyBytes)+2*ethcommon.AddressLength+8+ethcommon.HashLength)
	i := copy(msg, publicKeyBytes)
	i += copy(msg[i:], clientAddress[:])
	i += copy(msg[i:], pipeAccountAddress[:])
	binary.BigEndian.PutUint64(msg[i:], amount)
	i += 8
	copy(msg[i:], txHash[:])

	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), privateKey)
	if err != nil {
		return nil, err
	}

	req := &TransferToPipeAccountNotification{
		WatcherPublicKey:   publicKeyBytes,
		ClientAddress:      clientAddress[:],
		PipeAccountAddress: pipeAccountAddress[:],
		Amount:             amount,
		TxHash:             txHash[:],
		Sig:                sig,
	}
	return marshalRequest(req, TxTransferToPipeAccountNotification)
}

func CreateNewCredentialRequest(privateKey *ecdsa.PrivateKey,
	pipeAccountAddress ethcommon.Address,
	bsm *coconut.BlindSignMaterials,
	value int64,
) ([]byte, error) {

	if value <= 0 {
		return nil, errors.New("invalid credential value")
	}

	nonce, err := utils.GenerateRandomBytes(tmconst.NonceLength)
	if err != nil {
		return nil, err
	}

	protoBlindSignMaterials, err := bsm.ToProto()
	if err != nil {
		return nil, err
	}

	// can't just marshal the proto materials to bytes as this serialisation is not guaranteed to be deterministic
	bsmBytes, err := protoBlindSignMaterials.OneWayToBytes()
	if err != nil {
		return nil, err
	}

	address := ethcrypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))

	msg := make([]byte, 2*ethcommon.AddressLength+len(bsmBytes)+8+tmconst.NonceLength)
	i := copy(msg, address[:])
	i += copy(msg[i:], pipeAccountAddress[:])
	i += copy(msg[i:], bsmBytes)
	binary.BigEndian.PutUint64(msg[i:], uint64(value))
	i += 8
	copy(msg[i:], nonce)

	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), privateKey)
	if err != nil {
		return nil, err
	}

	req := &CredentialRequest{
		ClientAddress:      address[:],
		PipeAccountAddress: pipeAccountAddress[:],
		CryptoMaterials:    protoBlindSignMaterials,
		Value:              value,
		Nonce:              nonce,
		Sig:                sig,
	}
	return marshalRequest(req, TxCredentialRequest)
}

func CreateNewCredentialVerificationNotification(privateKey *ecdsa.PrivateKey,
	providerAddress ethcommon.Address,
	value int64,
	zeta []byte,
	wasValid bool,
) ([]byte, error) {

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	publicKeyBytes := ethcrypto.FromECDSAPub(publicKey)

	msg := make([]byte, len(publicKeyBytes)+ethcommon.AddressLength+8+len(zeta)+1)
	i := copy(msg, publicKeyBytes)
	i += copy(msg[i:], providerAddress[:])
	binary.BigEndian.PutUint64(msg[i:], uint64(value))
	i += 8
	if wasValid {
		msg[i] = 1
	}
	// by default it's 0

	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), privateKey)
	if err != nil {
		return nil, err
	}

	req := &CredentialVerificationNotification{
		VerifierPublicKey:  publicKeyBytes,
		ProviderAddress:    providerAddress[:],
		Value:              value,
		Zeta:               zeta,
		CredentialValidity: wasValid,
		Sig:                sig,
	}
	return marshalRequest(req, TxCredentialVerificationNotification)
}

func CreateNewTokenRedemptionRequest(privateKey *ecdsa.PrivateKey, amount uint64) ([]byte, error) {
	address := ethcrypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))

	nonce, err := utils.GenerateRandomBytes(tmconst.NonceLength)
	if err != nil {
		return nil, err
	}

	msg := make([]byte, ethcommon.AddressLength+8+tmconst.NonceLength)
	i := copy(msg, address[:])
	binary.BigEndian.PutUint64(msg[i:], amount)
	i += 8
	copy(msg[i:], nonce)

	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), privateKey)
	if err != nil {
		return nil, err
	}

	req := &TokenRedemptionRequest{
		UserAddress: address[:],
		Amount:      amount,
		Nonce:       nonce,
		Sig:         sig,
	}
	return marshalRequest(req, TxTokenRedemptionRequest)
}

func CreateNewTokenRedemptionConfirmationNotification(privateKey *ecdsa.PrivateKey,
	userAddress ethcommon.Address,
	amount uint64,
	nonce []byte,
) ([]byte, error) {

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	publicKeyBytes := ethcrypto.FromECDSAPub(publicKey)

	msg := make([]byte, len(publicKeyBytes)+ethcommon.AddressLength+ethcommon.AddressLength+8+tmconst.NonceLength)
	i := copy(msg, publicKeyBytes)
	i += copy(msg[i:], userAddress[:])
	binary.BigEndian.PutUint64(msg[i:], amount)
	i += 8
	copy(msg[i:], nonce)

	sig, err := ethcrypto.Sign(tmconst.HashFunction(msg), privateKey)
	if err != nil {
		return nil, err
	}

	req := &TokenRedemptionConfirmationNotification{
		RedeemerPublicKey: publicKeyBytes,
		UserAddress:       userAddress[:],
		Amount:            amount,
		Nonce:             nonce,
		Sig:               sig,
	}
	return marshalRequest(req, TxTokenRedemptionConfirmationNotification)
}
