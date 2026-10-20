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

type resultantTraceWriter struct {
	strings.Builder
	t                 *testing.T
	m                 *MultivariatePolynomialManager
	entry, normalized bool
}

func (w *resultantTraceWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *resultantTraceWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if !w.entry && strings.Contains(w.String(), ":5362 ") {
		w.entry = true
		s := w.m.gcdFrames[w.m.gcdDepth-1]
		if s.depth != 2 {
			w.t.Fatal("entry followed later lexical declarations", s.depth)
		}
	}
	if !w.normalized && strings.Contains(w.String(), ":5402 ") {
		w.normalized = true
		s := w.m.gcdFrames[w.m.gcdDepth-1]
		if s.depth != 9 {
			w.t.Fatal("normalization crossed later lexical declarations", s.depth)
		}
	}
	for _, action := range []func() error{w.m.CloseChecked, w.m.SourceFinalize, w.m.CleanupGCD} {
		if e := action(); !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatal("executing diagnostic owner", e)
		}
	}
	if e := w.m.core.integer.CloseChecked(); e == nil {
		w.t.Fatal("diagnostic allowed numerical owner destruction")
	}
	return n, e
}
func TestPolynomialResultantTraceLiteralNormalizationAndLifetimes(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a := polynomialQueryTerms(t, c, []int32{2, -4}, [][]MonomialPower{{{0, 2}}, nil})
	b := polynomialQueryTerms(t, c, []int32{3, -9}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	w := &resultantTraceWriter{t: t, m: m}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("resultant")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, m.Resultant(a, b, 0, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 126})
	if tags.ShouldTrace("resultant") {
		want := "-------- [resultant] resultant src/math/polynomial/polynomial.cpp:5362 ---------\nresultant(A, B, x)\nA: 2 x0^2 - 4\nB: 3 x0 - 9\nx: 0\n------------------------------------------------\n" +
			"-------- [resultant] resultant src/math/polynomial/polynomial.cpp:5402 ---------\nresultant(A, B, x) after normalization\nA: x0^2 - 2\nB: x0 - 3\nx: 0\nt: 18\n------------------------------------------------\n"
		if !w.entry || !w.normalized || w.String() != want {
			t.Fatal("literal normalized records", w.String())
		}
	} else if w.Len() != 0 {
		t.Fatal("erased record executed")
	}
}
func TestPolynomialResultantTraceZeroAndConstantEarlyExits(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	three := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{nil})
	seven := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	var text strings.Builder
	tags := diagnostic.NewTraceTagsWithWriter(&text)
	tags.Enable("resultant")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	for _, a := range []Polynomial{c.zero, three} {
		text.Reset()
		serviceMust(t, m.Resultant(a, seven, 0, out))
		want := int64(1)
		if a == c.zero {
			want = 0
		}
		expected := map[[3]uint32]int64{}
		if want != 0 {
			expected[[3]uint32{}] = want
		}
		polynomialQueryExpect(t, c, out.root.value, expected)
		if tags.ShouldTrace("resultant") && (strings.Count(text.String(), ":5362 ") != 1 || strings.Contains(text.String(), ":5402 ")) {
			t.Fatal("source early exit records", text.String())
		}
	}
}

func TestPolynomialResultantTraceActualEntryTLSRefusal(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	m, q, rt, budget := polynomialDisplayTLSOwner(t)
	c := m.core
	z := q.Integers()
	var big unumeral.Int
	var p Polynomial
	var out *PolynomialRef
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupGCD())
		if out != nil {
			serviceMust(t, out.Close())
		}
		if p != (Polynomial{}) {
			serviceMust(t, c.DecRef(p))
		}
		serviceMust(t, z.Del(&big))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("actual owner refunds", budget.Used())
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
	out, e = NewPolynomialRef(c)
	serviceMust(t, e)
	tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
	tags.Enable("resultant")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, rt.SetMaxSize(1))
	body := m.Resultant(p, c.zero, 0, out)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || len(m.gcdFrames) != 1 {
		t.Fatal("actual entry display refusal", body)
	}
	s := m.gcdFrames[0]
	if !c.displayReceiptPending(&s.traceDisplay) || out.root.value != (Polynomial{}) {
		t.Fatal("entry display lost accepted receipt or published result")
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots == ([3]umemory.Block{}) || roots != s.traceDisplay.frame.numeric.Allocator {
		t.Fatal("exact pending formatter allocation", roots)
	}
	serviceMust(t, m.CleanupGCD())
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) || c.displayReceiptPending(&s.traceDisplay) {
		t.Fatal("matched cleanup retained formatter", roots)
	}
}

