// Source: src/util/mpq.cpp; src/math/polynomial/polynomial.cpp:6436-6522.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import "testing"

func TestPolynomialEvaluationRawGroupIdentity(t *testing.T) {
	q, _ := rationalManager(t, false)
	a, out := NewRational(), NewRational()
	serviceMust(t, q.SetFraction(&a, 3, 2))
	serviceMust(t, q.Power(&a, 2, &out))
	first := q.rawArithmetic.powerGeneration
	if first == 0 {
		t.Fatal("completed actual Power has no continuation identity")
	}
	serviceMust(t, q.CleanupRawArithmetic())
	serviceMust(t, q.Power(&a, 2, &out))
	if q.rawArithmetic.powerGeneration <= first {
		t.Fatal("cleanup reused a continuation identity")
	}
	serviceMust(t, q.cleanupRawGroup(rationalRawPower))
	serviceMust(t, q.Del(&a))
	serviceMust(t, q.Del(&out))
}
