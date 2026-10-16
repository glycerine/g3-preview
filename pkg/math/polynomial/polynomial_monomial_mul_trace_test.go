// Source: src/math/polynomial/polynomial.cpp:960-985; src/util/trace.h:40-46.
package polynomial

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestMonomialMultiplySourceTraceBeforeAndAfterIntern(t *testing.T) {
	m := monomialFixture(t)
	for i := 0; i < 4; i++ {
		_, err := m.MkVar()
		serviceMust(t, err)
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("monomial_mul_bug")
	m.config.Trace = tags
	left := []MonomialPower{{0, 2}, {2, 3}}
	right := []MonomialPower{{1, 5}, {2, 4}, {3, 1}}
	r, err := m.MulPowers(left, right)
	serviceMust(t, err)
	serviceMust(t, m.IncRef(r))
	t.Cleanup(func() { serviceMust(t, m.DecRef(r)) })
	monomialEqualPowers(t, m, r, []MonomialPower{{0, 2}, {1, 5}, {2, 7}, {3, 1}})
	a := monomialOwn(t, m, left)
	b := monomialOwn(t, m, right)
	again, err := m.Mul(a, b)
	serviceMust(t, err)
	if again != r {
		t.Fatal("owned multiplication did not intern the same result", again, r)
	}
	unit, err := m.Unit()
	serviceMust(t, err)
	unchanged, err := m.Mul(unit, a)
	serviceMust(t, err)
	if unchanged != a {
		t.Fatal("unit shortcut", unchanged, a)
	}
	if !tags.ShouldTrace("monomial_mul_bug") {
		if output.Len() != 0 {
			t.Fatal("erased source trace", output.String())
		}
		return
	}
	record := func(line int, body string) string {
		label := "960"
		if line == 967 {
			label = "967"
		}
		return "-------- [monomial_mul_bug] mul src/math/polynomial/polynomial.cpp:" + label + " ---------\n" + body + "------------------------------------------------\n"
	}
	before := record(960, "before mk_monomial\npws1: x0^2 x2^3 \npws2: x1^5 x2^4 x3 \nproduct_tmp: x0^2 x1^5 x2^7 x3 \n")
	after := record(967, "j: 4\nr: x0^2 x1^5 x2^7 x3\npws1: x0^2 x2^3 \npws2: x1^5 x2^4 x3 \nproduct_tmp: x0^2 x1^5 x2^7 x3 \n")
	want := before + after + before + after
	if got := output.String(); got != want {
		t.Fatalf("source multiply records: got %q, want %q", got, want)
	}
	tags.Disable("monomial_mul_bug")
	_, err = m.MulPowers(left, right)
	serviceMust(t, err)
	if got := output.String(); got != want {
		t.Fatalf("disabled multiply trace: got %q", got)
	}
}
