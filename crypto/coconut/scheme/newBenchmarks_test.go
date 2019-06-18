package coconut_test

import (
	"fmt"
	math "math"
	"testing"

	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	. "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
	// . "github.com/nymtech/nym/crypto/testutils"
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
		_, err := ShowBlindSignature(params, vk, sig, privs)
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
		blindShowMats, _ := ShowBlindSignature(params, vk, sig, privs)

		b.StartTimer()
		isValid := BlindVerify(params, vk, sig, blindShowMats, pubs)
		if !isValid {
			panic(isValid)
		}
	}
}
