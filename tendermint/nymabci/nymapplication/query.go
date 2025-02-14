// query.go - Query-related logic for Tendermint ABCI for Nym
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
	"fmt"

	"github.com/nymtech/nym/tendermint/nymabci/code"
	tmconst "github.com/nymtech/nym/tendermint/nymabci/constants"
	"github.com/tendermint/tendermint/abci/types"
)

func (app *NymApplication) checkAccountBalanceQuery(req types.RequestQuery) types.ResponseQuery {
	val, err := app.retrieveAccountBalance(req.Data)
	if err != nil {
		return types.ResponseQuery{Code: code.ACCOUNT_DOES_NOT_EXIST}
	}
	return types.ResponseQuery{Code: code.OK, Key: req.Data, Value: balanceToBytes(val)}
}

// DEPRECATED: Use queryCheckZetaStatus instead
func (app *NymApplication) checkZeta(req types.RequestQuery) types.ResponseQuery {
	isSpent := bytes.HasPrefix(app.checkZetaStatus(req.Data), tmconst.ZetaStatusSpent.DbEntry())
	app.log.Debug(fmt.Sprintf("Zeta %v is spent: %v", req.Data, isSpent))
	isSpentB := []byte{0}
	if isSpent {
		isSpentB = []byte{1}
	}
	return types.ResponseQuery{Code: code.OK, Key: req.Data, Value: isSpentB}
}

func (app *NymApplication) queryCheckZetaStatus(req types.RequestQuery) types.ResponseQuery {
	status := app.checkZetaStatus(req.Data)
	app.log.Debug(fmt.Sprintf("Zeta %v status: %v", req.Data, status))
	return types.ResponseQuery{Code: code.OK, Key: req.Data, Value: status}
}

//nolint: unparam
func (app *NymApplication) printVk(req types.RequestQuery) (types.ResponseQuery, error) {
	if !tmconst.DebugMode {
		app.log.Info("Trying to use printVk not in debug mode")
		return types.ResponseQuery{}, tmconst.ErrNotInDebug
	}
	avk, err := app.retrieveAggregateVerificationKey()
	if err != nil {
		return types.ResponseQuery{Code: code.UNKNOWN}, err
	}
	fmt.Println(avk)
	return types.ResponseQuery{Code: code.OK}, nil
}
