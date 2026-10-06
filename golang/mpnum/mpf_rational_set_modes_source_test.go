// Source: src/util/mpf.cpp:233-300,2043-2068.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// These bounded dyadic expectations are independent checks, not assertions
// from the two registered SET driver helpers.
func TestArbitraryFloatExponentRationalSetIndependentModes(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for mode := ArbitraryFloatNearestEven; mode <= ArbitraryFloatTowardZero; mode++ {
			b := arena.NewBudget(1 << 28)
			m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
			serviceMust(t, e)
			q := NewRational()
			var out ArbitraryFloat
			ex := NewInt(0)
			for n := int64(1); n <= 63; n++ {
				signed := n
				if negative {
					signed = -signed
				}
				serviceMust(t, m.rational.SetFraction(&q, signed, 16))
				serviceMust(t, m.SetExponentRational(&out, 8, 4, mode, &ex, &q))
				k := uint(0)
				for int64(1)<<(k+1) <= n {
					k++
				}
				den := int64(1)
				if k > 3 {
					den = 1 << (k - 3)
				}
				integer, rem := n/den, n%den
				inc := false
				switch mode {
				case ArbitraryFloatNearestEven:
					inc = 2*rem > den || 2*rem == den && integer&1 != 0
				case ArbitraryFloatNearestAway:
					inc = 2*rem >= den
				case ArbitraryFloatTowardPositive:
					inc = !negative && rem != 0
				case ArbitraryFloatTowardNegative:
					inc = negative && rem != 0
				}
				if inc {
					integer++
				}
				want := float64(integer*den) / 16
				if negative {
					want = -want
				}
				got, e := m.ToDouble(&out)
				serviceMust(t, e)
				if got != want {
					t.Fatal("independent rounding", negative, mode, n, got, want)
				}
			}
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.rational.Del(&q))
			serviceMust(t, m.SourceFinalize())
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(b.Used())
			}
		}
	}
}
