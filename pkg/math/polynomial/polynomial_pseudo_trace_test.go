package polynomial

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
)

type pseudoTraceWriter struct {
	strings.Builder
	t               *testing.T
	c               *MultivariatePolynomialCore
	entry, adjusted bool
	verification    bool
	reduced         int
}

func (w *pseudoTraceWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *pseudoTraceWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	s := &w.c.pseudoDivision
	if !w.entry && strings.Contains(w.String(), ":4909 ") {
		w.entry = true
		if s.depth != 0 || s.constructed != 0 {
			w.t.Fatal("entry constructed core locals", s.depth, s.constructed)
		}
	}
	if !w.adjusted && strings.Contains(w.String(), ":4973 ") {
		w.adjusted = true
		if s.depth != 4 || s.refs[pseudoAdjusted].value == (Polynomial{}) {
			w.t.Fatal("correction local ended before its record", s.depth)
		}
	}
	if strings.Count(w.String(), ":5011 ") > w.reduced {
		w.reduced++
		if s.monomial.value == (Monomial{}) || w.c.generatedActive {
			w.t.Fatal("reduced trace preceded actual reference assignment")
		}
	}
	if !w.verification && strings.Contains(w.String(), ":5072 ") {
		w.verification = true
		if s.depth != 7 || s.constructed != 127 || len(w.c.queryPositions.values) == 0 {
			w.t.Fatal("verification preceded equality or crossed source lifetime", s.depth, s.constructed)
		}
	}
	for _, action := range []func() error{w.c.CloseChecked, w.c.CleanupPseudoDivision} {
		if e := action(); !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatal("executing pseudo record owner", e)
		}
	}
	if e := w.c.integer.CloseChecked(); e == nil {
		w.t.Fatal("executing record released numeral owner")
	}
	return n, e
}
func TestPolynomialPseudoTraceExactCorrectionAndSourceLifetimes(t *testing.T) {
	c := polynomialCellFixture(t)
	a := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 3}}})
	b := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{0, 1}}})
	q, r := polynomialPseudoOutputs(t, c)
	w := &pseudoTraceWriter{t: t, c: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, c.ExactPseudoDivide(a, b, 0, q, r))
	polynomialQueryExpect(t, c, q.root.value, map[[3]uint32]int64{{2, 0, 0}: 4})
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{})
	if tags.ShouldTrace("polynomial") {
		for _, text := range []string{"pseudo_division\np: x0^3\nq: 2 x0\nx: 0\n", "A: x0^3\nB: 2 x0\nl_B: 2\nr_B: 0\nQ: 0\nR: x0^3\nd: 0\n", "Q: x0^2\nR: 0\nd: 1\n", "l_B_e: 4\n", "deg_R: 3, deg_B: 1, x: 0\nx0^3, x0^2\n"} {
			if !strings.Contains(w.String(), text) {
				t.Fatal("source pseudo record", text, w.String())
			}
		}
		if !w.entry || !w.adjusted || w.reduced != 1 || strings.Count(w.String(), ":4955 ") != 2 || !strings.Contains(w.String(), "pseudo_division_core src/math/polynomial/polynomial.cpp:4909 ") {
			t.Fatal("source record sites", w.String())
		}
	} else if w.Len() != 0 {
		t.Fatal("erased record executed")
	}
}

func TestPolynomialPseudoTraceActualEntryTLSRefusal(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	full, qm, rt, budget := polynomialDisplayTLSOwner(t)
	c := full.core
	z := qm.Integers()
	var big unumeral.Int
	var p Polynomial
	var quotient, remainder *PolynomialRef
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.CleanupPseudoDivision())
		if remainder != nil {
			serviceMust(t, remainder.Close())
		}
		if quotient != nil {
			serviceMust(t, quotient.Close())
		}
		if p != (Polynomial{}) {
			serviceMust(t, c.DecRef(p))
		}
		serviceMust(t, z.Del(&big))
		serviceMust(t, full.CloseChecked())
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("actual pseudo owner refunds", budget.Used())
		}
	}()
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, z.SetLimbs(&big, digits))
	p, e = c.MkConstantTake(&big)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	quotient, e = NewPolynomialRef(c)
	serviceMust(t, e)
	remainder, e = NewPolynomialRef(c)
	serviceMust(t, e)
	tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, rt.SetMaxSize(1))
	body := c.ExactPseudoDivide(p, c.unit, 0, quotient, remainder)
	serviceMust(t, rt.SetMaxSize(0))
	s := &c.pseudoDivision
	if !numericPhysicalReservationRefusal(body) || !s.active || s.depth != 0 || s.constructed != 0 || !c.displayReceiptPending(&s.traceDisplay) {
		t.Fatal("entry crossed core local declarations or lost formatter", body, s.active, s.depth)
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots == ([3]umemory.Block{}) || roots != s.traceDisplay.frame.numeric.Allocator {
		t.Fatal("exact pseudo diagnostic allocation", roots)
	}
	serviceMust(t, c.CleanupPseudoDivision())
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || s.active || c.displayReceiptPending(&s.traceDisplay) {
		t.Fatal("matched pseudo cleanup retained diagnostic", roots)
	}
}

