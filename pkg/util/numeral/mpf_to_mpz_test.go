// Source: src/util/mpf.cpp:1170-1181; src/util/mpf.h:236-238.
package numeral

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatToMPZIndependentTruncationAndOwners(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	foreign, _ := mpzPublicationDirectOwner(t)
	rng := rand.New(rand.NewChaCha8([32]byte{11, 70, 11, 81}))
	for _, sb := range []uint32{2, 24, 80, 120} {
		for sample := 0; sample < 30; sample++ {
			sig := new(big.Int)
			for bits := uint32(0); bits < sb; bits += 64 {
				sig.Lsh(sig, 64)
				sig.Or(sig, new(big.Int).SetUint64(rng.Uint64()))
			}
			sig.Mod(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb)))
			sig.SetBit(sig, int(sb)-1, 1)
			exp := int64(rng.IntN(301)) - 150
			for _, sign := range []bool{false, true} {
				for _, owner := range []*IntegerManager{m.integer, foreign} {
					for _, alias := range []bool{false, true} {
						if alias && owner != m.integer {
							continue
						}
						var x ArbitraryFloat
						x.format = arbitraryFloatFormat(11, sb, sign)
						x.exponent = exp
						serviceMust(t, m.integer.SetString(&x.significand, sig.String()))
						before := x
						separate := NewInt(7)
						out := &separate
						if alias {
							out = &x.significand
						}
						exact := mpfAddSubDyadic(sig, exp-int64(sb)+1)
						if sign {
							exact.Neg(exact)
						}
						want := new(big.Int).Quo(exact.Num(), exact.Denom())
						serviceMust(t, m.ToMPZ(&x, owner, out))
						got, e := owner.String(out)
						serviceMust(t, e)
						if got != want.String() || !alias && x != before {
							t.Fatal("independent truncation", sb, sample, exp, sign, owner == foreign, alias, got, want)
						}
						serviceMust(t, owner.Del(&separate))
						serviceMust(t, m.Del(&x))
					}
				}
			}
		}
	}
}

func TestArbitraryFloatToMPZSourceBitFieldExponentAndOverflowPrefix(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	// The unsigned 16-bit sbits bit-field promotes to signed int. Source
	// subtraction can overflow before the following +1, even if the final
	// mathematical expression would fit. COPY/sign have already happened.
	x := ArbitraryFloat{format: arbitraryFloatFormat(11, 2, true), exponent: -2147483647, significand: NewInt(6)}
	out := NewInt(7)
	body := m.ToMPZ(&x, m.integer, &out)
	if !errors.Is(body, arena.ErrBounds) || out != NewInt(-6) {
		t.Fatal("signed subtraction after COPY/sign prefix", body, out)
	}
	serviceMust(t, m.CleanupToMPZ())
	// A negative exponent below INT_MIN passes the source's one-sided assertion
	// and narrows to 1. No huge shift should replace the literal signed32 cast.
	x.exponent = -4294967295
	x.significand = NewInt(6)
	serviceMust(t, m.ToMPZ(&x, m.integer, &out))
	if out != NewInt(-6) {
		t.Fatal("literal narrowed exponent", out)
	}
	if !modularSourceDebug {
		x.exponent = 4294967297
		serviceMust(t, m.ToMPZ(&x, m.integer, &out))
		if out != NewInt(-6) {
			t.Fatal("release narrowed exponent", out)
		}
	}
}
