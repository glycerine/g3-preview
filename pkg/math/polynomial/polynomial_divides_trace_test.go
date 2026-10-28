package polynomial

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialDividesTracePrecedesZeroShortcut(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("polynomial")
	tags.Enable("divides")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	var receipt polynomialDividesInvocationReceipt
	budget := c.integer.Budget()
	budget.FailAfter(0)
	answer, e := c.dividesWithReceipt(c.zero, c.zero, &receipt)
	budget.FailAfter(-1)
	serviceMust(t, e)
	if !answer || receipt.frame != nil || c.dividesLocal.invocation != 0 {
		t.Fatal("zero shortcut constructed algorithm locals", answer, receipt)
	}
	if tags.ShouldTrace("divides") {
		want := "-------- [polynomial] divides src/math/polynomial/polynomial.cpp:5155 ---------\ndivides\nq: 0\np: 0\n------------------------------------------------\n" +
			"-------- [divides] divides src/math/polynomial/polynomial.cpp:5156 ---------\ndivides\nq: 0\np: 0\n------------------------------------------------\n"
		if out.String() != want {
			t.Fatalf("literal entry records %q want %q", out.String(), want)
		}
	} else if out.Len() != 0 {
		t.Fatal("erased TRACE wrote output")
	}
	tags.Disable("polynomial")
	tags.Disable("divides")
	before, text := receipt, out.String()
	answer, e = c.dividesWithReceipt(c.zero, c.zero, &receipt)
	serviceMust(t, e)
	if !answer || receipt != before || out.String() != text {
		t.Fatal("disabled zero shortcut changed the retained receipt")
	}
}

func TestPolynomialDividesTraceQuotientAndRefusal(t *testing.T) {
	for _, divides := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		pd, qd := uint32(1), uint32(2)
		if divides {
			pd, qd = qd, pd
		}
		p := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, pd}}})
		q := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, qd}}})
		var out bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&out)
		tags.Enable("polynomial")
		c.monomials.config.Trace = tags
		answer, e := c.Divides(q, p)
		serviceMust(t, e)
		if answer != divides {
			t.Fatal("independent monomial divisibility", answer, divides)
		}
		if tags.ShouldTrace("polynomial") {
			want := "m_r: x0\nm_q: x0^2\n"
			if divides {
				want = "m_r: x0^2\nm_q: x0\nm_r_q: x0\n"
			}
			if !strings.Contains(out.String(), "[polynomial] divides src/math/polynomial/polynomial.cpp:5181") || !strings.Contains(out.String(), want) {
				t.Fatalf("quotient record %q want %q", out.String(), want)
			}
		}
		tags.Finalize()
		serviceMust(t, c.CleanupDivides())
	}
}

type polynomialDividesTraceWriter struct {
	m      *MultivariatePolynomialManager
	t      *testing.T
	refuse bool
	calls  int
}

func (w *polynomialDividesTraceWriter) Write(p []byte) (int, error) {
	w.calls++
	for _, e := range []error{w.m.CloseChecked(), w.m.SourceFinalize(), w.m.core.CloseChecked(), w.m.core.SourceFinalizePrefix(), w.m.core.CleanupDisplay(), w.m.core.CleanupDivides()} {
		if !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatalf("trace callback acquired borrowed owner: %v", e)
		}
	}
	if w.refuse {
		return 0, errors.New("trace refused")
	}
	return len(p), nil
}
func TestPolynomialDividesTraceExecutingCallbackAndFailbit(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		w := &polynomialDividesTraceWriter{m: m, t: t, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("divides")
		m.core.monomials.config.Trace = tags
		answer, e := m.Divides(m.core.zero, m.core.zero)
		serviceMust(t, e)
		if !answer || tags.ShouldTrace("divides") && w.calls == 0 {
			t.Fatal("callback/failbit changed zero divisibility")
		}
		tags.Finalize()
	}
}

func TestPolynomialDividesTraceExactNumericReceiptBeforeLocals(t *testing.T) {
	m, q, rt, budget := polynomialDisplayTLSOwner(t)
	c := m.core
	z := q.Integers()
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("divides")
	var big unumeral.Int
	var p Polynomial
	defer func() {
		tags.Finalize()
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.CleanupDivides())
		if p != (Polynomial{}) {
			serviceMust(t, c.DecRef(p))
		}
		serviceMust(t, z.Del(&big))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("retained display graph", budget.Used())
		}
	}()
	if !tags.ShouldTrace("divides") {
		t.Skip("TRACE compiled out")
	}
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, z.SetLimbs(&big, digits))
	var e error
	p, e = c.MkConstantTake(&big)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	c.monomials.config.Trace = tags
	serviceMust(t, rt.SetMaxSize(1))
	var receipt polynomialDividesInvocationReceipt
	_, body := c.dividesWithReceipt(p, c.zero, &receipt)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !c.dividesTraceInvocationPending(&receipt.trace) || receipt.frame != nil || c.dividesLocal.invocation != 0 {
		t.Fatal("actual pre-local display not captured", body, receipt)
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || roots != c.dividesTrace.display.frame.numeric.Allocator {
		t.Fatal("actual copy raw not retained", roots)
	}
	stale := receipt
	stale.trace.invocation++
	serviceMust(t, c.cleanupDividesInvocation(&stale, true))
	copyScope := c.dividesTrace
	copied := receipt
	copied.trace.frame = &copyScope
	serviceMust(t, c.cleanupDividesInvocation(&copied, true))
	now, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if now != roots {
		t.Fatal("stale/copied trace acquired actual raw")
	}
	serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	now, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if now != ([3]umemory.Block{}) || c.dividesInvocationPending(&receipt) {
		t.Fatal("exact trace cleanup omitted pre-local child", now)
	}
}

func TestPolynomialDividesTraceZeroKeepsOlderAlgorithmReceipt(t *testing.T) {
	probe := diagnostic.NewTraceTags()
	probe.Enable("divides")
	enabled := probe.ShouldTrace("divides")
	probe.Finalize()
	if !enabled {
		t.Skip("TRACE compiled out")
	}
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{5, 4, 0}}})
	c := m.core
	q := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}, {1, 2}}})
	var receipt polynomialDividesInvocationReceipt
	var limit control.Reslimit
	limit.InitEmbedded(c.integer.Budget())
	previousCheckpoint := c.config.Checkpoint
	c.config.Checkpoint = limit.Inc
	t.Cleanup(func() { c.config.Checkpoint = previousCheckpoint; limit.Close() })
	// Cancel at the actual algorithm checkpoint. Small release numerals and
	// warm allocator pages need not make any fallible budget reservation.
	limit.Cancel()
	_, cause := c.dividesWithReceipt(q, p, &receipt)
	limit.ResetCancel()
	defer func() {
		serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
		serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	}()
	if cause == nil || limit.Count() != 1 || !c.dividesInvocationPending(&receipt) {
		t.Fatal("actual checkpoint did not retain an algorithm invocation", cause, limit.Count())
	}
	before := receipt
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("polynomial")
	tags.Enable("divides")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	answer, e := c.dividesWithReceipt(c.zero, c.zero, &receipt)
	serviceMust(t, e)
	if !answer || receipt.trace.core != c || out.Len() == 0 {
		t.Fatal("actual zero diagnostic invocation was not issued")
	}
	algorithm := receipt
	algorithm.trace = before.trace
	if algorithm != before {
		t.Fatal("zero diagnostics replaced older algorithm/prelude authority")
	}

}
