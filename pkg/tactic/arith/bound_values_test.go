package arith

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestBoundEquationStoreExactBoundsRoundingConflictAndPolynomial(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(1, true))
	equationMust(t, s.MkVar(2, false))
	equationMust(t, s.MkVar(3, false))

	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	set := func(text string) { t.Helper(); equationMust(t, m.SetString(&value, text)) }
	assert := func(x uint32, lower bool, strict bool, want bool) {
		t.Helper()
		var got bool
		var e error
		if lower {
			got, e = s.AssertLower(x, &value, strict)
		} else {
			got, e = s.AssertUpper(x, &value, strict)
		}
		equationMust(t, e)
		if got != want {
			t.Fatalf("assert bound on x%d accepted=%v, want %v", x, got, want)
		}
	}
	set("3/2")
	assert(1, true, false, true) // integer lower rounds up to 2
	set("9/2")
	assert(1, false, false, true) // integer upper rounds down to 4
	set("1/2")
	assert(2, true, true, true)
	set("3/2")
	assert(2, false, false, true)

	out := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&out)) }()
	found, strict, ts, err := s.Lower(1, &out)
	equationMust(t, err)
	got, err := m.String(&out)
	equationMust(t, err)
	if !found || strict || ts != 0 || got != "2" {
		t.Fatalf("rounded lower = (%v,%v,%d,%s)", found, strict, ts, got)
	}
	coeff := []numeral.Rational{numeral.NewRational(), numeral.NewRational()}
	defer func() {
		for i := range coeff {
			equationMust(t, m.Del(&coeff[i]))
		}
	}()
	equationMust(t, m.SetInt32(&coeff[0], 2))
	equationMust(t, m.SetInt32(&coeff[1], -3))
	found, strict, err = s.PolynomialBound(coeff, []uint32{1, 2}, true, &out)
	equationMust(t, err)
	got, err = m.String(&out)
	equationMust(t, err)
	if !found || strict || got != "-1/2" {
		t.Fatalf("polynomial lower = (%v,%v,%s)", found, strict, got)
	}
	found, strict, err = s.PolynomialBound(coeff, []uint32{1, 2}, false, &out)
	equationMust(t, err)
	got, err = m.String(&out)
	equationMust(t, err)
	if !found || !strict || got != "13/2" {
		t.Fatalf("polynomial upper = (%v,%v,%s)", found, strict, got)
	}
	found, _, err = s.PolynomialBound(coeff, []uint32{1, 3}, false, &out)
	equationMust(t, err)
	got, err = m.String(&out)
	equationMust(t, err)
	if found || got != "0" {
		t.Fatalf("missing polynomial upper = (%v,%s)", found, got)
	}

	set("2")
	assert(1, true, true, true) // strict integer lower advances to 3
	set("4")
	assert(1, false, true, true) // strict integer upper retreats to 3
	assert(1, false, true, false)
	conflict, err := s.ConflictVar()
	equationMust(t, err)
	if conflict != noEquationPosition {
		t.Fatalf("equal inclusive integer bounds conflict: %d", conflict)
	}
	set("4")
	assert(1, true, false, true)
	conflict, err = s.ConflictVar()
	equationMust(t, err)
	if conflict != 1 {
		t.Fatalf("expected conflict at x1, got %d", conflict)
	}
}

func TestBoundEquationStoreRationalStrictTieAndRelaxedApproximation(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(4, false))
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	equationMust(t, m.SetString(&value, "1/3"))
	accepted, err := s.AssertLower(4, &value, false)
	equationMust(t, err)
	if !accepted || math.Abs(s.variables[4].lower.approx-0.333333) > 1e-12 {
		t.Fatalf("lower relaxed approximation = %g", s.variables[4].lower.approx)
	}
	accepted, err = s.AssertLower(4, &value, true)
	equationMust(t, err)
	if !accepted || !s.variables[4].lower.strict || s.variables[4].lower.prev == nil {
		t.Fatal("strict tie did not replace nonstrict lower")
	}
	accepted, err = s.AssertUpper(4, &value, false)
	equationMust(t, err)
	if !accepted || math.Abs(s.variables[4].upper.approx-0.333334) > 1e-12 {
		t.Fatalf("upper relaxed approximation = %g", s.variables[4].upper.approx)
	}
	conflict, err := s.ConflictVar()
	equationMust(t, err)
	if conflict != 4 {
		t.Fatalf("strict equality must conflict at x4, got %d", conflict)
	}
}
