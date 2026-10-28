package polynomial

// Source: src/math/polynomial/polynomial.cpp:2757-2768;
// src/util/mpzzp.h:167-185,195-208; src/util/mpz.h:468-485.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// Nil direct context uses the literal existing engine and acquires no authority.
func TestScalarDivDirectNumericNilContextSourceMath(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a, b, out := unumeral.NewInt(-17), unumeral.NewInt(5), unumeral.NewInt(0)
	serviceMust(t, numericIntegerModularManager_divWithScalarDivReceipt(c.modular, &a, &b, &out, nil))
	if out != unumeral.NewInt(-4) {
		t.Fatal("literal Euclidean quotient", out)
	}
	serviceMust(t, numericIntegerModularManager_copyWithScalarDivReceipt(c.modular, &out, &a, nil))
	if out != a {
		t.Fatal("literal SET", out)
	}
	serviceMust(t, c.modular.SetZPUint64(7))
	a, b = unumeral.NewInt(3), unumeral.NewInt(2)
	serviceMust(t, numericIntegerModularManager_divWithScalarDivReceipt(c.modular, &a, &b, &out, nil))
	if out != unumeral.NewInt(-2) {
		t.Fatal("literal finite-field quotient", out)
	}
}
