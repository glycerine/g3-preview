// Source: src/util/mpf.cpp:335-374; src/util/mpf.cpp:2043-2068.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

// Independent bounded fractions validate the literal conversion/round switch;
// these are not assertions from the two original SET driver helpers.
func TestArbitraryFloatFormattedSetIndependentFractions(t *testing.T) {
	for _, widths := range [][2]uint32{{8, 9}, {8, 6}, {12, 6}} {
		for _, sign := range []bool{false, true} {
			for mode := ArbitraryFloatNearestEven; mode <= ArbitraryFloatTowardZero; mode++ {
				b := arena.NewBudget(1 << 27)
				m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
				serviceMust(t, e)
				for f := int32(0); f < 64; f++ {
					var x, y ArbitraryFloat
					sig := NewInt(f)
					serviceMust(t, m.SetComponents(&x, 8, widths[0], sign, 0, &sig))
					serviceMust(t, m.SetFormat(&y, 8, widths[1], mode, &x))
					var want int64
					if widths[1] > widths[0] {
						want = int64(f) << uint(widths[1]-widths[0])
					} else {
						shift := widths[0] - widths[1]
						den := int64(1) << shift
						want = int64(f) / den
						rem := int64(f) % den
						inc := false
						switch mode {
						case ArbitraryFloatNearestEven:
							inc = 2*rem > den || 2*rem == den && want&1 != 0
						case ArbitraryFloatNearestAway:
							inc = 2*rem >= den
						case ArbitraryFloatTowardPositive:
							inc = !sign && rem != 0
						case ArbitraryFloatTowardNegative:
							inc = sign && rem != 0
						}
						if inc {
							want++
						}
					}
					got, e := m.integer.Int64(&y.significand)
					serviceMust(t, e)
					if got != want || y.Sign() != sign || y.exponent != 0 {
						t.Fatal(widths, sign, mode, f, got, want, y.exponent)
					}
					serviceMust(t, m.Del(&x))
					serviceMust(t, m.Del(&y))
				}
				if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
					t.Fatal(e)
				}
				serviceMust(t, m.Close())
				if b.Used() != 0 {
					t.Fatal(b.Used())
				}
			}
		}
	}
}
func TestArbitraryFloatFormattedSetLiteralSelfAlias(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	serviceMust(t, e)
	var x ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&x, 8, 24, false, 0, &one))
	// x.sbits is read after o.sbits publication. This gives ds=3 rather than
	// the nonalias ds=9; literal result is six exponents below normalized1.
	serviceMust(t, m.SetFormat(&x, 8, 30, ArbitraryFloatNearestEven, &x))
	got, e := m.integer.Int64(&x.significand)
	serviceMust(t, e)
	if x.SBits() != 30 || got != 64 || x.exponent != -6 {
		t.Fatal("literal self-alias read order", got, x.exponent)
	}
	serviceMust(t, m.Del(&x))
	if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
		t.Fatal(e)
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
