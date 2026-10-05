// Source: src/math/polynomial/polynomial.cpp:2757-2768;
// src/util/mpzzp.h:167-185,195-208; src/util/mpz.h:468-485.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import "testing"

// Nil direct context uses the literal existing engine and acquires no authority.
func TestScalarDivDirectNumericNilContextSourceMath(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a, b, out := NewInt(-17), NewInt(5), NewInt(0)
	serviceMust(t, c.modular.divWithScalarDivReceipt(&a, &b, &out, nil))
	if out != NewInt(-4) {
		t.Fatal("literal Euclidean quotient", out)
	}
	serviceMust(t, c.modular.copyWithScalarDivReceipt(&out, &a, nil))
	if out != a {
		t.Fatal("literal SET", out)
	}
	serviceMust(t, c.modular.SetZPUint64(7))
	a, b = NewInt(3), NewInt(2)
	serviceMust(t, c.modular.divWithScalarDivReceipt(&a, &b, &out, nil))
	if out != NewInt(-2) {
		t.Fatal("literal finite-field quotient", out)
	}
}
