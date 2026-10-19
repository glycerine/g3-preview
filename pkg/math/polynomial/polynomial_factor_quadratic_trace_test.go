package polynomial

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strings"
	"testing"
)

func TestPolynomialQuadraticFactorTraceSourceRecordsAndContents(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, {{1, 2}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.CleanupGCD()); serviceMust(t, f.CloseChecked()) }()
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("factor")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, m.Factor(p, f, DefaultUPolynomialFactorParams()))
	polynomialFactorAssertReconstruction(t, m, p, f, 2)
	if tags.ShouldTrace("factor") {
		last := -1
		for _, text := range []string{":6604 ", ":6612 ", ":6636 ", ":6652 ", ":6666 ", ":6672 "} {
			at := strings.Index(out.String(), text)
			if at <= last {
				t.Fatalf("source record %q missing/out of order in %q", text, out.String())
			}
			last = at
		}
		if !strings.Contains(out.String(), "content(f1): 2\ncontent(f2): 2\n") {
			t.Fatalf("actual content calls not printed: %q", out.String())
		}
	} else if out.Len() != 0 {
		t.Fatal("TRACE erased but emitted output")
	}
}

type polynomialQuadraticTraceWriter struct {
	m      *MultivariatePolynomialManager
	f      *PolynomialFactors
	t      *testing.T
	calls  int
	refuse bool
}

func (w *polynomialQuadraticTraceWriter) Write(p []byte) (int, error) {
	w.calls++
	for _, e := range []error{w.m.CloseChecked(), w.m.SourceFinalize(), w.m.core.CloseChecked(), w.m.core.SourceFinalizePrefix(), w.m.core.CleanupDisplay(), w.m.CleanupGCD(), w.f.CloseChecked(), w.f.Reset(), w.f.Cleanup()} {
		if !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatalf("quadratic trace callback acquired owner: %v", e)
		}
	}
	if w.refuse {
		return 0, errors.New("ostream refused")
	}
	return len(p), nil
}
func TestPolynomialQuadraticFactorTraceBorrowAndFailedStreamComputesContent(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		p := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, {{1, 2}}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		w := &polynomialQuadraticTraceWriter{m: m, f: f, t: t, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		defer func() { tags.Finalize(); serviceMust(t, m.CleanupGCD()); serviceMust(t, f.CloseChecked()) }()
		c.monomials.config.Trace = tags
		count := func() uint64 {
			var n uint64
			for _, frame := range m.gcdFrames {
				n += frame.invocation
			}
			return n
		}
		serviceMust(t, m.factorQuadraticSquareFreePrimitive(p, f, 0, 1))
		control := count()
		serviceMust(t, f.Reset())
		tags.Enable("factor")
		before := count()
		serviceMust(t, m.factorQuadraticSquareFreePrimitive(p, f, 0, 1))
		if tags.ShouldTrace("factor") && w.calls == 0 {
			t.Fatal("writer never reached")
		}
		// Two content calls each execute both real source overloads, even
		// after the first stream insertion sets failbit.
		if tags.ShouldTrace("factor") && count()-before < control+4 {
			t.Fatal("failed stream suppressed source content computations")
		}
		tags.Finalize()
		polynomialFactorAssertReconstruction(t, m, p, f, 2)
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, f.CloseChecked())
	}
}

func TestPolynomialQuadraticFactorTraceExactEntryReceiptBeforeReferences(t *testing.T) {
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
		serviceMust(t, m.CleanupGCD())
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
			t.Fatal("retained quadratic display graph", budget.Used())
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
	var receipt polynomialGCDInvocationReceipt
	body := m.factorQuadraticSquareFreePrimitiveWithReceipts(p, factors, 0, 1, nil, &receipt, nil)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || !m.matchesGCDInvocation(&receipt) || !c.displayReceiptPending(&receipt.frame.traceDisplay) {
		t.Fatal("entry formatting invocation not retained", body)
	}
	if receipt.frame.depth != 0 {
		t.Fatal("entry failure constructed later source polynomial references", receipt.frame.depth)
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || roots != receipt.frame.traceDisplay.frame.numeric.Allocator {
		t.Fatal("exact BIG copy raw not captured", roots)
	}
	serviceMust(t, m.CleanupGCD())
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || c.displayReceiptPending(&receipt.frame.traceDisplay) {
		t.Fatal("owning cleanup omitted quadratic formatter", roots)
	}
}
