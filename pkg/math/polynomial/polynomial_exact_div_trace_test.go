package polynomial

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	"strings"
	"testing"
)

func TestPolynomialExactDivisionTraceBeforeZeroShortcut(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	defer tags.Finalize()
	tags.Enable("polynomial")
	c.monomials.config.Trace = tags
	var r polynomialExactDivInvocationReceipt
	c.integer.Budget().FailAfter(0)
	p, e := c.exactDivWithReceipt(c.zero, c.zero, &r)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, e)
	if p != c.zero || r.frame != nil || c.exactDivision.invocation != 0 {
		t.Fatal("zero path constructed locals", p, r)
	}
	if tags.ShouldTrace("polynomial") {
		want := "-------- [polynomial] exact_div src/math/polynomial/polynomial.cpp:5117 ---------\nexact division\np: 0\nq: 0\n------------------------------------------------\n"
		if out.String() != want {
			t.Fatalf("entry %q want %q", out.String(), want)
		}
	} else if out.Len() != 0 {
		t.Fatal("compiled out trace wrote")
	}
	before := r
	tags.Disable("polynomial")
	_, e = c.exactDivWithReceipt(c.zero, c.zero, &r)
	serviceMust(t, e)
	if r != before {
		t.Fatal("disabled zero path changed receipt")
	}
}

type exactDivisionTraceWriter struct {
	t        *testing.T
	m        *MultivariatePolynomialManager
	text     strings.Builder
	refuse   bool
	calls    int
	observed bool
}

func (w *exactDivisionTraceWriter) Write(p []byte) (int, error) {
	w.calls++
	c := w.m.core
	for _, action := range []func() error{w.m.CloseChecked, w.m.SourceFinalize, c.CloseChecked, c.SourceFinalizePrefix, c.CleanupDisplay, c.CleanupExactDivision} {
		if e := action(); !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatal("executing owner", e)
		}
	}
	for _, action := range []func() error{c.monomials.CloseChecked, c.monomials.SourceFinalize, c.monomials.CleanupCreation, c.monomials.CleanupRename} {
		if e := action(); !errors.Is(e, ErrMonomialPending) {
			w.t.Fatal("executing monomial owner", e)
		}
	}
	if e := c.integer.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		w.t.Fatal("executing integer owner", e)
	}
	if c.generatedActive && c.exactDivision.monomial.value == (Monomial{}) {
		w.observed = true
	}
	if w.refuse {
		return 0, errors.New("refused")
	}
	return w.text.Write(p)
}
func TestPolynomialExactDivisionTraceActualQuotientBeforeAssignment(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		p := polynomialQueryTerms(t, c, []int32{6}, [][]MonomialPower{{{0, 3}, {1, 2}}})
		q := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{0, 1}}})
		w := &exactDivisionTraceWriter{t: t, m: m, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("polynomial")
		c.monomials.config.Trace = tags
		defer tags.Finalize()
		result, e := c.ExactDiv(p, q)
		serviceMust(t, e)
		polynomialCaller(t, c, result)
		polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{2, 2, 0}: 3})
		if tags.ShouldTrace("polynomial") {
			if refuse {
				if w.calls != 1 {
					t.Fatal("stream failure not sticky", w.calls)
				}
			} else {
				if !strings.Contains(w.text.String(), "m_r: x0^3 x1^2\nm_q: x0\nm_r_q: x0^2 x1^2\n") || !w.observed {
					t.Fatal("missing quotient/frontier", w.text.String(), w.observed)
				}
			}
		}
	}
}

