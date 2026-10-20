package polynomial

import (
	"bytes"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialUnivariateFactorTraceExactEntryReceiptBeforeLocals(t *testing.T) {
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
	var receipt polynomialFactorUnivariateInvocationReceipt
	defer func() {
		tags.Finalize()
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, false))
		serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, true))
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
			t.Fatal("retained leaf display graph", budget.Used())
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
	b, e := c.monomials.Unit()
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
	body := m.factorSquareFreePrimitiveUnivariateWithReceipt(p, factors, 1, DefaultUPolynomialFactorParams(), &receipt)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !m.matchesFactorUnivariate(&receipt) || !m.matchesGCDInvocation(&receipt.frame.traceEntry) || !c.displayReceiptPending(&receipt.frame.traceEntry.frame.traceDisplay) {
		t.Fatal("entry formatting invocation not retained", body)
	}
	if receipt.frame.constructed != 0 {
		t.Fatal("entry failure constructed later source polynomial references", receipt.frame.constructed)
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || roots != receipt.frame.traceEntry.frame.traceDisplay.frame.numeric.Allocator {
		t.Fatal("exact BIG copy raw not captured", roots)
	}
	if len(factors.factors) != 0 {
		t.Fatal("formatter failure performed factor push")
	}
	serviceMust(t, m.cleanupFactorUnivariateInvocation(&receipt, false))
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || c.displayReceiptPending(&receipt.frame.traceEntry.frame.traceDisplay) {
		t.Fatal("owning cleanup omitted leaf formatter", roots)
	}
}
