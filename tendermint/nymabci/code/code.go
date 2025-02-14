// code.go - Nym application return codes
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

// Package code defines return codes for the Nym application
package code

// TODO: reorder and group codes in a more logical way (currently they're added as needed)
// nolint: golint
const (
	// as per spec, codes have to be represented as uint32 and 0 is reserved for OK

	// OK represents a success.
	OK uint32 = 0
	// UNKNOWN represents a failure due to unknown causes.
	UNKNOWN uint32 = 1
	// INVALID_TX_LENGTH represents error due to tx having unexpected length.
	INVALID_TX_LENGTH uint32 = 2
	// INVALID_TX_PARAMS represents error due to tx having incorrect attributes embedded.
	INVALID_TX_PARAMS uint32 = 3
	// INVALID_QUERY_PARAMS represents error due to query having incorrect attributes embedded.
	INVALID_QUERY_PARAMS uint32 = 4
	// ACCOUNT_DOES_NOT_EXIST represents error due to trying to interact with an account that does not exist.
	ACCOUNT_DOES_NOT_EXIST uint32 = 5
	// INSUFFICIENT_BALANCE represents error due to an account having insufficient funds for the desired operation.
	INSUFFICIENT_BALANCE uint32 = 6
	// INVALID_CREDENTIAL represents error due to failing to verify credential.
	INVALID_CREDENTIAL uint32 = 7
	// INVALID_SIGNATURE represents error due to failing to verify signature.
	INVALID_SIGNATURE uint32 = 8
	// INVALID_MERCHANT_ADDRESS represents error due to malformed merchant address.
	INVALID_MERCHANT_ADDRESS uint32 = 9
	// MERCHANT_DOES_NOT_EXIST represents error when trying to spend credential at non-existing merchant.
	// Only applicable if system is set to not create accounts for non-existent merchants.
	MERCHANT_DOES_NOT_EXIST uint32 = 10
	// ISSUING_AUTHORITY_DOES_NOT_EXIST represents error when trying to verify credential/signature with IA that
	// is not known by the abci
	ISSUING_AUTHORITY_DOES_NOT_EXIST uint32 = 11
	// MALFORMED_ADDRESS represents error due to address being malformed (incorrect length, incorrect prefix, etc)
	MALFORMED_ADDRESS uint32 = 12
	// DOUBLE_SPENDING_ATTEMPT represents error due to trying to spend credential with the same sequence number
	DOUBLE_SPENDING_ATTEMPT uint32 = 13
	// SELF_TRANSFER represents error when trying to send funds from account X back to account X
	SELF_TRANSFER uint32 = 14
	// REPLAY_ATTACK_ATTEMPT represents error due to trying to transfer tokens
	// to the pipe account with repeating same nonce.
	REPLAY_ATTACK_ATTEMPT uint32 = 15
	// UNDEFINED_TX represents error due to using tx prefix for an undefined tx.
	UNDEFINED_TX uint32 = 16
	// ETHEREUM_WATCHER_DOES_NOT_EXIST represents error when trying to verify signature signed by an unknown watcher
	ETHEREUM_WATCHER_DOES_NOT_EXIST uint32 = 17
	// ALREADY_CONFIRMED represents error when some entity, like the watcher,
	// sends same event confirmation multiple times
	ALREADY_CONFIRMED uint32 = 18
	// MALFORMED_PUBLIC_KEY represents error when some entity presents a malformed public key, for example by having
	// invalid length or structure (or can't be unmarshalled)
	MALFORMED_PUBLIC_KEY uint32 = 19
	// ALREADY_COMMITTED represents error when watcher wants to notify about transaction
	// while a threshold number of watchers already sent their notifications
	ALREADY_COMMITTED uint32 = 20
	// INVALID_PIPE_ACCOUNT represents error due to using different than specified address of the pipe account
	INVALID_PIPE_ACCOUNT uint32 = 21
	// INVALID_VALUE represents error due to credential request (or possibly transfer) having an invalid value
	INVALID_VALUE uint32 = 22
	// CREDENTIAL_VERIFIER_DOES_NOT_EXIST represents error when trying to verify signature signed by an unknown verifier
	CREDENTIAL_VERIFIER_DOES_NOT_EXIST uint32 = 23
	// INVALID_ZETA_STATUS represents error when trying to register verification notification while zeta was either
	// already spent or not requested to be deposited
	INVALID_ZETA_STATUS uint32 = 24
	// TOKEN_REDEEMER_DOES_NOT_EXIST represents error when trying to verify signature signed by an unknown redeemer
	TOKEN_REDEEMER_DOES_NOT_EXIST uint32 = 25
	// COULD_NOT_TRANSFER represents a generic error for failing to transfer funds between accounts.
	COULD_NOT_TRANSFER uint32 = 100 // todo: replace occurrences with more specific errors
)

// ToString returns string representation of the return code. It is useful for making human-readable responses.
//nolint: gocyclo
func ToString(code uint32) string {
	switch code {
	case OK:
		return "OK"
	case UNKNOWN:
		return "Unknown"
	case INVALID_TX_LENGTH:
		return "Invalid Tx Length"
	case INVALID_TX_PARAMS:
		return "Invalid Tx Params"
	case INVALID_QUERY_PARAMS:
		return "Invalid Query Params"
	case ACCOUNT_DOES_NOT_EXIST:
		return "Account Does Not Exist"
	case INSUFFICIENT_BALANCE:
		return "Insufficient Balance"
	case INVALID_CREDENTIAL:
		return "Invalid Credential"
	case INVALID_SIGNATURE:
		return "Invalid Signature"
	case INVALID_MERCHANT_ADDRESS:
		return "Invalid Merchant Address"
	case MERCHANT_DOES_NOT_EXIST:
		return "Merchant Does Not Exist"
	case ISSUING_AUTHORITY_DOES_NOT_EXIST:
		return "Issuing Authority Does Not Exist"
	case MALFORMED_ADDRESS:
		return "Malformed Address"
	case DOUBLE_SPENDING_ATTEMPT:
		return "Double Spending Attempt"
	case SELF_TRANSFER:
		return "Self Transfer"
	case REPLAY_ATTACK_ATTEMPT:
		return "Replay Attack Attempt"
	case UNDEFINED_TX:
		return "Undefined Tx"
	case COULD_NOT_TRANSFER:
		return "Could Not Perform Transfer"
	case ETHEREUM_WATCHER_DOES_NOT_EXIST:
		return "Ethereum Watcher Does Not Exist"
	case ALREADY_CONFIRMED:
		return "Already Confirmed"
	case MALFORMED_PUBLIC_KEY:
		return "Malformed Public Key"
	case ALREADY_COMMITTED:
		return "Already Committed"
	case INVALID_PIPE_ACCOUNT:
		return "Invalid Pipe Account"
	case INVALID_VALUE:
		return "Invalid Value"
	case CREDENTIAL_VERIFIER_DOES_NOT_EXIST:
		return "Credential Verifier Does Not Exist"
	case INVALID_ZETA_STATUS:
		return "Invalid Zeta Status"
	case TOKEN_REDEEMER_DOES_NOT_EXIST:
		return "Token Redeemer Does Not Exist"
	default:
		return "Unknown Error Code"
	}
}
