package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func algebraicTinyBinaryBoundsRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p, z := c.config.Polynomials, c.config.Rationals.Integers()
	leading := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&leading, "170141183460469231731687303715884105728"))
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-1), unumeral.NewInt(0), leading}))
	view, e := p.wholeView(poly)
	serviceMust(t, e)
	lower, upper := unumeral.NewBinaryIntPower(1, 64), unumeral.NewBinaryIntPower(1, 63)
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, view, &lower, &upper, true))
	serviceMust(t, p.DelPolynomial(&poly))
	serviceMust(t, z.Del(&leading))
	return root
}

func TestAlgebraicBoundPowerActualEntryCleanupRetry(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(map[bool]string{false: "lower", true: "upper"}[upper], func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			root := algebraicTinyBinaryBoundsRoot(t, c)
			q := c.config.Rationals
			out := unumeral.NewRational()
			get := func() error {
				if upper {
					return c.GetUpperRational(&root, &out)
				}
				return c.GetLowerRational(&root, &out)
			}
			pad := algebraicCellSourceTLSBoundary(t, c)
			body := get()
			if body == nil || !c.bounds.power.pending(c) {
				t.Fatal("missing actual failed power entry", body)
			}
			if e := get(); !errors.Is(e, ErrAlgebraicPending) {
				t.Fatal("pending power replaced", e)
			}
			runtime := q.Integers().RuntimeOwner()
			serviceMust(t, runtime.SetMaxSize(0))
			serviceMust(t, runtime.Deallocate(&pad))
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, q.Del(&out))
			out = unumeral.NewRational()
			serviceMust(t, get())
			want := unumeral.NewRational()
			text := "1/18446744073709551616"
			if upper {
				text = "1/9223372036854775808"
			}
			serviceMust(t, q.SetString(&want, text))
			equal, e := q.Equal(&out, &want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("retry bound changed", text)
			}
			serviceMust(t, q.Del(&want))
			serviceMust(t, q.Del(&out))
			serviceMust(t, c.Del(&root))
		})
	}
}

func TestAlgebraicBoundPowerPreservesOlderAllocation(t *testing.T) {
	c, _ := algebraicCellSourceFixtureProfile(t, true)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	q := c.config.Rationals
	z := q.Integers()
	runtime := z.RuntimeOwner()
	pad := algebraicCellSourceTLSBoundary(t, c)
	var older unumeral.IntegerCopyAllocationReceipt
	destination := unumeral.NewInt(0)
	body := z.CopyFromOwnerWithReceipt(&destination, z, &z.Two64, &older)
	if body == nil || !z.MatchesCopyAllocationInvocation(&older) {
		t.Fatal("missing older allocation", body)
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, runtime.Deallocate(&pad))
	out := unumeral.NewRational()
	if e := c.GetLowerRational(&root, &out); e == nil {
		t.Fatal("power ignored older allocation")
	}
	state, used, stats := z.CopyAllocation, z.Budget().Used(), runtime.Stats()
	serviceMust(t, c.CleanupRetained())
	if !z.MatchesCopyAllocationInvocation(&older) || state != z.CopyAllocation || used != z.Budget().Used() || stats != runtime.Stats() {
		t.Fatal("bound cleanup consumed older allocation")
	}
	serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
	serviceMust(t, z.Del(&destination))
	serviceMust(t, q.Del(&out))
	out = unumeral.NewRational()
	serviceMust(t, c.GetLowerRational(&root, &out))
	serviceMust(t, q.Del(&out))
	serviceMust(t, c.Del(&root))
}
