package polynomial

// Source: algebraic_numbers.cpp:3001,3028; mpbq.cpp to_mpq numerical call.

import (
	"errors"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBoundConversionNumericContextRejectsReplacementInvocation(t *testing.T) {
	var c *AlgebraicRootCore
	var old unumeral.NumericAllocationContext
	var firstInvocation uint64
	captured, probe, checked, armed := false, false, false, false
	transport := func(f umemory.Failure) error {
		if !armed || !c.bounds.conversion.executing {
			return f
		}
		armed = false
		s := &c.bounds.conversion
		if !probe {
			captured = true
			firstInvocation = c.bounds.invocation
			old = (unumeral.NumericAllocationContext{}).WithCaller(&s.hooks)
			return f
		}
		checked = true
		if c.bounds.invocation == firstInvocation {
			t.Error("replacement did not enter a new bound")
		}
		z := c.config.Rationals.Integers()
		a, b, out := unumeral.NewInt(2), unumeral.NewInt(3), unumeral.NewInt(99)
		values, remaining, used, stats := c.bounds.values, c.bounds.remaining, z.Budget().Used(), z.RuntimeOwner().Stats()
		e := z.WithAllocationContext(old).Add(&a, &b, &out)
		if !errors.Is(e, unumeral.ErrNumericOwner) || out != unumeral.NewInt(99) {
			t.Error("stale context acquired replacement conversion", e, out)
		}
		if values != c.bounds.values || remaining != c.bounds.remaining || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
			t.Error("stale context changed actual bound owners")
		}
		current := (unumeral.NumericAllocationContext{}).WithCaller(&s.hooks)
		e = z.WithAllocationContext(current).Add(&a, &b, &out)
		if e != nil || out != unumeral.NewInt(5) {
			t.Error("actual current context refused valid arithmetic", e, out)
		}
		return f
	}
	var rt *umemory.Runtime
	c, _, rt = algebraicNegTLSFixture(t, transport)
	root := algebraicTinyBinaryBoundsRoot(t, c)
	defer func() { serviceMust(t, c.Del(&root)) }()
	out := unumeral.NewRational()
	defer func() { serviceMust(t, c.config.Rationals.Del(&out)) }()
	for call := 0; call < 2; call++ {
		probe = call == 1
		pad := algebraicCellSourceTLSBoundary(t, c)
		armed = true
		body := c.GetUpperRational(&root, &out)
		armed = false
		serviceMust(t, rt.SetMaxSize(0))
		if body == nil {
			t.Error("actual conversion allocation failure not reached")
		}
		serviceMust(t, c.CleanupRetained())
		serviceMust(t, rt.Deallocate(&pad))
	}
	if !captured || !checked {
		t.Fatal("actual conversion failure callbacks not reached", captured, checked)
	}
}
