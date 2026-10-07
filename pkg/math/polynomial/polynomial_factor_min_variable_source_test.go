// Source: src/math/polynomial/polynomial.cpp:6558-6577;
// src/math/polynomial/polynomial.cpp:6307-6344.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"strconv"
	"testing"
)

func TestPolynomialFactorMinimumVariableLiteralEncounterTie(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	m := full.core
	p := polynomialQueryTerms(t, m, []int32{3, 5, 7}, [][]MonomialPower{{{0, 4}, {2, 2}}, {{1, 2}}, {{0, 1}}})
	// Obtain the actual source helper encounter stream, not a sorted variable set.
	d := &m.substitution.max
	serviceMust(t, d.Init(p))
	n, e := d.NumVariables()
	serviceMust(t, e)
	xs, e := d.Vars()
	serviceMust(t, e)
	expected, min := ^uint32(0), ^uint32(0)
	for i := uint32(0); i < n; i++ {
		x, e := xs.At(i)
		serviceMust(t, e)
		degree, e := d.Degree(x)
		serviceMust(t, e)
		if degree < min {
			expected, min = x, degree
		}
	}
	serviceMust(t, d.Reset())
	got, scope, e := full.minimumDegreeVariable(p)
	serviceMust(t, e)
	if scope == nil || !scope.finalized || !scope.closed {
		t.Fatal("actual completed constructor/source DT")
	}
	if got != expected || min != 2 {
		t.Fatal("literal strict comparison and encounter tie", got, expected, min)
	}
	n, e = d.NumVariables()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("scoped var_max_degree DT precedes caller declarations", n)
	}
}

func TestPolynomialFactorMinimumVariableEveryActualConstructorPrefix(t *testing.T) {
	success := false
	for prefix := int64(0); prefix < 32 && !success; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			p := polynomialQueryTerms(t, m, []int32{3, 5, 7}, [][]MonomialPower{{{0, 4}, {2, 2}}, {{1, 2}}, {{0, 1}}})
			m.integer.Budget().FailAfter(prefix)
			x, scope, cause := full.minimumDegreeVariable(p)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
			if cause != nil {
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual scoped maximum degree refusal", cause)
				}
				if scope != nil && !scope.finalized {
					t.Fatal("reservation body failure cannot invent destructor failure")
				}
			} else {
				success = true
				if x != 1 && x != 2 {
					t.Fatal("actual minimum encounter selection", x)
				}
			}
			m.integer.Budget().FailAfter(-1)
			if scope != nil {
				serviceMust(t, scope.CloseChecked())
			}
		})
	}
	if !success {
		t.Fatal("actual first successful scoped constructor never reached")
	}
}
