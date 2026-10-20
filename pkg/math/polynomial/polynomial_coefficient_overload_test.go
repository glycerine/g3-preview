package polynomial

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialCoefficientOverloadDiagnostics(t *testing.T) {
	for _, mutable := range []bool{false, true} {
		name := "const"
		if mutable {
			name = "mutable"
		}
		t.Run(name, func(t *testing.T) {
			output := polynomialAssertionActionsFixture(t, "T")
			c := polynomialCellFixture(t)
			value := unumeral.NewInt(7)
			p, e := c.MkConstantTake(&value)
			serviceMust(t, e)
			defer func() { serviceMust(t, c.CleanupValue(&p)) }()
			if mutable {
				_, e = c.mutableCoefficientCore(p, 1)
			} else {
				_, e = c.coefficientCore(p, 1)
			}
			if !polynomialSourceDebug {
				if !errors.Is(e, ErrPolynomialPartial) {
					t.Fatal(e)
				}
				return
			}
			if !errors.Is(e, ErrPolynomialAssertion) {
				t.Fatal(e)
			}
			_, e = output.Seek(0, 0)
			serviceMust(t, e)
			text, e := io.ReadAll(output)
			serviceMust(t, e)
			line := "1511"
			if mutable {
				line = "1512"
			}
			if !strings.Contains(string(text), "File: src/math/polynomial/polynomial.cpp\nLine: "+line+"\nidx < size()") {
				t.Fatalf("wrong source overload diagnostic: %q", text)
			}
		})
	}
}

func polynomialClosedCellAssertionContext(t *testing.T) {
	t.Helper()
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	t.Cleanup(func() {
		_, e := diagnostic.BindProcessDebugContext(previous)
		serviceMust(t, e)
		serviceMust(t, d.Close())
		serviceMust(t, region.Close())
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	})
	serviceMust(t, d.Close())
}

func TestPolynomialRawSwapErasesAccessorAssertions(t *testing.T) {
	c := polynomialCellFixture(t)
	x, e := c.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(x))
	serviceMust(t, c.monomials.IncRef(unit))
	coefficients := []unumeral.Int{unumeral.NewInt(3), unumeral.NewInt(7)}
	p, e := c.MkCoreTake(coefficients, []Monomial{x, unit})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, c.CleanupValue(&p)) })
	polynomialClosedCellAssertionContext(t)
	// Source swap(m_as[0],m_as[pos]); swap(m_ms[0],m_ms[pos])
	// does not call either indexed accessor, even in a debug build.
	serviceMust(t, c.swapTerms(p, 0, 1))
	a, e := c.coefficientFieldCore(p, 0)
	serviceMust(t, e)
	b, e := c.coefficientFieldCore(p, 1)
	serviceMust(t, e)
	first, e := c.monomialFieldCore(p, 0)
	serviceMust(t, e)
	second, e := c.monomialFieldCore(p, 1)
	serviceMust(t, e)
	if a != unumeral.NewInt(7) || b != unumeral.NewInt(3) || first != unit || second != x {
		t.Fatal(a, b, first, second)
	}
}

func TestPolynomialPendingCoefficientOwningCodec(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source constructor assertion stop")
	}
	c := polynomialCellFixture(t)
	term, e := c.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(term))
	t.Cleanup(func() { serviceMust(t, c.monomials.DecRef(term)) })
	polynomialClosedCellAssertionContext(t)
	coefficients := []unumeral.Int{unumeral.NewInt(7)}
	p, e := c.MkCoreTake(coefficients, []Monomial{term})
	if !errors.Is(e, arena.ErrClosed) || p != (Polynomial{}) {
		t.Fatal(p, e)
	}
	l, e := c.polynomialLocation(c.pending.value)
	serviceMust(t, e)
	if l.initializedCoefficients != 1 || l.initializedMonomials != 0 {
		t.Fatal(l)
	}
	// This explicit owning abort retires only the actual coefficient-only
	// prefix. It must not replay the source a(idx) assertion on that field.
	serviceMust(t, c.cleanupCreation())
	if c.pending.value != (Polynomial{}) || coefficients[0] != (unumeral.Int{}) {
		t.Fatal(c.pending, coefficients)
	}
}
