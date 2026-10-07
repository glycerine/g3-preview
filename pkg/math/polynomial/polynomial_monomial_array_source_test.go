// Source: src/math/polynomial/polynomial.cpp:2217-2245,3168-3180.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialTermsIndexedActualInlineAndResetBeforeRead(t *testing.T) {
	m := polynomialCellFixture(t)
	var as polynomialVector
	as.width = 12
	serviceMust(t, as.appendCoefficient(m, util.NewInt(3)))
	unit, e := m.monomials.Unit()
	serviceMust(t, e)
	var inline [128]Monomial
	inline[0] = unit
	p, e := m.mkTermsIndexed(&as, monomialArrayView{inline: &inline, count: 1})
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

func TestPolynomialTermsIndexedActualRuntimeAndUnreachedTail(t *testing.T) {
	m := polynomialCellFixture(t)
	var as polynomialVector
	as.width = 12
	serviceMust(t, as.appendCoefficient(m, util.NewInt(5)))
	unit, e := m.monomials.Unit()
	serviceMust(t, e)
	raw, e := m.memory.Allocate(16)
	serviceMust(t, e)
	serviceMust(t, skeletonWriteMonomial(m.memory, raw, 0, unit))
	view := monomialArrayView{runtime: m.memory, raw: raw, count: 19, stride: 16}
	p, e := m.mkTermsIndexed(&as, view)
	serviceMust(t, e)
	p = polynomialCaller(t, m, p)
	a, e := m.coefficientCore(p, 0)
	serviceMust(t, e)
	if a != util.NewInt(5) {
		t.Fatal("actual Runtime pointer span", a)
	}
	serviceMust(t, as.reset(m))
	empty, e := m.mkTermsIndexed(&as, monomialArrayView{runtime: &util.Runtime{}, count: 19})
	serviceMust(t, e)
	_ = polynomialCaller(t, m, empty)
	serviceMust(t, as.appendCoefficient(m, util.NewInt(1)))
	view.offset = 16
	_, e = m.mkTermsIndexed(&as, view)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached physical mono boundary", e)
	}
	serviceMust(t, as.close(m))
	serviceMust(t, m.memory.Deallocate(&raw))
}
