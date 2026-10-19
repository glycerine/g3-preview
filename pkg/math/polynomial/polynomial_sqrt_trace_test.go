package polynomial

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestPolynomialSqrtSourceTraceAndZeroRoute(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, full.CleanupSqrt()); serviceMust(t, out.Close()) })
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sqrt_bug")
	m.monomials.config.Trace = tags
	ok, e := full.Sqrt(m.zero, out)
	serviceMust(t, e)
	if !ok || output.Len() != 0 {
		t.Fatal("zero route entered square-root trace", ok, output.String())
	}
	ok, e = full.Sqrt(m.unit, out)
	serviceMust(t, e)
	if !ok {
		t.Fatal("unit square root refused")
	}
	want := ""
	if tags.ShouldTrace("sqrt_bug") {
		want = "-------- [sqrt_bug] sqrt src/math/polynomial/polynomial.cpp:5971 ---------\nsqrt:    1\nmin pos: 0\nmax pos: 0\n------------------------------------------------\n" +
			"-------- [sqrt_bug] sqrt src/math/polynomial/polynomial.cpp:6029 ---------\nR: 1*1\nC: \n------------------------------------------------\n"
	}
	if output.String() != want {
		t.Fatalf("source square-root trace: got %q, want %q", output.String(), want)
	}
	output.Reset()
	p := polynomialQueryTerms(t, m, []int32{1, 2, 1}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	ok, e = full.Sqrt(p, out)
	serviceMust(t, e)
	if !ok {
		t.Fatal("nonconstant perfect square refused")
	}
	if tags.ShouldTrace("sqrt_bug") && !strings.Contains(output.String(), "R: 1*x0 + 1*1\n") {
		t.Fatal("source multirow R buffer display", output.String())
	}
	want = output.String()
	tags.Disable("sqrt_bug")
	unread := Polynomial{owner: m.origin, slot: ^uint32(0), generation: 1}
	serviceMust(t, full.traceSqrtEntry(unread))
	if output.String() != want {
		t.Fatal("disabled square-root trace read or emitted an operand")
	}
	tags.Enable("sqrt_bug")
	e = full.traceSqrtEntry(unread)
	if tags.ShouldTrace("sqrt_bug") && e == nil {
		t.Fatal("enabled square-root trace omitted formatting")
	}
	if !tags.ShouldTrace("sqrt_bug") {
		serviceMust(t, e)
	}
}
