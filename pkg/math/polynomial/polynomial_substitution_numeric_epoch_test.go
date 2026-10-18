package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

type substitutionEpochValues struct {
	t                        *testing.T
	core                     *MultivariatePolynomialCore
	value                    unumeral.Rational
	old                      unumeral.NumericAllocationContext
	invocation               uint64
	captured, probe, checked bool
}

func (v *substitutionEpochValues) Contains(x uint32) (bool, error) {
	c := v.core
	s := &c.substitution.scope
	n := &s.powerNumeric
	if s.executing && n.owner == c {
		if !v.probe && !v.captured {
			v.old = (unumeral.NumericAllocationContext{}).WithCaller(&n.hooks)
			v.invocation = s.invocation
			v.captured = true
		}
		if v.probe && !v.checked {
			v.checked = true
			if s.invocation == v.invocation {
				v.t.Error("replacement did not enter a new substitution")
			}
			a, b, out := unumeral.NewInt(2), unumeral.NewInt(3), unumeral.NewInt(99)
			state, used := c.memory.State(), c.integer.Budget().Used()
			e := c.integer.WithAllocationContext(v.old).Add(&a, &b, &out)
			if !errors.Is(e, unumeral.ErrNumericOwner) || out != unumeral.NewInt(99) || state != c.memory.State() || used != c.integer.Budget().Used() {
				v.t.Error("stale numeric context acquired replacement substitution", e, out)
			}
			current := (unumeral.NumericAllocationContext{}).WithCaller(&n.hooks)
			if e = c.integer.WithAllocationContext(current).Add(&a, &b, &out); e != nil || out != unumeral.NewInt(5) {
				v.t.Error("actual current context refused valid arithmetic", e, out)
			}
		}
	}
	return x == 0, nil
}
func (v *substitutionEpochValues) Rational(x uint32) (*unumeral.Rational, error) {
	return &v.value, nil
}

func TestPolynomialSubstitutionNumericContextRejectsReplacementEpoch(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1, 1}, [][]MonomialPower{{{0, 2}, {1, 1}}, {{0, 1}, {1, 1}}, nil})
	values := substitutionEpochValues{t: t, core: c, value: unumeral.RationalFromBorrowedNumerals(unumeral.NewInt(2), unumeral.NewInt(1))}
	first, e := c.SubstituteRationalMap(p, &values)
	serviceMust(t, e)
	first = polynomialCaller(t, c, first)
	if !values.captured {
		t.Fatal("actual first power context not observed")
	}
	values.probe = true
	second, e := c.SubstituteRationalMap(p, &values)
	serviceMust(t, e)
	second = polynomialCaller(t, c, second)
	if !values.checked {
		t.Fatal("actual replacement power context not observed")
	}
	for _, result := range []Polynomial{first, second} {
		polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{0, 1, 0}: 6, {}: 1})
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, first, second}, nil, 0)
}
