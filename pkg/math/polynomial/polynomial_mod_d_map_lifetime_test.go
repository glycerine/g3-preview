package polynomial

// Source: src/math/polynomial/polynomial.cpp:6898-6922; src/util/vector.h:191-203.

import "testing"

func TestPolynomialModDConstantWithNativeMapCopy(t *testing.T) {
	m := polynomialCellFixture(t)
	target, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	source, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, target.CloseChecked())
		serviceMust(t, source.CloseChecked())
	})
	serviceMust(t, target.SetDegree(1, 2))
	serviceMust(t, source.SetDegree(1, 3))
	serviceMust(t, target.CopyFrom(source))
	serviceMust(t, source.SetDegree(1, 4))
	v, e := target.Degree(1)
	serviceMust(t, e)
	if v != 3 {
		t.Fatal("native map copy shares backing", v)
	}
	got, e := m.ModD(m.unit, target)
	serviceMust(t, e)
	if got != m.unit {
		t.Fatal("constant source branch changed", got)
	}
}
