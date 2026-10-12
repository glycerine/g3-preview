package numeral

import (
	"math/big"
	"testing"
)

// The polynomial division and factorization callers use submul with its
// destination aliased to an input. Compare both fused routes against an
// independent big.Int calculation across all destination alias positions.
func TestIntegerModularFusedLargeAliasOracle(t *testing.T) {
	operands := [][3]string{
		{"18446744073709551617", "-4294967299", "9223372036854775811"},
		{"-123456789012345678901234567890", "98765432109876543210", "-99999999999999999999"},
		{"0", "-18446744073709551617", "18446744073709551619"},
	}
	for _, modulus := range []uint64{0, 7, 4294967291} {
		m, z, _ := modularManager(t, modulus)
		var p *big.Int
		if modulus != 0 {
			p = new(big.Int).SetUint64(modulus)
		}
		for _, values := range operands {
			original := [3]*big.Int{}
			for i, spelling := range values {
				original[i], _ = new(big.Int).SetString(spelling, 10)
				if p != nil {
					original[i].Mod(original[i], p)
					if original[i].Cmp(new(big.Int).Rsh(new(big.Int).Set(p), 1)) > 0 {
						original[i].Sub(original[i], p)
					}
				}
			}
			for _, subtract := range []bool{false, true} {
				for output := 0; output < 4; output++ {
					var got [4]Int
					for i := 0; i < 3; i++ {
						if e := m.SetString(&got[i], values[i]); e != nil {
							t.Fatal(modulus, values, e)
						}
					}
					product := new(big.Int).Mul(original[1], original[2])
					want := new(big.Int).Set(original[0])
					if subtract {
						want.Sub(want, product)
					} else {
						want.Add(want, product)
					}
					if p != nil {
						want.Mod(want, p)
						if want.Cmp(new(big.Int).Rsh(new(big.Int).Set(p), 1)) > 0 {
							want.Sub(want, p)
						}
					}
					var e error
					if subtract {
						e = m.SubMul(&got[0], &got[1], &got[2], &got[output])
					} else {
						e = m.AddMul(&got[0], &got[1], &got[2], &got[output])
					}
					if e != nil {
						t.Fatal(modulus, values, subtract, output, e)
					}
					spelling, e := z.String(&got[output])
					if e != nil || spelling != want.String() {
						t.Fatalf("modulus=%d values=%v subtract=%v output=%d: got %q/%v, want %s", modulus, values, subtract, output, spelling, e, want)
					}
					for i := 0; i < 3; i++ {
						if i == output {
							continue
						}
						spelling, e = z.String(&got[i])
						if e != nil || spelling != original[i].String() {
							t.Fatalf("modulus=%d values=%v subtract=%v output=%d: input %d changed to %q/%v, want %s", modulus, values, subtract, output, i, spelling, e, original[i])
						}
					}
					for i := range got {
						if e := m.Del(&got[i]); e != nil {
							t.Fatal(modulus, values, subtract, output, "cleanup", i, e)
						}
					}
				}
			}
		}
	}
}
