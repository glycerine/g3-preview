package polynomial

import (
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"io"
	bigmath "math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialPseudoRemainderEqualityRunsWithAssertionsDisabled(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("debug wrapper erased")
	}
	polynomialAssertionActionsFixture(t, "T")
	serviceMust(t, diagnostic.BorrowProcessDebugContext().EnableAssertions(false))
	c := polynomialCellFixture(t)
	tags := diagnostic.NewTraceTags()
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	_, r := polynomialPseudoOutputs(t, c)
	if c.queryPositions.self != nil || len(c.queryPositions.values) != 0 {
		t.Fatal("fixture already used equality table")
	}
	d := uint32(0)
	serviceMust(t, c.PseudoRemainder(p, q, 0, &d, r))
	if d != 2 {
		t.Fatal("independent exponent", d)
	}
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{}: 5})
	// Source lhs=4*x^2+4 and rhs=(2*x-1)*(2*x+1)+5 are distinct
	// polynomial objects. Eager eq must populate and reset the actual m_m2pos.
	if c.queryPositions.self != &c.queryPositions || len(c.queryPositions.values) == 0 {
		t.Fatal("source equality was skipped with assertions disabled")
	}
	for _, v := range c.queryPositions.values {
		if v != ^uint32(0) {
			t.Fatal("source equality did not reset its real table", v)
		}
	}
	if c.display.next != 0 {
		t.Fatal("trace-disabled equality regression used diagnostic formatting")
	}
}

func TestPolynomialPseudoRemainderVerificationTraceLiteralAndLifetime(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	_, r := polynomialPseudoOutputs(t, c)
	w := &pseudoTraceWriter{t: t, c: c}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("pseudo_remainder")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	d := uint32(0)
	serviceMust(t, c.PseudoRemainder(p, q, 0, &d, r))
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{}: 5})
	if polynomialSourceDebug && tags.ShouldTrace("pseudo_remainder") {
		want := "-------- [pseudo_remainder] pseudo_remainder src/math/polynomial/polynomial.cpp:5072 ---------\npseudo_division bug\np:   x0^2 + 1\nq:   2 x0 + 1\nQ:   2 x0 - 1\nR:   5\nl^d: 4\nlhs: 4 x0^2 + 4\nrhs: 4 x0^2 + 4\n------------------------------------------------\n"
		if w.String() != want || !w.verification {
			t.Fatal("literal verification record/frontier", w.String(), w.verification)
		}
	} else if w.Len() != 0 {
		t.Fatal("erased debug record executed", w.String())
	}
}

func TestPolynomialPseudoReducedTraceTagAndPredicateReadOrder(t *testing.T) {
	c := polynomialCellFixture(t)
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, c.tracePseudoReducedMonomial(Monomial{}, ^uint32(0), 0, 0))
	tags.Enable("polynomial")
	if !tags.ShouldTrace("polynomial") {
		return
	}
	// A false predicate must not read the unused m_prime source operand.
	unit, e := c.monomials.MkMonomial(nil)
	serviceMust(t, e)
	serviceMust(t, c.tracePseudoReducedMonomial(unit, 0, 0, 0))
	if text.Len() != 0 {
		t.Fatal("false conditional emitted")
	}
	if e := c.tracePseudoReducedMonomial(Monomial{}, 0, 0, 0); e == nil {
		t.Fatal("enabled conditional erased the real degree read")
	}
	if text.Len() != 0 {
		t.Fatal("failed predicate reached output")
	}
}

func TestPolynomialPseudoRemainderVerificationFailedStreamStillFormats(t *testing.T) {
	c := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	_, r := polynomialPseudoOutputs(t, c)
	writer := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("pseudo_remainder")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	d := uint32(0)
	before := c.display.next
	serviceMust(t, c.PseudoRemainder(p, q, 0, &d, r))
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{}: 5})
	if polynomialSourceDebug && tags.ShouldTrace("pseudo_remainder") {
		want := uint64(12) // Actual terms in p/q/Q/R/ld/lhs/rhs.
		enabled, e := diagnostic.SourceAssertionsEnabled(diagnostic.BorrowProcessDebugContext())
		serviceMust(t, e)
		if enabled {
			want += 5
		} // Q/R/ld/lhs/rhs use public reference display.
		if writer.calls != 1 || c.display.next-before != want {
			t.Fatal("failed stream skipped source formatting", writer.calls, c.display.next-before, want)
		}
	} else if writer.calls != 0 {
		t.Fatal("erased verification reached writer")
	}
}

func TestPolynomialPseudoRemainderVerificationEveryReservation(t *testing.T) {
	if !polynomialSourceDebug || !polynomialTraceEnabledForTest() {
		t.Skip("verification trace erased")
	}
	success, refused := false, false
	for prefix := int64(0); prefix < 512 && !success; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
			q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			defer func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, c.CleanupPseudoDivision())
				serviceMust(t, r.Close())
			}()
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("pseudo_remainder")
			c.monomials.config.Trace = tags
			defer tags.Finalize()
			d := uint32(0)
			c.integer.Budget().FailAfter(prefix)
			body := c.PseudoRemainder(p, q, 0, &d, r)
			c.integer.Budget().FailAfter(-1)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value}, nil, r.header)
			if body == nil {
				success = true
				if d != 2 {
					t.Fatal("pseudo exponent", d)
				}
				polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{}: 5})
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual allocation category", body)
				}
			}
			serviceMust(t, c.CleanupPseudoDivision())
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value}, nil, r.header)
		}) {
			return
		}
	}
	if !success || !refused {
		t.Fatal("verification reservation campaign incomplete", success, refused)
	}
}