func TestPolynomialExactDivisionTraceExactNumericReceiptBeforeLocals(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		t.Run(fmt.Sprint(refuse), func(t *testing.T) { polynomialExactDivisionTraceTLSRefusal(t, refuse) })
	}
}
func polynomialExactDivisionTraceTLSRefusal(t *testing.T, refuse bool) {
	m, q, rt, budget := polynomialDisplayTLSOwner(t)
	c := m.core
	z := q.Integers()
	var out bytes.Buffer
	var writer io.Writer = &out
	refusing := &exactDivisionTraceWriter{t: t, m: m, refuse: true}
	if refuse {
		writer = refusing
	}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("polynomial")
	var big unumeral.Int
	var p Polynomial
	defer func() {
		tags.Finalize()
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.CleanupExactDivision())
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
	if !tags.ShouldTrace("polynomial") {
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
	var receipt polynomialExactDivInvocationReceipt
	_, body := c.exactDivWithReceipt(p, c.unit, &receipt)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !c.exactDivTraceInvocationPending(&receipt.trace) || receipt.frame != nil || c.exactDivision.invocation != 0 {
		t.Fatal("actual pre-local display not captured", body, receipt)
	}
	if refuse && refusing.calls != 1 {
		t.Fatal("failed output did not remain sticky", refusing.calls)
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || roots != c.exactDivTrace.display.frame.numeric.Allocator {
		t.Fatal("actual copy raw not retained", roots)
	}
	stale := receipt
	stale.trace.invocation++
	serviceMust(t, c.cleanupExactDivInvocation(&stale, true))
	copyScope := c.exactDivTrace
	copied := receipt
	copied.trace.frame = &copyScope
	serviceMust(t, c.cleanupExactDivInvocation(&copied, true))
	now, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if now != roots {
		t.Fatal("stale/copied trace acquired actual raw")
	}
	serviceMust(t, c.cleanupExactDivInvocation(&receipt, false))
	serviceMust(t, c.cleanupExactDivInvocation(&receipt, true))
	now, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if now != ([3]umemory.Block{}) || c.exactDivInvocationPending(&receipt) {
		t.Fatal("exact trace cleanup omitted pre-local child", now)
	}
}

func TestPolynomialExactDivisionTraceEveryReservation(t *testing.T) {
	for _, direct := range []bool{false, true} {
		success, refused := false, false
		for prefix := int64(0); prefix < 512 && !success; prefix++ {
			if !t.Run(fmt.Sprint(direct, "/", prefix), func(t *testing.T) {
				var c *MultivariatePolynomialCore
				if direct {
					c = polynomialCellDirectObjectFixture(t)
				} else {
					c = polynomialCellFixture(t)
				}
				p := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
				q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
				tags := diagnostic.NewTraceTagsWithWriter(&bytes.Buffer{})
				tags.Enable("polynomial")
				c.monomials.config.Trace = tags
				defer tags.Finalize()
				t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, c.CleanupExactDivision()) })
				budget := c.integer.Budget()
				budget.FailAfter(prefix)
				result, body := c.ExactDiv(p, q)
				budget.FailAfter(-1)
				if body == nil {
					polynomialCaller(t, c, result)
					polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 1, {}: -1})
					success = true
				} else {
					refused = true
					if !numericPhysicalReservationRefusal(body) {
						t.Fatal("actual refusal category", body)
					}
				}
				roots := []Polynomial{p, q}
				if body == nil {
					roots = append(roots, result)
				}
				polynomialExactDivisionPhysical(t, c, roots)
				serviceMust(t, c.CleanupExactDivision())
				polynomialExactDivisionPhysical(t, c, roots)
			}) {
				break
			}
		}
		if !success || !refused {
			t.Fatal("reservation campaign incomplete", direct, success, refused)
		}
	}
}

func TestPolynomialExactDivisionExecutingCleanupRefusalDoesNotMutateCause(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{6}, [][]MonomialPower{{{0, 3}}})
	q := polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{{{0, 1}}})
	seen, mutated := false, false
	var refused error
	c.config.Checkpoint = func() bool {
		if !seen && c.exactDivision.executing {
			seen = true
			before := c.exactDivision.cleanupCause
			refused = c.CleanupExactDivision()
			mutated = c.exactDivision.cleanupCause != before
		}
		return true
	}
	result, e := c.ExactDiv(p, q)
	serviceMust(t, e)
	polynomialCaller(t, c, result)
	if !seen || mutated || !errors.Is(refused, ErrPolynomialPending) {
		t.Fatal("executing cleanup changed actual cause", seen, mutated, refused)
	}
	polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{2, 0, 0}: 3})
}
