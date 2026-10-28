package polynomial

// Source: algebraic_numbers.cpp:180-216; qualified signed-host TLS release.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Public reallocations reach the real signed accounting frontier before the
// next source free. No private counters or completed-release flags are set.
func algebraicCellDeletionTLSLane(t *testing.T, rt *umemory.Runtime) umemory.Block {
	t.Helper()
	contextTLSNearMinimum(t, rt)
	lane, e := rt.Allocate(8)
	serviceMust(t, e)
	delta := int64(-1000000)
	if e = rt.Reallocate(&lane, uint64(16)+uint64(delta)-8); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("public TLS setup missed signed guard", e)
	}
	return lane
}
func algebraicCellDeletionFinishLane(t *testing.T, rt *umemory.Runtime, lane *umemory.Block) {
	t.Helper()
	contextTLSFinish(t, rt, func() error {
		if *lane == (umemory.Block{}) {
			return nil
		}
		return rt.Deallocate(lane)
	})
}
func TestAlgebraicCellDeletionReleasedHeaderSourceGuard(t *testing.T) {
	c, _, rt := algebraicNegTLSFixture(t, nil)
	one := unumeral.NewInt(1)
	var a Algebraic
	serviceMust(t, c.SetInteger(&a, &one))
	original := a
	lane := algebraicCellDeletionTLSLane(t, rt)
	before := rt.Stats().LiveBlocks
	body := c.Del(&a)
	if !errors.Is(body, arena.ErrBounds) || a != original || rt.Stats().LiveBlocks+1 != before {
		t.Fatal("header physical release/source caller frontier", body, a)
	}
	z := c.config.Rationals.Integers()
	used, stats := z.Budget().Used(), rt.Stats()
	if e := c.Del(&a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Error("literal source deletion acknowledged consumed header", e)
	}
	if a != original || used != z.Budget().Used() || stats != rt.Stats() {
		t.Error("stale header source guard changed actual state")
	}
	contextTLSFinish(t, rt, func() error { return c.CleanupValue(&a) })
	if a != (Algebraic{}) {
		t.Fatal("owning cleanup retained released header")
	}
	algebraicCellDeletionFinishLane(t, rt, &lane)
}
func TestAlgebraicCellDeletionReleasedNumeralSourceGuard(t *testing.T) {
	c, _, rt := algebraicNegTLSFixture(t, nil)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377"))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.config.Rationals.Del(&q))
	l, e := c.location(&a)
	serviceMust(t, e)
	r, e := c.load(l)
	serviceMust(t, e)
	token := r.value.Num.NativeCellToken()
	lane := algebraicCellDeletionTLSLane(t, rt)
	before := rt.Stats().LiveBlocks
	body := c.Del(&a)
	if !errors.Is(body, arena.ErrBounds) || l.stage != 0 || rt.Stats().LiveBlocks+1 != before {
		t.Fatal("NUM physical release/source member frontier", body, l.stage)
	}
	z := c.config.Rationals.Integers()
	used, stats := z.Budget().Used(), rt.Stats()
	original := a
	if e := c.Del(&a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Error("literal source deletion acknowledged consumed numeral", e)
	}
	if a != original || l.stage != 0 || used != z.Budget().Used() || stats != rt.Stats() {
		t.Error("stale NUM source guard changed actual state")
	}
	contextTLSFinish(t, rt, func() error { return c.CleanupValue(&a) })
	if z.IsLiveCellToken(token) || a != (Algebraic{}) {
		t.Fatal("owning cleanup retained NUM descriptor or caller")
	}
	algebraicCellDeletionFinishLane(t, rt, &lane)
}

func TestAlgebraicCellDeletionTagSwitchBeforeMembers(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertion erased")
	}
	for _, root := range []bool{false, true} {
		t.Run(map[bool]string{false: "basic", true: "root"}[root], func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			var a Algebraic
			if root {
				a = algebraicTinyBinaryBoundsRoot(t, c)
			} else {
				n := unumeral.NewInt(1)
				serviceMust(t, c.SetInteger(&a, &n))
			}
			polynomialAssertionActionsFixture(t, "")
			serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
			before := a
			z := c.config.Rationals.Integers()
			used, stats := z.Budget().Used(), z.RuntimeOwner().Stats()
			if e := c.Del(&a); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("deletion read members before established tag switch", e)
			}
			if a != before || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
				t.Fatal("closed tag switch followed cell mutation")
			}
			serviceMust(t, c.CleanupValue(&a))
		})
	}
}
func TestAlgebraicCellDeletionZeroSkipsTagSwitch(t *testing.T) {
	c, _, _ := algebraicCompareFixtureProfile(t, true)
	if polynomialSourceDebug {
		polynomialAssertionActionsFixture(t, "")
		serviceMust(t, diagnostic.BorrowProcessDebugContext().Close())
	}
	var a Algebraic
	serviceMust(t, c.Del(&a))
}
func TestAlgebraicCellDeletionPolynomialNUMSourceGuard(t *testing.T) {
	c, _, rt := algebraicNegTLSFixture(t, nil)
	a := algebraicLargeDyadicPromotionRoot(t, c)
	l, e := c.location(&a)
	serviceMust(t, e)
	lane := algebraicCellDeletionTLSLane(t, rt)
	before := rt.Stats().LiveBlocks
	body := c.Del(&a)
	if !errors.Is(body, arena.ErrBounds) || l.coefficientCursor != 0 || rt.Stats().LiveBlocks+1 != before {
		t.Fatal("polynomial NUM source release frontier", body, l.coefficientCursor)
	}
	original := a
	z := c.config.Rationals.Integers()
	used, stats := z.Budget().Used(), rt.Stats()
	if e := c.Del(&a); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("source polynomial deletion consumed released coefficient", e)
	}
	if a != original || l.coefficientCursor != 0 || used != z.Budget().Used() || stats != rt.Stats() {
		t.Fatal("coefficient guard changed actual source state")
	}
	contextTLSFinish(t, rt, func() error { return c.CleanupValue(&a) })
	algebraicCellDeletionFinishLane(t, rt, &lane)
}