func TestPolynomialResultantTraceRefusingStreamStillFormats(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a := polynomialQueryTerms(t, c, []int32{1, -2}, [][]MonomialPower{{{0, 2}}, nil})
	b := polynomialQueryTerms(t, c, []int32{1, -3}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	w := &linearTraceFailWriter{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("resultant")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	before := c.display.next
	serviceMust(t, m.Resultant(a, b, 0, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 7})
	if tags.ShouldTrace("resultant") && (w.calls != 1 || c.display.next < before+5) {
		t.Fatal("stream refusal suppressed project formatting", w.calls, c.display.next-before)
	}
}

type discriminantSignTraceWriter struct {
	strings.Builder
	t        *testing.T
	m        *MultivariatePolynomialManager
	observed bool
}

func (w *discriminantSignTraceWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *discriminantSignTraceWriter) Write(p []byte) (int, error) {
	n, e := w.Builder.Write(p)
	if !w.observed && strings.Contains(w.String(), ":5483 ") {
		w.observed = true
		if w.m.discriminant.depth != 1 || w.m.gcdDepth != 0 {
			w.t.Fatal("sign record crossed lc construction or resultant return", w.m.discriminant.depth, w.m.gcdDepth)
		}
		for _, action := range []func() error{w.m.CloseChecked, w.m.SourceFinalize, w.m.CleanupGCD, w.m.CleanupDiscriminant} {
			if e := action(); !errors.Is(e, ErrPolynomialPending) {
				w.t.Fatal("executing sign record owner", e)
			}
		}
		if e := w.m.core.integer.CloseChecked(); e == nil {
			w.t.Fatal("sign record released numerical owner")
		}
	}
	return n, e
}
func TestPolynomialDiscriminantTraceSignBeforeLeadingNumeral(t *testing.T) {
	for _, tc := range []struct {
		degree uint32
		want   int64
		sign   string
	}{{0, 0, ""}, {1, 1, "0"}, {2, 8, "1"}, {3, -108, "1"}, {4, -2048, "0"}} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		var p Polynomial
		if tc.degree == 0 {
			p = polynomialQueryTerms(t, c, []int32{2}, [][]MonomialPower{nil})
		} else {
			p = polynomialQueryTerms(t, c, []int32{1, -2}, [][]MonomialPower{{{0, tc.degree}}, nil})
		}
		out, e := NewPolynomialRef(c)
		serviceMust(t, e)
		w := &discriminantSignTraceWriter{t: t, m: m}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("resultant")
		c.monomials.config.Trace = tags
		t.Cleanup(func() { tags.Finalize(); serviceMust(t, m.CleanupDiscriminant()); serviceMust(t, out.Close()) })
		serviceMust(t, m.Discriminant(p, 0, out))
		expected := map[[3]uint32]int64{}
		if tc.want != 0 {
			expected[[3]uint32{}] = tc.want
		}
		polynomialQueryExpect(t, c, out.root.value, expected)
		if tags.ShouldTrace("resultant") && tc.degree != 0 {
			if !w.observed || !strings.Contains(w.String(), "discriminant src/math/polynomial/polynomial.cpp:5483 ") || !strings.Contains(w.String(), "discriminant sign: "+tc.sign+"\n") {
				t.Fatal("sign record", tc.degree, w.String())
			}
		} else if w.observed {
			t.Fatal("unreached source record executed", tc.degree)
		}
	}
}

func TestPolynomialResultantTraceEveryReservation(t *testing.T) {
	if !polynomialTraceEnabledForTest() {
		t.Skip("trace erased")
	}
	success, refused := false, false
	for prefix := int64(0); prefix < 512 && !success; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			a := polynomialQueryTerms(t, c, []int32{2, -4}, [][]MonomialPower{{{0, 2}}, nil})
			b := polynomialQueryTerms(t, c, []int32{3, -9}, [][]MonomialPower{{{0, 1}}, nil})
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			defer func() { c.integer.Budget().FailAfter(-1); serviceMust(t, m.CleanupGCD()); serviceMust(t, out.Close()) }()
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("resultant")
			c.monomials.config.Trace = tags
			defer tags.Finalize()
			c.integer.Budget().FailAfter(prefix)
			body := m.Resultant(a, b, 0, out)
			c.integer.Budget().FailAfter(-1)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{a, b, out.root.value}, nil, out.header)
			if body == nil {
				success = true
				polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 126})
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual reservation category", body)
				}
			}
			serviceMust(t, m.CleanupGCD())
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{a, b, out.root.value}, nil, out.header)
		}) {
			return
		}
	}
	if !success || !refused {
		t.Fatal("reservation endpoints", success, refused)
	}
}
