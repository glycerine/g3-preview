package polynomial

import (
	"errors"
	"testing"
)

func TestPolynomialReferenceVectorSourceFinalizeStoppedDEC(t *testing.T) {
	m := polynomialCellFixture(t)
	v, err := NewPolynomialRefVector(m)
	serviceMust(t, err)
	t.Cleanup(func() { serviceMust(t, v.Close()) })
	var ps []Polynomial
	for _, k := range []uint32{2, 3} {
		p, err := m.MkVariable(0, k)
		serviceMust(t, err)
		ps = append(ps, p)
		serviceMust(t, v.Push(p))
	}
	stop := errors.New("stop actual lexical vector DEC")
	var order []Polynomial
	h, err := NewPolynomialDeletionHandler(m.integer.Budget(), func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
		order = append(order, p)
		if p == ps[0] {
			return stop
		}
		return nil
	})
	serviceMust(t, err)
	serviceMust(t, m.AddDeletionHandler(h))
	t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) })
	err = v.SourceFinalize()
	if !errors.Is(err, stop) || v.state.drain != 0 || !v.state.drainDec || len(order) != 1 {
		t.Fatal("actual first consumed vector DEC not established", err)
	}
	err = v.SourceFinalize()
	if !errors.Is(err, stop) || v.state.drain != 0 || !v.state.drainDec || len(order) != 1 {
		t.Fatal("source destructor adopted stopped owning DEC", err, order)
	}
	serviceMust(t, v.Close())
	if len(order) != 2 || order[0] != ps[0] || order[1] != ps[1] {
		t.Fatal("owning vector continuation replayed DEC or order", order)
	}
}