func TestAlgebraicCellDeletionVectorOwningContinuation(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(map[bool]string{false: "close", true: "reset"}[reset], func(t *testing.T) {
			c, _, rt := algebraicNegTLSFixture(t, nil)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377"))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			serviceMust(t, c.config.Rationals.Del(&q))
			v, e := c.NewVector()
			serviceMust(t, e)
			serviceMust(t, v.PushCopy(&a))
			serviceMust(t, v.PushCopy(&a))
			serviceMust(t, c.Del(&a))
			used, stats := c.config.Rationals.Budget().Used(), rt.Stats()
			serviceMust(t, v.CleanupRetained())
			if v.Len() != 2 || used != c.config.Rationals.Budget().Used() || stats != rt.Stats() {
				t.Fatal("healthy vector cleanup acquired live values")
			}
			lane := algebraicCellDeletionTLSLane(t, rt)
			operation := v.Close
			if reset {
				operation = v.Reset
			}
			if e := operation(); !errors.Is(e, arena.ErrBounds) {
				t.Fatal("vector source release frontier missing", e)
			}
			used, stats = c.config.Rationals.Budget().Used(), rt.Stats()
			first := v.values[0]
			if e := operation(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal("vector source replay acknowledged consumed NUM", e)
			}
			if first != v.values[0] || used != c.config.Rationals.Budget().Used() || stats != rt.Stats() {
				t.Fatal("source replay changed vector ownership")
			}
			contextTLSFinish(t, rt, v.CleanupRetained)
			if v.Len() != 0 {
				t.Fatal("owning vector continuation retained values")
			}
			serviceMust(t, v.Close())
			algebraicCellDeletionFinishLane(t, rt, &lane)
		})
	}
}

func TestAlgebraicCellDeletionMappingOwningContinuation(t *testing.T) {
	c, _, rt := algebraicNegTLSFixture(t, nil)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377"))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.config.Rationals.Del(&q))
	values, e := NewSimpleAlgebraicValues(c)
	serviceMust(t, e)
	serviceMust(t, values.Push(0, &a))
	serviceMust(t, c.Del(&a))
	lane := algebraicCellDeletionTLSLane(t, rt)
	if e := values.Close(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("mapping source close did not reach release stop", e)
	}
	used, stats := c.config.Rationals.Budget().Used(), rt.Stats()
	if e := values.Close(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("mapping replay acknowledged source release", e)
	}
	if used != c.config.Rationals.Budget().Used() || stats != rt.Stats() {
		t.Fatal("mapping source replay changed physical state")
	}
	contextTLSFinish(t, rt, values.CleanupRetained)
	serviceMust(t, values.Close())
	algebraicCellDeletionFinishLane(t, rt, &lane)
}

func TestAlgebraicCellDeletionCoreOwnsSelectionContinuation(t *testing.T) {
	c, _, rt := algebraicNegTLSFixture(t, nil)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetString(&q, "1267650600228229401496703205377"))
	var a Algebraic
	serviceMust(t, c.SetRational(&a, &q))
	serviceMust(t, c.config.Rationals.Del(&q))
	v, e := c.NewVector()
	serviceMust(t, e)
	serviceMust(t, v.pushTake(&a))
	c.rootSelection = v
	lane := algebraicCellDeletionTLSLane(t, rt)
	if e := c.Close(); !errors.Is(e, arena.ErrBounds) || !c.closing || c.closed || c.rootSelection != v {
		t.Fatal("core owning close lost stopped selection", e)
	}
	contextTLSFinish(t, rt, c.Close)
	if !c.closed || c.rootSelection != nil || !v.closed {
		t.Fatal("core owning continuation retained selection")
	}
	algebraicCellDeletionFinishLane(t, rt, &lane)
}
