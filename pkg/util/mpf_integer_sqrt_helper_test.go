// Source: src/util/mpf.cpp:951-1003.
package util

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestMyMPZSqrtIndependentCeilingAndAlias(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	z := m.integer
	rng := rand.New(rand.NewChaCha8([32]byte{95, 1, 100, 3}))
	for _, sb := range []uint32{2, 24, 120} {
		for _, odd := range []bool{false, true} {
			for sample := 0; sample < 16; sample++ {
				sig := new(big.Int).SetUint64(rng.Uint64())
				sig.Mod(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				sig.Add(sig, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
				extra := uint32(7)
				if odd {
					extra = 6
				}
				input := new(big.Int).Lsh(sig, uint(sb+extra))
				want := new(big.Int).Sqrt(input)
				if new(big.Int).Mul(want, want).Cmp(input) != 0 {
					want.Add(want, big.NewInt(1))
				}
				for _, alias := range []bool{false, true} {
					var in, separate Int
					serviceMust(t, z.SetString(&in, input.String()))
					old := in
					out := &separate
					if alias {
						out = &in
					}
					serviceMust(t, MyMPZSqrt(z, sb, odd, &in, out))
					got, e := z.String(out)
					serviceMust(t, e)
					if got != want.String() || !alias && in != old {
						t.Fatal("independent ceiling root", sb, odd, sample, alias, got, want)
					}
					serviceMust(t, z.Del(&separate))
					serviceMust(t, z.Del(&in))
				}
			}
		}
	}
}

func TestMyMPZSqrtLiteralEqualityAndEndpointsOnBareInteger(t *testing.T) {
	m, _ := mpzPublicationDirectOwner(t)
	if m.MpfParent != nil {
		t.Fatal("fixture unexpectedly depends on an MPF manager")
	}
	for _, odd := range []bool{false, true} {
		shift := uint(3)
		if odd {
			shift = 4
		}
		var input, out Int
		serviceMust(t, m.SetInt32(&input, int32(32>>shift)))
		serviceMust(t, MyMPZSqrt(m, 4, odd, &input, &out))
		got, e := m.Int64(&out)
		serviceMust(t, e)
		if got != 32 {
			t.Fatal("literal lower==upper early SET", odd, got)
		}
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.Del(&out))
	}
	for _, v := range []int32{1024, 1025, 1521} {
		input, out := NewInt(v), NewInt(7)
		serviceMust(t, MyMPZSqrt(m, 4, false, &input, &out))
		actual, e := m.Int64(&out)
		serviceMust(t, e)
		want := int64(32)
		if v == 1025 {
			want = 33
		}
		if v == 1521 {
			want = 39
		}
		if actual != want || input != NewInt(v) {
			t.Fatal("literal lower/upper/mid endpoint", v, actual, want)
		}
		serviceMust(t, m.Del(&out))
	}
}
