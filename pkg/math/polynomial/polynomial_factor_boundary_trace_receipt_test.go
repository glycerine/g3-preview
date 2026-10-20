package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialFactorBoundaryTraceActualTLSReceiptBeforeLocals(t *testing.T) {
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	m, q, rt, budget := polynomialDisplayTLSOwner(t)
	c := m.core
	z := q.Integers()
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("factor")
	var big unumeral.Int
	var p Polynomial
	var factors *PolynomialFactors
	defer func() {
		tags.Finalize()
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupFactor())
		serviceMust(t, c.CleanupDisplay())
		if factors != nil {
			serviceMust(t, factors.CloseChecked())
		}
		if p != (Polynomial{}) {
			serviceMust(t, c.DecRef(p))
		}
		serviceMust(t, z.Del(&big))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("retained boundary display graph", budget.Used())
		}
	}()
	if !tags.ShouldTrace("factor") {
		t.Skip("TRACE compiled out")
	}
	for i := 0; i < 2; i++ {
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
	}
	a, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}})
	serviceMust(t, e)
	b, e := c.monomials.MkMonomial([]MonomialPower{{1, 2}})
	serviceMust(t, e)
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, z.SetLimbs(&big, digits))
	p, e = c.MkTerms([]unumeral.Int{big, unumeral.NewInt(-1)}, []Monomial{a, b})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	factors, e = NewPolynomialFactors(m)
	serviceMust(t, e)
	c.monomials.config.Trace = tags
	serviceMust(t, rt.SetMaxSize(1))

	body := m.Factor(p, factors, DefaultUPolynomialFactorParams())
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !m.factorDiagnosticActual(&m.factorPrelude) || !c.displayReceiptPending(&m.factorPrelude.polynomial) {
		t.Fatal("entry formatting invocation not retained", body)
	}
	if len(m.factorFrames) != 0 {
		t.Fatal("entry failure constructed later source polynomial references", len(m.factorFrames))
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || roots != m.factorPrelude.polynomial.frame.numeric.Allocator {
		t.Fatal("exact BIG copy raw not captured", roots)
	}
	for _, e := range []error{m.CloseChecked(), m.SourceFinalize()} {
		if !errors.Is(e, ErrPolynomialPending) {
			t.Fatal("pending diagnostic allowed manager destruction", e)
		}
	}
	copied := m.factorPrelude
	serviceMust(t, m.cleanupFactorDiagnostic(&copied))
	if !m.factorPrelude.active || !c.displayReceiptPending(&m.factorPrelude.polynomial) {
		t.Fatal("copied diagnostic scope acquired actual formatter")
	}
	serviceMust(t, m.CleanupFactor())
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || c.displayReceiptPending(&m.factorPrelude.polynomial) {
		t.Fatal("owning cleanup omitted entry formatter", roots)
	}
}