func TestPolynomialPseudoExecutingCleanupRefusalDoesNotBecomeSourceCause(t *testing.T) {
	c := polynomialCellFixture(t)
	a := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 3}}})
	b := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{0, 1}}})
	q, r := polynomialPseudoOutputs(t, c)
	t.Cleanup(func() { serviceMust(t, c.CleanupPseudoDivision()) })
	seen, mutated := false, false
	var refused error
	c.config.Checkpoint = func() bool {
		if !seen && c.pseudoDivision.executing {
			seen = true
			before := c.pseudoDivision.cleanupCause
			refused = c.CleanupPseudoDivision()
			mutated = c.pseudoDivision.cleanupCause != before
		}
		return true
	}
	serviceMust(t, c.ExactPseudoDivide(a, b, 0, q, r))
	if !seen || mutated || !errors.Is(refused, ErrPolynomialPending) {
		t.Fatal("executing refusal poisoned source continuation", seen, mutated, refused)
	}
	polynomialQueryExpect(t, c, q.root.value, map[[3]uint32]int64{{2, 0, 0}: 4})
}

func TestPolynomialPseudoTraceEarlyExitAndRemainderOnly(t *testing.T) {
	for _, constant := range []bool{false, true} {
		c := polynomialCellFixture(t)
		a := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 3}}})
		var b Polynomial
		if constant {
			b = polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
		} else {
			b = polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{0, 1}}})
		}
		q, r := polynomialPseudoOutputs(t, c)
		var output strings.Builder
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		tags.Enable("polynomial")
		c.monomials.config.Trace = tags
		if constant {
			d := uint32(0)
			serviceMust(t, c.PseudoDivide(a, b, 0, &d, q, r))
			if d != 1 {
				t.Fatal("constant exponent", d)
			}
		} else {
			serviceMust(t, c.ExactPseudoRemainder(a, b, 0, r))
		}
		polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{})
		if tags.ShouldTrace("polynomial") {
			if strings.Count(output.String(), ":4909 ") != 1 {
				t.Fatal("entry record", output.String())
			}
			if constant && strings.Contains(output.String(), ":4955 ") {
				t.Fatal("constant branch constructed loop record")
			}
			if !constant && (!strings.Contains(output.String(), "Q: 0\nR: 0\nd: 1\n") || !strings.Contains(output.String(), "l_B_e: 4\n")) {
				t.Fatal("remainder-only diagnostic operands", output.String())
			}
		}
		tags.Finalize()
	}
}
func TestPolynomialPseudoTraceEveryReservation(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	success, refused := false, false
	for prefix := int64(0); prefix < 512 && !success; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			a := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 3}}})
			b := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{0, 1}}})
			q, r := polynomialPseudoOutputs(t, c)
			t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, c.CleanupPseudoDivision()) })
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("polynomial")
			c.monomials.config.Trace = tags
			defer tags.Finalize()
			c.integer.Budget().FailAfter(prefix)
			body := c.ExactPseudoDivide(a, b, 0, q, r)
			c.integer.Budget().FailAfter(-1)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{a, b, q.root.value, r.root.value}, nil, q.header+r.header)
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, q.root.value, map[[3]uint32]int64{{2, 0, 0}: 4})
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual refusal category", body)
				}
			}
			serviceMust(t, c.CleanupPseudoDivision())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{a, b, q.root.value, r.root.value}, nil, q.header+r.header)
		}) {
			return
		}
	}
	if !success || !refused {
		t.Fatal("reservation endpoints", success, refused)
	}
}