func TestPolynomialPseudoRemainderVerificationActualTLSCopyRefusal(t *testing.T) {
	if !polynomialSourceDebug || !polynomialTraceEnabledForTest() {
		t.Skip("verification trace erased")
	}
	full, qm, rt, budget := polynomialDisplayTLSOwner(t)
	c := full.core
	z := qm.Integers()
	c.config.Checkpoint = func() bool { return true }
	c.config.Cooperation = func() error { return nil }
	var external []umemory.Block
	t.Cleanup(func() {
		serviceMust(t, full.CloseChecked())
		serviceMust(t, qm.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("verification graph refund", budget.Used())
		}
	})
	var big unumeral.Int
	t.Cleanup(func() { serviceMust(t, z.Del(&big)) })
	digits := make([]uint32, 64)
	digits[0], digits[63] = 3, 1
	serviceMust(t, z.SetLimbs(&big, digits))
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	square, e := c.monomials.MkMonomial([]MonomialPower{{0, 2}})
	serviceMust(t, e)
	unit, e := c.monomials.MkMonomial(nil)
	serviceMust(t, e)
	p, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(1), big}, []Monomial{square, unit})
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	q := polynomialQueryTerms(t, c, []int32{2, 1}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, r.Close()) })
	t.Cleanup(func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.CleanupPseudoDivision())
		for i := range external {
			serviceMust(t, rt.Deallocate(&external[i]))
		}
	})
	armed := false
	writer := &linearOperationProbeWriter{probe: func() {
		if armed {
			return
		}
		armed = true
		s := &c.pseudoDivision
		if s.depth != 7 || s.constructed != 127 || len(c.queryPositions.values) == 0 {
			t.Fatal("verification gate crossed actual source frontier", s.depth, s.constructed)
		}
		// Hold a real caller allocation at the TLS accounting boundary; do not edit
		// the source lane. The following project Copy acquires the refused raw block.
		remaining := int64(100000) - rt.State().LocalSize - 8
		if remaining < 0 {
			block, e := rt.Allocate(0)
			serviceMust(t, e)
			external = append(external, block)
			remaining = int64(100000) - rt.State().LocalSize - 8
		}
		if remaining < 0 {
			t.Fatal("unreachable TLS prefix", rt.State())
		}
		block, e := rt.Allocate(uint64(remaining))
		serviceMust(t, e)
		external = append(external, block)
		if rt.State().LocalSize != 100000 {
			t.Fatal("actual TLS boundary", rt.State())
		}
		serviceMust(t, rt.SetMaxSize(1))
	}}
	tags := diagnostic.NewTraceTagsWithWriter(writer)
	tags.Enable("pseudo_remainder")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	var receipt polynomialPseudoDivisionInvocationReceipt
	d := uint32(0)
	body := c.pseudoRemainderWithReceipt(p, q, 0, &d, r, &receipt)
	serviceMust(t, rt.SetMaxSize(0))
	if !armed || !numericPhysicalReservationRefusal(body) || !c.matchesPseudoInvocation(&receipt) {
		t.Fatal("actual verification copy refusal", body, armed)
	}
	display := c.pseudoDivision.traceDisplay
	if !c.displayReceiptPending(&display) {
		t.Fatal("accepted verification formatter missing")
	}
	objects, blocks := display.frame.numeric.PendingObject(), display.frame.numeric.PendingAllocatorBlocks()
	if objects == (umemory.ObjectRef{}) && blocks == ([3]umemory.Block{}) {
		t.Fatal("no actual formatter allocation")
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p, q, r.root.value}, []*unumeral.Int{&big}, r.header, external)
	stale := receipt
	stale.invocation++
	serviceMust(t, c.cleanupPseudoInvocation(&stale, true))
	copied := receipt
	copyFrame := *receipt.frame
	copied.frame = &copyFrame
	serviceMust(t, c.cleanupPseudoInvocation(&copied, true))
	if !c.displayReceiptPending(&display) || display.frame.numeric.PendingObject() != objects || display.frame.numeric.PendingAllocatorBlocks() != blocks {
		t.Fatal("stale/copy retired the actual verification formatter")
	}
	serviceMust(t, c.cleanupPseudoInvocation(&receipt, false))
	serviceMust(t, c.cleanupPseudoInvocation(&receipt, true))
	if c.displayReceiptPending(&display) || c.matchesPseudoInvocation(&receipt) {
		t.Fatal("matched verification cleanup incomplete")
	}
	polynomialImpPhysicalAuditNamedExtra(t, full, []Polynomial{p, q, r.root.value}, []*unumeral.Int{&big}, r.header, external)
	// The caller's computed remainder survives a later diagnostic failure.
	a, e := c.coefficientCore(r.root.value, 0)
	serviceMust(t, e)
	text, e := z.String(&a)
	serviceMust(t, e)
	want := new(bigmath.Int).Lsh(bigmath.NewInt(1), 2018)
	want.Add(want, bigmath.NewInt(13))
	if d != 2 || text != want.String() {
		t.Fatal("published remainder after diagnostic failure", d, text, want)
	}
}
