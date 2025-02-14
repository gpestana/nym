// token.go - Nym token definition
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

// Package token defines Nym token structure and associated methods.
package token

import (
	"fmt"

	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
)

// TODO: refactor the entire file? - move somewhere more appropriate and perhaps rename it

// For future reference:
// tags can be accessed via reflections;
// t := reflect.TypeOf(T{})
// f, _ := t.FieldByName("f")
// f.Tag

//nolint: gochecknoglobals
var (
	AllowedValues = []int64{1, 2, 5, 10, 20, 50, 100}
)

type Token struct {
	longtermSecret *Curve.BIG `coconut:"private"`
	sequenceNum    *Curve.BIG `coconut:"private"`
	value          int64      `coconut:"public"` // should be limited to set of possible values to prevent traffic analysis
	// ttl         time.Time  `coconut:"public"`
}

func (t *Token) LongtermSecret() *Curve.BIG {
	return t.longtermSecret
}

func (t *Token) SequenceNum() *Curve.BIG {
	return t.sequenceNum
}

func (t *Token) Value() int64 {
	return t.value
}

func (t *Token) GetPublicAndPrivateSlices() ([]*Curve.BIG, []*Curve.BIG) {
	// first private attribute has to be the sequence number
	// and the first public attribute should be the value
	pubM := make([]*Curve.BIG, 1)
	privM := make([]*Curve.BIG, 2)

	valBig := Curve.NewBIGint(int(t.value))
	// for any additional public attributes (that are not ints), just hash them into BIG:
	// attrBig := utils.HashBytesToBig(amcl.SHA256, attr)

	privM[0] = t.sequenceNum
	privM[1] = t.longtermSecret

	pubM[0] = valBig
	return pubM, privM
}

// // should be associated with given client/user rather than token if I understand it correctly
// type PrivateKey *Curve.BIG

type Credential *coconut.Signature

func (t *Token) PrepareBlindSign(params *coconut.Params, egPub *elgamal.PublicKey) (*coconut.Lambda, error) {
	pubM, privM := t.GetPublicAndPrivateSlices()
	return coconut.PrepareBlindSign(params, egPub, pubM, privM)
}

func ValidateValue(val int64) bool {
	for _, allowed := range AllowedValues {
		if val == allowed {
			return true
		}
	}
	return false
}

// temp, havent decided on where attrs will be generated, but want token instance for test
func New(s, k *Curve.BIG, val int64) (*Token, error) {
	if !ValidateValue(val) {
		return nil, fmt.Errorf("disallowed credential value: %v, allowed: %v", val, AllowedValues)
	}
	// TODO: validate val
	return &Token{
		longtermSecret: k,
		sequenceNum:    s,
		value:          val,
	}, nil
}
