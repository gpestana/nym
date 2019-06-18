// newBenchmarks_test.go - Nym system crypto-related benchmarkss
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
package coconut_test

import (
	"fmt"
	math "math"
	"testing"

	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/crypto/coconut/concurrency/coconutworker"
	"github.com/nymtech/nym/crypto/coconut/concurrency/jobqueue"
	"github.com/nymtech/nym/crypto/coconut/concurrency/jobworker"
	. "github.com/nymtech/nym/crypto/coconut/scheme"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
	"github.com/nymtech/nym/logger"
)

func BenchmarkTTPKeygen(b *testing.B) {
	q := 5
	ias := []int{3, 5, 10, 20, 50, 100}
	for _, n := range ias {
		t := int(math.Round(float64(n) * 2 / 3))
		b.Run(fmt.Sprintf("q=%d/threshold=%d/IAs=%d", q, t, n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				params, _ := Setup(q)
				b.StartTimer()
				_, _, err := TTPKeygen(params, t, n)
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func BenchmarkTTPKeygenConcurrent(b *testing.B) {
	numWorkers := 4
	jobqueue := jobqueue.New()

	params, err := coconut.Setup(5)
	if err != nil {
		panic(err)
	}

	workers := make([]*jobworker.JobWorker, numWorkers)
	ccw := coconutworker.New(jobqueue.In(), params)

	// log needs to be a non-nil, but just void whatever is to be logged
	logger, err := logger.New("", "CRITICAL", true)
	if err != nil {
		panic(err)
	}

	for i := 0; i < numWorkers; i++ {
		workers[i] = jobworker.New(jobqueue.Out(), uint64(i), logger)
	}

	q := 5
	ias := []int{3, 5, 10, 20, 50, 100}
	for _, n := range ias {
		t := int(math.Round(float64(n) * 2 / 3))
		b.Run(fmt.Sprintf("q=%d/threshold=%d/IAs=%d", q, t, n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				b.StartTimer()
				_, _, err := ccw.TTPKeygenWrapper(t, n)
				if err != nil {
					panic(err)
				}
			}
		})
	}

}

var aggregateRes *Signature

func BenchmarkUnblindAndAggregate(b *testing.B) {
	// since unblind and aggregation takes constant time in relation to number of attributes,
	// there is no point in embedding variable number of them into a credential
	ias := []int{3, 5, 10, 20, 50, 100}
	for _, n := range ias {
		t := int(math.Round(float64(n) * 2 / 3))

		b.Run(fmt.Sprintf("threshold=%d/IAs=%d", t, n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				params, _ := Setup(1)
				p, rng := params.P(), params.G.Rng()

				privs := []*Curve.BIG{Curve.Randomnum(p, rng)}
				pubs := []*Curve.BIG{}

				egPriv, egPub := elgamal.Keygen(params.G)
				blindSignMats, _ := PrepareBlindSign(params, egPub, pubs, privs)

				blindSigs := make([]*BlindedSignature, t)
				// we need to generate keys and sign all messages to 'simulate' IAs
				sks, _, _ := TTPKeygen(params, t, n)
				xs := make([]*Curve.BIG, t)
				for i, sk := range sks {
					if i == t {
						break
					}
					sig, _ := BlindSign(params, sk.SecretKey, blindSignMats, egPub, pubs)
					blindSigs[i] = sig
					xs[i] = Curve.NewBIGint(int(sk.ID()))
				}
				pp := NewPP(xs)
				b.StartTimer()
				sigs := make([]*Signature, t)
				for i, bsig := range blindSigs {
					if i == t {
						break
					}
					sigs[i] = Unblind(params, bsig, egPriv)
				}
				aggregateRes = AggregateSignatures(params, sigs, pp)
			}
		})
	}
}

func BenchmarkUnblindAndAggregateConcurrent(b *testing.B) {
	numWorkers := 4
	jobqueue := jobqueue.New()

	params, err := coconut.Setup(5)
	if err != nil {
		panic(err)
	}

	workers := make([]*jobworker.JobWorker, numWorkers)
	ccw := coconutworker.New(jobqueue.In(), params)

	// log needs to be a non-nil, but just void whatever is to be logged
	logger, err := logger.New("", "CRITICAL", true)
	if err != nil {
		panic(err)
	}

	for i := 0; i < numWorkers; i++ {
		workers[i] = jobworker.New(jobqueue.Out(), uint64(i), logger)
	}

	ias := []int{3, 5, 10, 20, 50, 100}
	for _, n := range ias {
		t := int(math.Round(float64(n) * 2 / 3))

		b.Run(fmt.Sprintf("threshold=%d/IAs=%d", t, n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()

				p, rng := params.P(), params.G.Rng()

				privs := []*Curve.BIG{Curve.Randomnum(p, rng)}
				pubs := []*Curve.BIG{}

				egPriv, egPub := ccw.ElGamalKeygenWrapper()
				blindSignMats, _ := ccw.PrepareBlindSignWrapper(egPub, pubs, privs)

				blindSigs := make([]*BlindedSignature, t)
				// we need to generate keys and sign all messages to 'simulate' IAs
				sks, _, _ := ccw.TTPKeygenWrapper(t, n)
				xs := make([]*Curve.BIG, t)
				for i, sk := range sks {
					if i == t {
						break
					}
					sig, _ := ccw.BlindSignWrapper(sk.SecretKey, blindSignMats, egPub, pubs)
					blindSigs[i] = sig
					xs[i] = Curve.NewBIGint(int(sk.ID()))
				}
				pp := NewPP(xs)
				b.StartTimer()
				sigs := make([]*Signature, t)
				for i, bsig := range blindSigs {
					if i == t {
						break
					}
					sigs[i] = ccw.UnblindWrapper(bsig, egPriv)
				}
				aggregateRes = ccw.AggregateSignaturesWrapper(sigs, pp)
			}
		})
	}
}

func BenchmarkShowBlindSignature(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		params, _ := Setup(5)
		p, rng := params.P(), params.G.Rng()

		privs := []*Curve.BIG{Curve.Randomnum(p, rng), Curve.Randomnum(p, rng)}
		pubs := []*Curve.BIG{Curve.Randomnum(p, rng)}

		egPriv, egPub := elgamal.Keygen(params.G)
		blindSignMats, _ := PrepareBlindSign(params, egPub, pubs, privs)

		sk, vk, _ := Keygen(params)
		blindSig, _ := BlindSign(params, sk, blindSignMats, egPub, pubs)
		sig := Unblind(params, blindSig, egPriv)
		b.StartTimer()
		_, err := ShowBlindSignatureTumbler(params, vk, sig, privs, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkShowBlindSignatureConcurrent(b *testing.B) {
	numWorkers := 4
	jobqueue := jobqueue.New()

	params, err := coconut.Setup(5)
	if err != nil {
		panic(err)
	}

	workers := make([]*jobworker.JobWorker, numWorkers)
	ccw := coconutworker.New(jobqueue.In(), params)

	// log needs to be a non-nil, but just void whatever is to be logged
	logger, err := logger.New("", "CRITICAL", true)
	if err != nil {
		panic(err)
	}

	for i := 0; i < numWorkers; i++ {
		workers[i] = jobworker.New(jobqueue.Out(), uint64(i), logger)
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p, rng := params.P(), params.G.Rng()

		privs := []*Curve.BIG{Curve.Randomnum(p, rng), Curve.Randomnum(p, rng)}
		pubs := []*Curve.BIG{Curve.Randomnum(p, rng)}

		egPriv, egPub := ccw.ElGamalKeygenWrapper()
		blindSignMats, _ := ccw.PrepareBlindSignWrapper(egPub, pubs, privs)

		sk, vk, _ := Keygen(params)
		blindSig, _ := ccw.BlindSignWrapper(sk, blindSignMats, egPub, pubs)
		sig := ccw.UnblindWrapper(blindSig, egPriv)
		b.StartTimer()
		_, err := ccw.ShowBlindSignatureWrapper(vk, sig, privs)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkBlindVerify(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		params, _ := Setup(5)
		p, rng := params.P(), params.G.Rng()

		privs := []*Curve.BIG{Curve.Randomnum(p, rng), Curve.Randomnum(p, rng)}
		pubs := []*Curve.BIG{Curve.Randomnum(p, rng)}

		egPriv, egPub := elgamal.Keygen(params.G)
		blindSignMats, _ := PrepareBlindSign(params, egPub, pubs, privs)

		sk, vk, _ := Keygen(params)
		blindSig, _ := BlindSign(params, sk, blindSignMats, egPub, pubs)
		sig := Unblind(params, blindSig, egPriv)
		blindShowMats, _ := ShowBlindSignatureTumbler(params, vk, sig, privs, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

		b.StartTimer()
		isValid := BlindVerifyTumbler(params, vk, sig, blindShowMats, pubs, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		if !isValid {
			panic(isValid)
		}
	}
}

func BenchmarkBlindVerifyConcurrent(b *testing.B) {
	numWorkers := 4
	jobqueue := jobqueue.New()

	params, err := coconut.Setup(5)
	if err != nil {
		panic(err)
	}

	workers := make([]*jobworker.JobWorker, numWorkers)
	ccw := coconutworker.New(jobqueue.In(), params)

	// log needs to be a non-nil, but just void whatever is to be logged
	logger, err := logger.New("", "CRITICAL", true)
	if err != nil {
		panic(err)
	}

	for i := 0; i < numWorkers; i++ {
		workers[i] = jobworker.New(jobqueue.Out(), uint64(i), logger)
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		p, rng := params.P(), params.G.Rng()

		privs := []*Curve.BIG{Curve.Randomnum(p, rng), Curve.Randomnum(p, rng)}
		pubs := []*Curve.BIG{Curve.Randomnum(p, rng)}

		egPriv, egPub := ccw.ElGamalKeygenWrapper()
		blindSignMats, _ := ccw.PrepareBlindSignWrapper(egPub, pubs, privs)

		sk, vk, _ := Keygen(params)
		blindSig, _ := ccw.BlindSignWrapper(sk, blindSignMats, egPub, pubs)
		sig := ccw.UnblindWrapper(blindSig, egPriv)
		// blindShowMats, _ := ccw.ShowBlindSignatureWrapper(vk, sig, privs)
		blindShowMatsTumbler, _ := ccw.ShowBlindSignatureTumblerWrapper(vk, sig, privs, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		b.StartTimer()
		// isValid := ccw.BlindVerifyWrapper(vk, sig, blindShowMats, pubs)
		isValid := ccw.BlindVerifyTumblerWrapper(vk, sig, blindShowMatsTumbler, pubs, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
		if !isValid {
			panic(isValid)
		}
	}
}
