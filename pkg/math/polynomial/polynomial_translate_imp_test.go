// Source: src/math/polynomial/polynomial.cpp:2303-2370,3282-3291,6865-6895,7392-7398;
// src/math/polynomial/polynomial.h:998-1000. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// Exercise actual imp members together: translation retains its source vector
// capacity and lex sorting retains its recursive buckets until the owner closes.
func TestPolynomialTranslateImpEveryReservationAndLexSort(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, _ := rationalManager(t, false)
			full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
			serviceMust(t, e)
			m := full.core
			for i := 0; i < 2; i++ {
				_, e = m.monomials.MkVar()
				serviceMust(t, e)
			}
			terms := make([]Monomial, 3)
			for i, powers := range [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}, {1, 1}}, nil} {
				terms[i], e = m.monomials.MkMonomial(powers)
				serviceMust(t, e)
			}
			p, e := m.MkTerms([]util.Int{util.NewInt(1), util.NewInt(3), util.NewInt(5)}, terms)
			serviceMust(t, e)
			serviceMust(t, m.IncRef(p))
			shift := util.NewInt(2)
			m.integer.Budget().FailAfter(ordinal)
			out, e := full.Translate(p, 0, &shift)
			named := []Polynomial{p}
			if out != (Polynomial{}) {
				named = append(named, out)
			}
			polynomialImpPhysicalAuditValues(t, full, named)
			if m.translation.active {
				stats, used := m.memory.Stats(), m.integer.Budget().Used()
				if closeErr := full.CloseChecked(); !errors.Is(closeErr, ErrPolynomialPending) {
					t.Fatal("retained translation locals must guard owner close", closeErr)
				}
				if full.closing || m.memory.Stats() != stats || m.integer.Budget().Used() != used {
					t.Fatal("close mutated retained translation before its guard")
				}
			}
			m.integer.Budget().FailAfter(-1)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual reservation category", e)
				}
				serviceMust(t, m.CleanupTranslation())
			} else {
				success = true
				serviceMust(t, m.IncRef(out))
				serviceMust(t, full.LexSort(out))
				polynomialQueryExpect(t, m, out, map[[3]uint32]int64{{2, 1, 0}: 1, {1, 1, 0}: 7, {0, 1, 0}: 10, {0, 0, 0}: 5})
				polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, out})
				serviceMust(t, m.DecRef(out))
			}
			polynomialImpPhysicalAuditValues(t, full, []Polynomial{p})
			serviceMust(t, m.DecRef(p))
			serviceMust(t, full.CloseChecked())
		})
	}
	if !success {
		t.Fatal("full imp reservation campaign did not reach success")
	}
}

func TestPolynomialTranslateImpArrayAndVectorFacade(t *testing.T) {
	full, p, _ := polynomialLexFixture(t)
	m := full.core
	visits, cooperations := 0, 0
	m.config.Checkpoint = func() bool { visits++; return true }
	m.config.Cooperation = func() error { cooperations++; return nil }
	xs, e := NewPolynomialVariableDegrees(m.memory)
	serviceMust(t, e)
	defer func() { serviceMust(t, xs.CloseChecked()) }()
	serviceMust(t, xs.SetDegree(0, 0))
	out, e := NewPolynomialRef(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, out.Close()) }()
	shift := []util.Int{util.NewInt(2)}
	serviceMust(t, full.TranslateMany(p, 1, variableView(xs), shift, out))
	want := map[[3]uint32]int64{{1, 2, 0}: 11, {0, 2, 0}: 22, {2, 0, 0}: 13, {1, 0, 0}: 52, {0, 0, 0}: 75, {0, 3, 0}: 17, {1, 1, 0}: 19, {0, 1, 0}: 38}
	polynomialQueryExpect(t, m, out.root.value, want)
	serviceMust(t, full.TranslateVariables(p, xs, shift, out))
	polynomialQueryExpect(t, m, out.root.value, want)
	serviceMust(t, full.TranslateMany(p, 0, PolynomialVariablesView{}, nil, out))
	if out.root.value != p {
		t.Fatal("public array initial assignment")
	}
	if visits != 4 || cooperations != 4 {
		t.Fatal("actual two degree-two translations and zero-count short circuit", visits, cooperations)
	}
}
