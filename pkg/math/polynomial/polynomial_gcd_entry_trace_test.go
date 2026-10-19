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

func TestPolynomialGCDEntrySourceTraceBeforeZeroShortcut(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, r.Close()) }()
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("gcd_calls")
	tags.Enable("polynomial_gcd")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, m.GCD(c.zero, c.unit, r))
	if r.root.value != c.unit {
		t.Fatal("zero shortcut changed")
	}
	if tags.ShouldTrace("gcd_calls") {
		want := "-------- [gcd_calls] gcd src/math/polynomial/polynomial.cpp:4355 ---------\ngcd\nu: 0\nv: 1\n------------------------------------------------\n" +
			"-------- [polynomial_gcd] gcd src/math/polynomial/polynomial.cpp:4356 ---------\ngcd\nu: 0\nv: 1\nis_zero(u): 1, is_const(u): 1\nis_zero(v): 0, is_const(v): 1\nmodular: 0\n------------------------------------------------\n"
		if out.String() != want {
			t.Fatalf("literal records %q want %q", out.String(), want)
		}
	} else if out.Len() != 0 {
		t.Fatal("TRACE compiled out but wrote output")
	}
	tags.Disable("gcd_calls")
	tags.Disable("polynomial_gcd")
	before := out.String()
	serviceMust(t, m.GCD(c.zero, c.unit, r))
	if out.String() != before {
		t.Fatal("disabled tags evaluated trace")
	}
}
func TestPolynomialGCDEntrySourceVariableRecords(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	u := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	v := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	defer func() { serviceMust(t, m.CleanupGCD()); serviceMust(t, r.Close()) }()
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("polynomial_gcd")
	c.monomials.config.Trace = tags
	defer tags.Finalize()
	serviceMust(t, m.GCD(u, v, r))
	if tags.ShouldTrace("polynomial_gcd") && !strings.Contains(out.String(), "[polynomial_gcd] gcd src/math/polynomial/polynomial.cpp:4390 ---------\nu var info\nx0^2 \nv var info\nx0 \n") {
		t.Fatalf("sorted power records missing %q", out.String())
	}
}

type polynomialGCDEntryWriter struct {
	m      *MultivariatePolynomialManager
	t      *testing.T
	calls  int
	refuse bool
}

func (w *polynomialGCDEntryWriter) Write(p []byte) (int, error) {
	w.calls++
	for _, e := range []error{w.m.CloseChecked(), w.m.SourceFinalize(), w.m.core.CloseChecked(), w.m.core.SourceFinalizePrefix(), w.m.CleanupGCD(), w.m.core.CleanupDisplay(), w.m.Rename([]uint32{})} {
		if !errors.Is(e, ErrPolynomialPending) {
			w.t.Fatalf("entry trace admitted public mutation: %v", e)
		}
	}
	if w.refuse {
		return 0, errors.New("ostream refused")
	}
	return len(p), nil
}
func TestPolynomialGCDEntryTraceCallbackAndRefusal(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		r, e := NewPolynomialRef(c)
		serviceMust(t, e)
		w := &polynomialGCDEntryWriter{m: m, t: t, refuse: refuse}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("gcd_calls")
		tags.Enable("polynomial_gcd")
		c.monomials.config.Trace = tags
		serviceMust(t, m.GCD(c.zero, c.unit, r))
		if tags.ShouldTrace("gcd_calls") && w.calls == 0 {
			t.Fatal("writer never reached")
		}
		if r.root.value != c.unit {
			t.Fatal("refusal changed result")
		}
		tags.Finalize()
		serviceMust(t, r.Close())
	}
}

func TestPolynomialGCDEntryExactNumericReceiptTLS(t *testing.T) {
	m, q, rt, budget := polynomialDisplayTLSOwner(t)
	c := m.core
	z := q.Integers()
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("gcd_calls")
	var big unumeral.Int
	var p Polynomial
	var r *PolynomialRef
	defer func() {
		tags.Finalize()
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupGCD())
		if r != nil {
			serviceMust(t, r.Close())
		}
		if p != (Polynomial{}) {
			serviceMust(t, c.DecRef(p))
		}
		serviceMust(t, z.Del(&big))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Close())
		if budget.Used() != 0 {
			t.Fatal("exact GCD display graph retained storage", budget.Used())
		}
	}()
	if !tags.ShouldTrace("gcd_calls") {
		t.Skip("TRACE compiled out")
	}
	digits := make([]uint32, 40000)
	digits[39999] = 1
	serviceMust(t, z.SetLimbs(&big, digits))
	var e error
	p, e = c.MkConstantTake(&big)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	r, e = NewPolynomialRef(c)
	serviceMust(t, e)
	c.monomials.config.Trace = tags
	serviceMust(t, rt.SetMaxSize(1))
	body := m.GCD(c.zero, p, r)
	serviceMust(t, rt.SetMaxSize(0))
	if !numericPhysicalReservationRefusal(body) || len(m.gcdFrames) == 0 {
		t.Fatal("actual formatter Copy refusal not reached", body)
	}
	scope := m.gcdFrames[0]
	if !scope.active || !c.displayReceiptPending(&scope.traceDisplay) {
		t.Fatal("GCD lost its accepted formatter receipt", body)
	}
	roots, e := z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (umemory.Block{}) || scope.traceDisplay.frame.numeric.Allocator != roots {
		t.Fatal("formatter omitted actual Copy raw", roots)
	}
	serviceMust(t, m.CleanupGCD())
	roots, e = z.Allocator.PendingRoots()
	serviceMust(t, e)
	if roots != ([3]umemory.Block{}) {
		t.Fatal("owning GCD left its formatter raw", roots)
	}
	if scope.active || c.displayReceiptPending(&scope.traceDisplay) {
		t.Fatal("owning GCD left accepted display active")
	}
}
