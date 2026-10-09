// Source: src/math/polynomial/polynomial.cpp:2217-2245,3168-3180.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialTermsIndexedNativeSliceAndResetBeforeRead(t *testing.T) {
	m := polynomialCellFixture(t)
	var as polynomialVector
	as.width = 12
	serviceMust(t, as.appendCoefficient(m, util.NewInt(3)))
	unit, e := m.monomials.Unit()
	serviceMust(t, e)
	values := []Monomial{unit}
	p, e := m.mkTermsIndexed(&as, monomialArrayView{values: values, count: 1})
	serviceMust(t, e)
	p = polynomialCaller(t, m, p)
	a, e := m.coefficientCore(p, 0)
	serviceMust(t, e)
	if a != util.NewInt(3) {
		t.Fatal("actual scalar coefficient", a)
	}
	x := util.NewInt(1)
	serviceMust(t, m.som.add(&x, unit, false))
	_, e = m.mkTermsIndexed(&as, monomialArrayView{count: 0})
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached missing mono", e)
	}
	n, e := m.som.as.size(m)
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("source SOM reset before failed read", n)
	}
	serviceMust(t, as.close(m))
}

func TestPolynomialTermsIndexedNativeSliceAndUnreachedTail(t *testing.T) {
	m := polynomialCellFixture(t)
	var as polynomialVector
	as.width = 12
	serviceMust(t, as.appendCoefficient(m, util.NewInt(5)))
	unit, e := m.monomials.Unit()
	serviceMust(t, e)
	view := monomialArrayView{values: []Monomial{unit}, count: 19}
	p, e := m.mkTermsIndexed(&as, view)
	serviceMust(t, e)
	p = polynomialCaller(t, m, p)
	a, e := m.coefficientCore(p, 0)
	serviceMust(t, e)
	if a != util.NewInt(5) {
		t.Fatal("actual borrowed monomial slice", a)
	}
	serviceMust(t, as.reset(m))
	empty, e := m.mkTermsIndexed(&as, monomialArrayView{count: 19})
	serviceMust(t, e)
	_ = polynomialCaller(t, m, empty)
	serviceMust(t, as.appendCoefficient(m, util.NewInt(1)))
	view.values = nil
	_, e = m.mkTermsIndexed(&as, view)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached native mono boundary", e)
	}
	serviceMust(t, as.close(m))
}

// Compare actual native handle spans after a guarded call.
func polynomialEqualMonomials(a, b []Monomial) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
