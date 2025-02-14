// utils_test.go - tests of auxiliary functions
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

package utils_test

import (
	"bytes"
	"testing"

	"github.com/nymtech/nym/common/utils"
	"github.com/nymtech/nym/constants"
	"github.com/nymtech/nym/crypto/bpgroup"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/stretchr/testify/assert"
)

func TestCompressECPBytes(t *testing.T) {
	bpgroup := bpgroup.New() // for easy access to rng

	for i := 0; i < 200; i++ {
		x := Curve.Randomnum(bpgroup.Order(), bpgroup.Rng())

		g := Curve.G1mul(bpgroup.Gen1(), x)

		uncompressed := make([]byte, constants.ECPLenUC)
		target := make([]byte, constants.ECPLen)

		g.ToBytes(target, true)
		g.ToBytes(uncompressed, false)

		compressed, err := utils.CompressECPBytes(uncompressed)
		assert.Nil(t, err)

		assert.True(t, bytes.Equal(target, compressed))
	}

	compRes, err := utils.CompressECPBytes(([]byte)(nil))
	assert.Nil(t, compRes)
	assert.Error(t, err)

	x := Curve.Randomnum(bpgroup.Order(), bpgroup.Rng())
	g := Curve.G1mul(bpgroup.Gen1(), x)

	uncompressed := make([]byte, constants.ECPLenUC)
	compressed := make([]byte, constants.ECPLen)

	g.ToBytes(uncompressed, false)
	g.ToBytes(compressed, true)

	compRes, err = utils.CompressECPBytes(compressed)
	assert.Nil(t, compRes)
	assert.Error(t, err)

	uncompressed[0] = 0x01
	compRes, err = utils.CompressECPBytes(uncompressed)
	assert.Nil(t, compRes)
	assert.Error(t, err)

	// restore original prefix
	uncompressed[0] = 0x04
	uncompressed = uncompressed[:len(uncompressed)-1]
	compRes, err = utils.CompressECPBytes(uncompressed)
	assert.Nil(t, compRes)
	assert.Error(t, err)
}

// NOTE THAT BELOW BENCHMARKS ARE ONLY USED FOR COMPARISON WITH EACH OTHER
// AS 'TRUE' RESULTS ARE BIASED BY THE G1MUL (INTRODUCING STOP AND START TIMER
// CAUSES THE ENTIRE BENCHMARK TO HANG INDEFINITELY)

func BenchmarkCompressECPBytes(b *testing.B) {
	bpgroup := bpgroup.New() // for easy access to rng
	for i := 0; i < b.N; i++ {
		x := Curve.Randomnum(bpgroup.Order(), bpgroup.Rng())
		g := Curve.G1mul(bpgroup.Gen1(), x)

		uncompressed := make([]byte, constants.ECPLenUC)
		g.ToBytes(uncompressed, false)

		_, err := utils.CompressECPBytes(uncompressed)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkNativeCompressECPBytes(b *testing.B) {
	bpgroup := bpgroup.New() // for easy access to rng
	for i := 0; i < b.N; i++ {
		x := Curve.Randomnum(bpgroup.Order(), bpgroup.Rng())
		g := Curve.G1mul(bpgroup.Gen1(), x)

		uncompressed := make([]byte, constants.ECPLenUC)
		g.ToBytes(uncompressed, false)

		comp := make([]byte, constants.ECPLen)
		gr := Curve.ECP_fromBytes(uncompressed)
		gr.ToBytes(comp, true)
	}
}
