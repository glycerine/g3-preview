// Source: algebraic_numbers.cpp:198-200,533-536; basic_interval.h:94-97.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicCopyIntervalDeletionReleasedNUMSourceGuard(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		for _, upper := range []bool{false, true} {
			name := "same/"
			if foreign {
				name = "foreign/"
			}
			if upper {
				name += "upper"
			} else {
				name += "lower"
			}
			t.Run(name, func(t *testing.T) { algebraicCopyIntervalDeletionCase(t, foreign, upper) })
		}
	}
}
func algebraicCopyIntervalDeletionCase(t *testing.T, foreign, upper bool) {
	c, _, rt := algebraicNegTLSFixture(t, nil)
	source := c
	if foreign {
		source, _, _ = algebraicCompareFixtureProfile(t, true)
	}
	src := algebraicTinyBinaryBoundsRoot(t, source)
	var dst Algebraic
	if upper {
		dst = algebraicCopyIntervalDeletionUpperFixture(t, c)
	} else {
		dst = algebraicBigBinaryBoundsRoot(t, c)
	}
	original := dst
	l, e := c.location(&dst)
	serviceMust(t, e)
	record, e := c.load(l)
	serviceMust(t, e)
	bound := record.interval.BorrowLower()
	if upper {
		bound = record.interval.BorrowUpper()
	}
	token := bound.BorrowNumerator().NativeCellToken()
	lane := algebraicCellDeletionTLSLane(t, rt)
	before := rt.Stats().LiveBlocks
	body := c.CopyFromCore(&dst, source, &src)
	if !errors.Is(body, arena.ErrBounds) || dst != original || rt.Stats().LiveBlocks+1 != before {
		t.Fatal("copy missed old bound release", body, dst)
	}
	record, e = c.load(l)
	serviceMust(t, e)
	if record.poly.slot != 0 || record.size != 0 {
		t.Fatal("copy deleted interval before polynomial")
	}
	serviceMust(t, c.CleanupRetained())
	z := c.config.Rationals.Integers()
	used, stats := z.Budget().Used(), rt.Stats()
	if e := c.CopyFromCore(&dst, source, &src); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Error("copy acknowledged source-destroyed bound", e)
	}
	if dst != original || used != z.Budget().Used() || stats != rt.Stats() {
		t.Error("repeated copy crossed old bound source guard")
	}
	serviceMust(t, c.CleanupRetained())
	contextTLSFinish(t, rt, func() error { return c.CleanupValue(&dst) })
	if dst != (Algebraic{}) || z.IsLiveCellToken(token) {
		t.Fatal("owning cleanup retained stopped copy")
	}
	if foreign {
		serviceMust(t, source.CleanupValue(&src))
	} else {
		contextTLSFinish(t, rt, func() error { return source.CleanupValue(&src) })
	}
	algebraicCellDeletionFinishLane(t, rt, &lane)
}

func algebraicCopyIntervalDeletionUpperFixture(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p, z, b := c.config.Polynomials, c.config.Rationals.Integers(), c.config.Binary
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	view, e := p.wholeView(poly)
	serviceMust(t, e)
	lower := unumeral.NewBinaryInt(1)
	var upper unumeral.BinaryRational
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&n, "26087635650665564425"))
	serviceMust(t, b.SetIntegerPower(&upper, &n, 64))
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, view, &lower, &upper, true))
	serviceMust(t, b.Del(&upper))
	serviceMust(t, z.Del(&n))
	serviceMust(t, p.DelPolynomial(&poly))
	return root
}
