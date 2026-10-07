// Source: src/math/polynomial/upolynomial.cpp:33-63;
// src/math/polynomial/polynomial.cpp:6683-6725.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactorUPFactorsLiteralSourceClass(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	fs, e := full.upm.NewFactors()
	serviceMust(t, e)
	a := util.NewInt(-7)
	serviceMust(t, fs.setConstantWithSourceContext(&a, integerAllocationInvocationContext{}))
	if fs.constant != a {
		t.Fatal("actual class SET")
	}
	var cursor upolyPreparedFactorCursor
	serviceMust(t, fs.closeWithSourceContext(&cursor, true, integerAllocationInvocationContext{}))
	if !fs.closed || full.upm.liveFactors != 0 {
		t.Fatal("source class header and retain")
	}
}
