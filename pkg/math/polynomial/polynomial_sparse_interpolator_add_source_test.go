// Source: src/math/polynomial/polynomial.cpp:3077-3131.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialSparseSourceMissingEntryKeepsInputThenReset(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	serviceMust(t, m.modular.SetZPUint64(97))
	p := polynomialQueryTerms(t, m, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	sk, e := NewPolynomialSkeleton(m, p, 0)
	serviceMust(t, e)
	s, e := NewPolynomialSparseInterpolator(full, sk)
	serviceMust(t, e)
	q := polynomialQueryTerms(t, m, []int32{1}, [][]MonomialPower{{{1, 1}}})
	in := util.NewInt(7)
	ok, e := s.Add(&in, q)
	serviceMust(t, e)
	if ok {
		t.Fatal("actual missing skeleton entry")
	}
	n := len(s.inputs)
	if n != 1 {
		t.Fatal("published input before missing entry", n)
	}
	a := s.inputs[0]
	if a != in {
		t.Fatal("actual copied input", a)
	}
	capBefore := cap(s.inputs)
	serviceMust(t, s.Reset())
	n = len(s.inputs)
	capAfter := cap(s.inputs)
	if n != 0 || capAfter != capBefore {
		t.Fatal("source reset retains capacity", n, capAfter, capBefore)
	}
	serviceMust(t, s.SourceFinalize())
	serviceMust(t, s.CloseChecked())
	serviceMust(t, sk.CloseChecked())
}
