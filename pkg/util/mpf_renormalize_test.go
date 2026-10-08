// Source: src/util/mpf.cpp:1255-1270.
package util

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatRenormalizeIndependentBitLengthOracle(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	z := m.integer
	rng := rand.New(rand.NewChaCha8([32]byte{12, 55, 12, 70}))
	for _, sb := range []uint32{1, 2, 24, 80, 120} {
		for _, length := range []uint32{1, 2, 31, 32, 33, 79, 121, 240} {
			for sample := 0; sample < 8; sample++ {
				v := new(big.Int)
				for bits := uint32(0); bits < length; bits += 64 {
					v.Lsh(v, 64)
					v.Or(v, new(big.Int).SetUint64(rng.Uint64()))
				}
				v.Mod(v, new(big.Int).Lsh(big.NewInt(1), uint(length)))
				v.SetBit(v, int(length)-1, 1)
				want := new(big.Int).Set(v)
				delta := int(length) - int(sb)
				if delta >= 0 {
					want.Rsh(want, uint(delta))
				} else {
					want.Lsh(want, uint(-delta))
				}
				var sig Int
				serviceMust(t, z.SetString(&sig, v.String()))
				exp := int64(sample - 4)
				wantExp := exp + int64(delta)
				serviceMust(t, m.Renormalize(8, sb, &exp, &sig))
				got, e := z.String(&sig)
				serviceMust(t, e)
				if got != want.String() || exp != wantExp {
					t.Fatal("independent bit-length normalization", sb, length, sample, got, want, exp, wantExp)
				}
				serviceMust(t, z.Del(&sig))
			}
		}
	}
}

func TestArbitraryFloatRenormalizeZeroAndWarmInlineNoReservation(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	zero := NewInt(0)
	exp := int64(-19)
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	serviceMust(t, m.Renormalize(8, 120, &exp, &zero))
	if zero != NewInt(0) || exp != -19 || b.Used() != used || r.State() != state {
		t.Fatal("zero touched caches or scalar fields")
	}
	b.FailAfter(-1)
	sig := NewInt(1)
	exp = 0
	serviceMust(t, m.Renormalize(8, 24, &exp, &sig))
	used, state = b.Used(), r.State()
	b.FailAfter(0)
	sig = NewInt(1)
	exp = 0
	serviceMust(t, m.Renormalize(8, 24, &exp, &sig))
	if sig != NewInt(1<<23) || exp != -23 || b.Used() != used || r.State() != state {
		t.Fatal("warm inline path allocated")
	}
	b.FailAfter(-1)
}
