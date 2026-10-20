package polynomial

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func polynomialCellAssertionAction(t *testing.T, action string) {
	t.Helper()
	choices := action
	if action == "disabled" {
		choices = ""
	}
	polynomialAssertionActionsFixture(t, choices)
	if polynomialSourceDebug && action == "disabled" {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
}

// A freshly returned source polynomial has zero references. Continue and
// disabled assertions preserve the defined unsigned decrement, without
// repairing its count. Owning cleanup retires this actual caller root.
func TestPolynomialCellDecrementAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			polynomialCellAssertionAction(t, action)
			c := polynomialCellFixture(t)
			one := unumeral.NewInt(7)
			p, e := c.MkConstantTake(&one)
			serviceMust(t, e)
			defer func() { serviceMust(t, c.CleanupValue(&p)) }()
			var out polynomialDecrementOutcome
			e = c.decRefCoreOutcome(p, &out)
			count, read := c.readWord(p, 0)
			serviceMust(t, read)
			if polynomialSourceDebug && action == "T" {
				if !errors.Is(e, ErrPolynomialAssertion) || count != 0 || out.consumed || out.last {
					t.Fatal(e, count, out)
				}
			} else if e != nil || count != math.MaxUint32 || !out.consumed || out.last {
				t.Fatal(e, count, out)
			}
		})
	}
}

func TestPolynomialCellIndexAssertionActions(t *testing.T) {
	for _, action := range []string{"T", "C", "disabled"} {
		t.Run(action, func(t *testing.T) {
			// Each failed accessor must independently reach its source action.
			choices := action + action
			if action == "disabled" {
				choices = action
			}
			polynomialCellAssertionAction(t, choices)
			c := polynomialCellFixture(t)
			one := unumeral.NewInt(7)
			p, e := c.MkConstantTake(&one)
			serviceMust(t, e)
			defer func() { serviceMust(t, c.CleanupValue(&p)) }()
			_, coefficient := c.Coefficient(p, 1)
			_, monomial := c.TermMonomial(p, 1)
			for _, body := range []error{coefficient, monomial} {
				want := ErrPolynomialPartial
				if polynomialSourceDebug && action == "T" {
					want = ErrPolynomialAssertion
				}
				if !errors.Is(body, want) {
					t.Fatal(body, want)
				}
			}
		})
	}
}

func TestPolynomialCellClosedAssertionOwnerPaths(t *testing.T) {
	c := polynomialCellFixture(t)
	one := unumeral.NewInt(7)
	p, e := c.MkConstantTake(&one)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.CleanupValue(&p)) }()
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	defer func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		serviceMust(t, e)
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}()
	serviceMust(t, d.Close())
	_, coefficient := c.Coefficient(p, 0)
	_, monomial := c.TermMonomial(p, 0)
	var out polynomialDecrementOutcome
	decrement := c.decRefCoreOutcome(p, &out)
	for _, body := range []error{coefficient, monomial, decrement} {
		if polynomialSourceDebug {
			if !errors.Is(body, arena.ErrClosed) {
				t.Fatal(body)
			}
		} else if body != nil {
			t.Fatal(body)
		}
	}
	if polynomialSourceDebug {
		count, e := c.readWord(p, 0)
		serviceMust(t, e)
		if count != 1 || out.consumed || out.last {
			t.Fatal(count, out)
		}
	} else {
		// Native last DEC deleted this value; owning cleanup has no remaining root.
		p = Polynomial{}
		if !out.consumed || !out.last {
			t.Fatal(out)
		}
	}
}

// Stop at the real constructor assertion after coefficient transfer, before
// monomial publication. A closed switch owner tests this even when the actual
// monomial has a positive reference count; no invalid payload is manufactured.
func TestPolynomialCellConstructorAssertionReadOrder(t *testing.T) {
	c := polynomialCellFixture(t)
	term, e := c.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(term))
	defer func() {
		if term != (Monomial{}) {
			serviceMust(t, c.monomials.DecRef(term))
		}
	}()
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	serviceMust(t, d.Close())
	a := []unumeral.Int{unumeral.NewInt(7)}
	p, body := c.MkCoreTake(a, []Monomial{term})
	_, e = diagnostic.BindProcessDebugContext(previous)
	serviceMust(t, e)
	serviceMust(t, region.Close())
	if polynomialSourceDebug {
		if !errors.Is(body, arena.ErrClosed) || p != (Polynomial{}) || a[0] != (unumeral.Int{}) {
			t.Fatal(body, p, a)
		}
		l, e := c.polynomialLocation(c.pending.value)
		serviceMust(t, e)
		coefficients, monomials := l.initializedCoefficients, l.initializedMonomials
		// Retire the actual constructor prefix before reporting a mismatch.
		// An omitted-assertion control can transfer this caller's monomial
		// before stopping at a later accessor; acknowledge that real transfer.
		if monomials == 1 {
			term = Monomial{}
		}
		serviceMust(t, c.cleanupCreation())
		if coefficients != 1 || monomials != 0 {
			t.Fatal("constructor assertion after coefficient move, before monomial publication", coefficients, monomials)
		}
	} else {
		serviceMust(t, body)
		term = Monomial{} // actual constructor transfer
		serviceMust(t, c.CleanupValue(&p))
	}
}
