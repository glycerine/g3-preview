// Source: src/math/polynomial/polynomial.cpp:2757-2768;
// src/util/mpzzp.h:167-185,195-208; src/util/mpz.h:468-485.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Nil direct context uses the literal existing engine and acquires no authority.
func TestScalarDivDirectNumericNilContextSourceMath(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a, b, out := util.NewInt(-17), util.NewInt(5), util.NewInt(0)
	serviceMust(t, numericIntegerModularManager_divWithScalarDivReceipt(c.modular, &a, &b, &out, nil))
	if out != util.NewInt(-4) {
		t.Fatal("literal Euclidean quotient", out)
	}
	serviceMust(t, numericIntegerModularManager_copyWithScalarDivReceipt(c.modular, &out, &a, nil))
	if out != a {
		t.Fatal("literal SET", out)
	}
	serviceMust(t, c.modular.SetZPUint64(7))
	a, b = util.NewInt(3), util.NewInt(2)
	serviceMust(t, numericIntegerModularManager_divWithScalarDivReceipt(c.modular, &a, &b, &out, nil))
	if out != util.NewInt(-2) {
		t.Fatal("literal finite-field quotient", out)
	}
}
