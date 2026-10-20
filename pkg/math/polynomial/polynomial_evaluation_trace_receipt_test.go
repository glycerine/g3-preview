package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialEvaluationTraceActualTLSReceiptBeforeAuxiliary(t *testing.T) {
	polynomialAssertionActionsFixture(t, "")
	if polynomialSourceDebug {
		serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	}
	m, q, rt, budget := polynomialDisplayTLSOwner(t)
	c := m.core
	z := q.Integers()
	var text bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("eval_bug")
	var big unumeral.Int
	var p Polynomial
	values := polynomialEvalRationalMap{manager: q}
	out := unumeral.NewRational()
	defer func() {
		tags.Finalize()
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupEvaluation())
		serviceMust(t, c.CleanupDisplay())
		serviceMust(t, q.Del(&out))
		if p != (Polynomial{}) {
			serviceMust(t, c.DecRef(p))
		}
		serviceMust(t, z.Del(&big))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("retained evaluation display graph", budget.Used())
		}
	}()
	if !tags.ShouldTrace("eval_bug") {
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
	c.monomials.config.Trace = tags
	serviceMust(t, rt.SetMaxSize(1))

	body := m.EvaluateRational(p, &values, &out)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !m.evaluationTraceActual(&m.evaluation.trace) || !c.displayReceiptPending(&m.evaluation.trace.display) {
		t.Fatal("entry formatting invocation not retained", body)
	}
	if len(m.evaluation.rationalFrames) != 0 {
		t.Fatal("entry failure constructed later source polynomial references", len(m.evaluation.rationalFrames))
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || roots != m.evaluation.trace.display.frame.numeric.Allocator {
		t.Fatal("exact BIG copy raw not captured", roots)
	}
	for _, e := range []error{m.CloseChecked(), m.SourceFinalize()} {
		if !errors.Is(e, ErrPolynomialPending) {
			t.Fatal("pending diagnostic allowed manager destruction", e)
		}
	}
	copied := m.evaluation.trace
	serviceMust(t, m.cleanupEvaluationTrace(&copied))
	if !m.evaluation.trace.active || !c.displayReceiptPending(&m.evaluation.trace.display) {
		t.Fatal("copied diagnostic scope acquired actual formatter")
	}
	serviceMust(t, m.CleanupEvaluation())
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || c.displayReceiptPending(&m.evaluation.trace.display) {
		t.Fatal("owning cleanup omitted entry formatter", roots)
	}
}
