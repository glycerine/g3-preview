// Source: src/math/polynomial/polynomial.cpp:6436-6550,7409-7414.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

type polynomialEvalIntervalMap struct {
	manager *util.BinaryIntervalManager
	value   util.BinaryInterval
}

func (v *polynomialEvalIntervalMap) Manager() (*util.BinaryIntervalManager, error) {
	return v.manager, nil
}
func (v *polynomialEvalIntervalMap) Contains(x uint32) (bool, error) { return x == 0, nil }
func (v *polynomialEvalIntervalMap) Interval(x uint32) (*util.BinaryInterval, error) {
	if x != 0 {
		return nil, ErrPolynomialAssertion
	}
	return &v.value, nil
}
func TestPolynomialEvaluationModularDomain(t *testing.T) {
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{3, 0, 0}}, {2, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	serviceMust(t, m.core.modular.SetZPUint64(5))
	x, out := util.NewInt(-2), util.Int{}
	serviceMust(t, m.UnivariateEvaluate(p, 0, &x, &out))
	v, e := m.core.integer.Int64(&out)
	serviceMust(t, e)
	if v != -1 {
		t.Fatalf("GF5 3^3+2*3+1 = %d, want balanced -1", v)
	}
	serviceMust(t, m.core.integer.Del(&out))
	serviceMust(t, m.core.modular.SetZ())
}
func TestPolynomialEvaluationIntervalDomain(t *testing.T) {
	b, _ := binaryManager(t)
	v, e := util.NewBinaryIntervalManager(b)
	serviceMust(t, e)
	t.Cleanup(v.Close)
	ctl := &polynomialEvalControl{limit: 10000}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: b.Integers(), Checkpoint: ctl.checkpoint, Cooperation: ctl.cooperate}})
	serviceMust(t, e)
	_, e = m.core.monomials.MkVar()
	serviceMust(t, e)
	as := []util.Int{util.NewInt(1), util.NewInt(-1), util.NewInt(1)}
	ms := make([]Monomial, 3)
	ms[0], e = m.core.monomials.MkVariable(0, 2)
	serviceMust(t, e)
	ms[1], e = m.core.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	ms[2], e = m.core.monomials.Unit()
	serviceMust(t, e)
	p, e := m.core.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	t.Cleanup(func() { serviceMust(t, m.core.DecRef(p)); serviceMust(t, m.CloseChecked()) })
	values := polynomialEvalIntervalMap{manager: v}
	intervalSet(t, v, &values.value, -1, 2)
	out := util.NewBinaryInterval()
	serviceMust(t, m.EvaluateInterval(p, &values, &out))
	intervalText(t, v, &out, "(-3, 3)")
	serviceMust(t, v.Del(&values.value))
	serviceMust(t, v.Del(&out))
}
