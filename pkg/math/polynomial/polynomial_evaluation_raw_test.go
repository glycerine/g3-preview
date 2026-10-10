// Source: src/util/mpq.cpp; src/math/polynomial/polynomial.cpp:6436-6522.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialEvaluationRawGroupIdentity(t *testing.T) {
	q, _ := rationalManager(t, false)
	a, out := unumeral.NewRational(), unumeral.NewRational()
	serviceMust(t, q.SetFraction(&a, 3, 2))
	serviceMust(t, q.Power(&a, 2, &out))
	first := q.RawArithmetic.PowerGeneration
	if first == 0 {
		t.Fatal("completed actual Power has no continuation identity")
	}
	serviceMust(t, q.CleanupRawArithmetic())
	serviceMust(t, q.Power(&a, 2, &out))
	if q.RawArithmetic.PowerGeneration <= first {
		t.Fatal("cleanup reused a continuation identity")
	}
	serviceMust(t, q.CleanupRawGroup(unumeral.RationalRawPower))
	serviceMust(t, q.Del(&a))
	serviceMust(t, q.Del(&out))
}
