// deliver.go - DeliverTx-related logic for Tendermint ABCI for Nym
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
	"encoding/base64"
	"encoding/binary"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"github.com/nymtech/nym/constants"
	"github.com/nymtech/nym/tendermint/nymabci/code"
	tmconst "github.com/nymtech/nym/tendermint/nymabci/constants"
	"github.com/nymtech/nym/tendermint/nymabci/transaction"
	"github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
)

const (
	startingBalance uint64 = 0 // this is for purely debug purposes. It will always be 0
)

// tx prefix was already removed
func (app *NymApplication) createNewAccount(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.NewAccountRequest{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		app.log.Info("Failed to unmarshal request")
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkNewAccountTx(reqb); checkResult != code.OK {
		app.log.Info("CreateNewAccount failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	// we already know recAddr is identical to the address sent
	didSucceed := app.createNewAccountOp(ethcommon.BytesToAddress(req.Address))
	if didSucceed {
		return types.ResponseDeliverTx{Code: code.OK}
	}
	return types.ResponseDeliverTx{Code: code.UNKNOWN}
}

// Currently and possibly only for debug purposes
// to freely transfer tokens between accounts to setup different scenarios.
func (app *NymApplication) transferFunds(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.AccountTransferRequest{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		app.log.Info("Failed to unmarshal request")
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkTransferBetweenAccountsTx(reqb); checkResult != code.OK {
		app.log.Info("TransferFunds failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	retCode, data := app.transferFundsOp(req.SourceAddress, req.TargetAddress, req.Amount)
	if retCode == code.OK {
		app.setNonce(req.Nonce, req.SourceAddress)
	}
	return types.ResponseDeliverTx{Code: retCode, Data: data}
}

func (app *NymApplication) handleTransferToPipeAccountNotification(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.TransferToPipeAccountNotification{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		app.log.Info("Failed to unmarshal request")
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkTransferToPipeAccountNotificationTx(reqb); checkResult != code.OK {
		app.log.Info("HandlePipeTransferNotification failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	// 'accept' the notification
	newCount := app.storeWatcherNotification(req.WatcherPublicKey, req.TxHash)

	app.log.Debug(fmt.Sprintf("Reached %v notifications out of required %v for %v",
		newCount,
		app.state.watcherThreshold,
		ethcommon.BytesToHash(req.TxHash).Hex(),
	))

	// commit the transaction if threshold is reached
	if newCount == app.state.watcherThreshold {
		app.log.Debug(fmt.Sprintf("Reached required threshold of %v for %v",
			app.state.watcherThreshold,
			ethcommon.BytesToHash(req.TxHash).Hex(),
		))
		// check if account exists
		currentBalance, err := app.retrieveAccountBalance(req.ClientAddress)
		if err != nil && createAccountOnPipeAccountTransferIfDoesntExist {
			didSucceed := app.createNewAccountOp(ethcommon.BytesToAddress(req.ClientAddress))
			if !didSucceed {
				app.log.Info(fmt.Sprintf("Failed to create new account for the client with address %v",
					ethcommon.BytesToAddress(req.ClientAddress).Hex()))
				return types.ResponseDeliverTx{Code: code.UNKNOWN}
			}
		} else if err != nil {
			app.log.Info("Client's account does not exist and system is not set to create new ones")
			return types.ResponseDeliverTx{Code: code.ACCOUNT_DOES_NOT_EXIST}
		}

		app.setAccountBalance(req.ClientAddress, currentBalance+req.Amount)
	}

	return types.ResponseDeliverTx{Code: code.OK}
}

// authorized user to obtain credential - writes crypto materials to the chain and removes his funds
func (app *NymApplication) handleCredentialRequest(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.CredentialRequest{}
	if err := proto.Unmarshal(reqb, req); err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkCredentialRequestTx(reqb); checkResult != code.OK {
		app.log.Info("HandleCredentialRequest failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	cryptoMaterialsBytes, err := proto.Marshal(req.CryptoMaterials)
	if err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	// remove funds
	if err := app.decreaseBalanceBy(req.ClientAddress, uint64(req.Value)); err != nil {
		// it's impossible for it to fail as err is only thrown if account does not exist or has insufficient balance
		// and we already checked for that
		app.log.Error(fmt.Sprintf("Undefined behaviour when trying to decrease client's (%v) balance: %v",
			ethcommon.BytesToAddress(req.ClientAddress).Hex(),
			err,
		))
		// TODO: panic or just continue?
	}

	// we need to include slightly more information in the key field in case given user performed
	// more than 1 transfer in given block. That way he wouldn't need to recreate byte materials to index the tx
	key := make([]byte, ethcommon.AddressLength+constants.ECPLen+len(tmconst.CredentialRequestKeyPrefix))
	i := copy(key, tmconst.CredentialRequestKeyPrefix)
	i += copy(key[i:], req.ClientAddress)
	// gamma is unique per credential request;
	// it's client's fault if he intentionally reuses is and is up to him to distinguish correct credentials
	copy(key[i:], req.CryptoMaterials.EgPub.Gamma)
	return types.ResponseDeliverTx{
		Code: code.OK,
		Tags: []cmn.KVPair{
			{Key: key, Value: cryptoMaterialsBytes},
		},
	}
}

func (app *NymApplication) handleDepositCredential(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.DepositCoconutCredentialRequest{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkDepositCoconutCredentialTx(reqb); checkResult != code.OK {
		app.log.Info("handleDepositCredential failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	address := ethcommon.BytesToAddress(req.ProviderAddress)

	if !app.checkIfAccountExists(address[:]) {
		// if it doesn't exist we know the flag is set to create new account on deposit,
		// otherwise checkTx would have failed
		didSucceed := app.createNewAccountOp(address)
		if !didSucceed {
			app.log.Error("Could not create account for the provider")
			return types.ResponseDeliverTx{Code: code.INVALID_MERCHANT_ADDRESS}
		}
		app.log.Debug(fmt.Sprintf("Created new account for %v", address.Hex()))
	}

	cryptoMaterialsBytes, err := proto.Marshal(req.CryptoMaterials)
	if err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	app.log.Debug(
		fmt.Sprintf("Deposit request from address %v, zeta %v", req.ProviderAddress, req.CryptoMaterials.Theta.Zeta),
	)

	app.setZetaStatus(req.CryptoMaterials.Theta.Zeta, tmconst.ZetaStatusBeingVerified)

	key := make([]byte,
		ethcommon.AddressLength+len(req.CryptoMaterials.Theta.Zeta)+len(tmconst.RedeemCredentialRequestKeyPrefix)+8,
	)
	i := copy(key, tmconst.RedeemCredentialRequestKeyPrefix)
	i += copy(key[i:], address[:])
	binary.BigEndian.PutUint64(key[i:], uint64(req.Value))
	i += 8
	copy(key[i:], req.CryptoMaterials.Theta.Zeta)
	return types.ResponseDeliverTx{
		Code: code.OK,
		Tags: []cmn.KVPair{
			// while it is not crucial we have unique keys here, verifiers will need to be able to
			// send a transaction back "confirming" status of this data and this will require an unique key field.
			// So we might as well use the same system already
			// [ Prefix || Provider || uint64(value) || Zeta(g^s) --- required crypto materials ]
			{Key: key, Value: cryptoMaterialsBytes},
		},
	}
}

func (app *NymApplication) handleCredentialVerificationNotification(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.CredentialVerificationNotification{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkCredentialVerificationNotificationTx(reqb); checkResult != code.OK {
		app.log.Info("handleCredentialVerificationNotification failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	// 'accept' the notification
	newCount := app.storeVerifierNotification(req.VerifierPublicKey, req.Zeta, req.Value, req.CredentialValidity)
	zetaB64 := base64.StdEncoding.EncodeToString(req.Zeta)

	app.log.Debug(fmt.Sprintf("Reached %v notifications out of required %v for zeta %v (value %v)",
		newCount,
		app.state.verifierThreshold,
		zetaB64,
		req.Value,
	))

	// commit the transaction if threshold is reached
	if newCount == app.state.verifierThreshold {
		app.log.Debug(fmt.Sprintf("Reached required threshold of %v for %v (value %v)",
			app.state.verifierThreshold,
			zetaB64,
			req.Value,
		))

		// check if account exists
		currentBalance, err := app.retrieveAccountBalance(req.ProviderAddress)
		// It should already exist since the provider had to send a request
		// before to actually request the deposit to happen, but double check it now anyway
		if err != nil && createAccountOnDepositIfDoesntExist {
			didSucceed := app.createNewAccountOp(ethcommon.BytesToAddress(req.ProviderAddress))
			if !didSucceed {
				app.log.Info(fmt.Sprintf("Failed to create new account for the provider with address %v",
					ethcommon.BytesToAddress(req.ProviderAddress).Hex()))
				return types.ResponseDeliverTx{Code: code.UNKNOWN}
			}
		} else if err != nil {
			app.log.Info("Provider's account does not exist and system is not set to create new ones")
			return types.ResponseDeliverTx{Code: code.ACCOUNT_DOES_NOT_EXIST}
		}

		app.setAccountBalance(req.ProviderAddress, currentBalance+uint64(req.Value))
		// mark zeta as 'fully' spent
		// TODO: perhaps do some mark and sweep later on for all credentials set as being verified for a long time
		// and invalidate them?
		app.log.Info(fmt.Sprintf("Marking zeta %v as spent and provider's %v balance was increased by %v",
			zetaB64,
			ethcommon.BytesToAddress(req.ProviderAddress).Hex(),
			req.Value,
		))
		app.setZetaStatus(req.Zeta, tmconst.ZetaStatusSpent, req.ProviderAddress...)
	}

	return types.ResponseDeliverTx{Code: code.OK}
}

func (app *NymApplication) handleTokenRedemption(reqb []byte) types.ResponseDeliverTx {
	req := &transaction.TokenRedemptionRequest{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkTokenRedemptionRequestTx(reqb); checkResult != code.OK {
		app.log.Info("handleTokenRedemption failed checkTx")
		return types.ResponseDeliverTx{Code: checkResult}
	}

	address := ethcommon.BytesToAddress(req.UserAddress)

	// we know user has enough funds, nonce is unique, signature is valid, etc.

	// TODO: do we just 'remove' the funds or somehow just 'lock' them
	// by say creating a temporary account and moving the funds there?
	// But since this solution will not be used in real deployment, the dummy and simpler solution can be used:
	// just remove funds here.

	// remove funds
	if err := app.decreaseBalanceBy(req.UserAddress, req.Amount); err != nil {
		// it's impossible for it to fail as err is only thrown if account does not exist or has insufficient balance
		// and we already checked for that
		app.log.Error(fmt.Sprintf("Undefined behaviour when trying to decrease client's (%v) balance: %v",
			ethcommon.BytesToAddress(req.UserAddress).Hex(),
			err,
		))
		// TODO: panic or just continue?
	}

	key := make([]byte,
		ethcommon.AddressLength+len(tmconst.RedeemTokensRequestKeyPrefix)+8+tmconst.NonceLength,
	)
	i := copy(key, tmconst.RedeemTokensRequestKeyPrefix)
	i += copy(key[i:], address[:])
	binary.BigEndian.PutUint64(key[i:], req.Amount)
	i += 8
	copy(key[i:], req.Nonce)
	return types.ResponseDeliverTx{
		Code: code.OK,
		Tags: []cmn.KVPair{
			// in this dummy implementation we don't really need to attach much information,
			// only just enough to identify this particular transaction because no processing on redeemer side is required
			// [ Prefix || User || Amount || Nonce --- nil? ]
			// TODO: resolve https://github.com/nymtech/nym/issues/7#issue-461004937 and put the data more nicely in here
			{Key: key, Value: nil},
		},
	}
}

func (app *NymApplication) handleTokenRedemptionConfirmationNotification(reqb []byte) types.ResponseDeliverTx {
	// nothing fancy needs to happen here, basically accept notification, keep count and return
	// the total count and threshold
	req := &transaction.TokenRedemptionConfirmationNotification{}

	if err := proto.Unmarshal(reqb, req); err != nil {
		return types.ResponseDeliverTx{Code: code.INVALID_TX_PARAMS}
	}

	if checkResult := app.checkTokenRedemptionConfirmationNotificationTx(reqb); checkResult != code.OK {
		app.log.Info("handleTokenRedemptionConfirmationNotification failed checkTx")
		// it will be thrown if threshold was already reached but that's alright, we only need to ensure only
		// single redeemer sends ethereum tx
		return types.ResponseDeliverTx{Code: checkResult}
	}

	userAddress := ethcommon.BytesToAddress(req.UserAddress)

	// 'accept' the notification
	newCount := app.storeRedeemerNotification(req.RedeemerPublicKey,
		userAddress,
		req.Nonce,
		req.Amount,
	)

	app.log.Debug(fmt.Sprintf("Reached %v notifications out of required %v for: user %v amount %v nonce %v",
		newCount,
		app.state.redeemerThreshold,
		userAddress.Hex(),
		req.Amount,
		req.Nonce,
	))

	// commit the transaction if threshold is reached
	if newCount == app.state.redeemerThreshold {
		app.log.Debug(fmt.Sprintf("Reached required threshold of %v for: user %v amount %v nonce %v",
			app.state.redeemerThreshold,
			userAddress.Hex(),
			req.Amount,
			req.Nonce,
		))

		// TODO: do we need to do anything more here?
	}

	thresholdB := make([]byte, 4)
	binary.BigEndian.PutUint32(thresholdB, app.state.redeemerThreshold)

	countB := make([]byte, 4)
	binary.BigEndian.PutUint32(countB, newCount)

	return types.ResponseDeliverTx{Code: code.OK, Data: append(thresholdB, countB...)}
}
