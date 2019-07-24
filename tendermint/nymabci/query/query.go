// query.go - query logic
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

// Package query defines query logic for the Nym application.
package query

//nolint: golint
const (
	QueryCheckBalancePath = "/balance"
	DEBUG_printVk         = "/DEBUG_PRINTVK"
	// only spent/unspent
	ZetaStatus = "/ZetaStatus"
	// unspent/being verified/spent(+ who was credited)
	FullZetaStatus   = "/FullZetaStatus"
	AccountExistence = "/accountExists"
)

// TODO: do similar thing for other responses
type AccountStatus []byte

var (
	AccountStatusDoesNotExists AccountStatus = []byte{0}
	AccountStatusExists        AccountStatus = []byte{1}
)
