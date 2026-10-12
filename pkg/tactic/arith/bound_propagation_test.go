package arith

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestBoundEquationImmediateAndWatchedPropagation(t *testing.T) {
	for _, beforeEquation := range []bool{true, false} {
		t.Run(map[bool]string{true: "immediate", false: "watched"}[beforeEquation], func(t *testing.T) {
			m, _ := equationManager(t)
			s, err := NewBoundEquationStore(m)
			equationMust(t, err)
			defer func() { equationMust(t, s.Close()) }()
			equationMust(t, s.MkVar(0, true))
			equationMust(t, s.MkVar(1, true))
			value := numeral.NewRational()
			defer func() { equationMust(t, m.Del(&value)) }()
			assertInput := func() {
				t.Helper()
				equationMust(t, m.SetInt32(&value, 2))
				accepted, err := s.AssertLower(0, &value, false)
				equationMust(t, err)
				if !accepted {
					t.Fatal("source lower was not accepted")
				}
				equationMust(t, m.SetInt32(&value, 4))
				accepted, err = s.AssertUpper(0, &value, false)
				equationMust(t, err)
				if !accepted {
					t.Fatal("source upper was not accepted")
				}
			}
			if beforeEquation {
				assertInput()
			}
			coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
			ids := []uint32{0, 1}
			_, err = s.MkIntEquation(coeff, ids)
			equationMust(t, err)
			for i := range coeff {
				equationMust(t, m.Integers().Del(&coeff[i]))
			}
			if !beforeEquation {
				assertInput()
				equationMust(t, s.Propagate())
			}
			out := numeral.NewRational()
			defer func() { equationMust(t, m.Del(&out)) }()
			for _, tc := range []struct {
				lower bool
				want  string
			}{{true, "-4"}, {false, "-2"}} {
				var found bool
				if tc.lower {
					found, _, _, err = s.Lower(1, &out)
				} else {
					found, _, _, err = s.Upper(1, &out)
				}
				equationMust(t, err)
				got, err := m.String(&out)
				equationMust(t, err)
				if !found || got != tc.want {
					t.Fatalf("x1 lower=%v = %q, want %q", tc.lower, got, tc.want)
				}
			}
			if conflict, err := s.ConflictVar(); err != nil || conflict != noEquationPosition {
				t.Fatalf("unexpected conflict %d, %v", conflict, err)
			}
			equationMust(t, s.Propagate()) // repeat pass does not add duplicate bounds
			if len(s.trail) != 4 {
				t.Fatalf("repeat propagation appended %d bounds", len(s.trail))
			}
		})
	}
}

func TestBoundEquationNegativeCoefficientAndDeadConstraint(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(0, true))
	equationMust(t, s.MkVar(1, true))
	coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(-2)}
	ids := []uint32{0, 1}
	index, err := s.MkIntEquation(coeff, ids)
	equationMust(t, err)
	for i := range coeff {
		equationMust(t, m.Integers().Del(&coeff[i]))
	}
	value := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&value)) }()
	equationMust(t, m.SetInt32(&value, 4))
	_, err = s.AssertLower(0, &value, false)
	equationMust(t, err)
	_, err = s.AssertUpper(0, &value, false)
	equationMust(t, err)
	equationMust(t, s.Propagate())
	out := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&out)) }()
	found, _, _, err := s.Lower(1, &out)
	equationMust(t, err)
	got, err := m.String(&out)
	equationMust(t, err)
	if !found || got != "2" {
		t.Fatalf("negative coefficient lower = (%v,%s)", found, got)
	}
	equationMust(t, s.DelVar(0))
	_, dead, err := s.Constraint(index)
	equationMust(t, err)
	if !dead {
		t.Fatal("deleted variable did not kill constraint")
	}
	equationMust(t, s.MkVar(0, true))
	equationMust(t, m.SetInt32(&value, 6))
	_, err = s.AssertLower(0, &value, false)
	equationMust(t, err)
	equationMust(t, s.Propagate())
	if len(s.trail) != 5 {
		t.Fatalf("dead constraint propagated: %d bounds", len(s.trail))
	}
	found, _, _, err = s.Upper(1, &out)
	equationMust(t, err)
	got, err = m.String(&out)
	equationMust(t, err)
	if !found || got != "2" {
		t.Fatalf("dead constraint changed x1 upper: (%v,%s)", found, got)
	}
}

func TestBoundEquationFullyBoundedTighteningAndStrictness(t *testing.T) {
	m, _ := equationManager(t)
	s, err := NewBoundEquationStore(m)
	equationMust(t, err)
	defer func() { equationMust(t, s.Close()) }()
	equationMust(t, s.MkVar(0, true))
	equationMust(t, s.MkVar(1, true))
	coeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
	ids := []uint32{0, 1}
	_, err = s.MkIntEquation(coeff, ids) // two missing bounds: no derived bound
	equationMust(t, err)
	for i := range coeff {
		equationMust(t, m.Integers().Del(&coeff[i]))
	}
	if len(s.trail) != 0 {
		t.Fatalf("unbounded equation published %d bounds", len(s.trail))
	}
	input := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&input)) }()
	for _, tc := range []struct {
		x     uint32
		lower bool
		value int32
	}{{0, true, 2}, {0, false, 4}, {1, true, -10}, {1, false, 10}} {
		equationMust(t, m.SetInt32(&input, tc.value))
		if tc.lower {
			_, err = s.AssertLower(tc.x, &input, false)
		} else {
			_, err = s.AssertUpper(tc.x, &input, false)
		}
		equationMust(t, err)
	}
	equationMust(t, s.Propagate())
	out := numeral.NewRational()
	defer func() { equationMust(t, m.Del(&out)) }()
	for _, tc := range []struct {
		lower bool
		want  string
	}{{true, "-4"}, {false, "-2"}} {
		if tc.lower {
			_, _, _, err = s.Lower(1, &out)
		} else {
			_, _, _, err = s.Upper(1, &out)
		}
		equationMust(t, err)
		got, err := m.String(&out)
		equationMust(t, err)
		if got != tc.want {
			t.Fatalf("fully bounded x1 lower=%v = %s", tc.lower, got)
		}
	}
	if len(s.trail) != 6 {
		t.Fatalf("fully bounded propagation published %d values", len(s.trail))
	}

	// A new noninteger variable exposes strictness without integer rounding.
	equationMust(t, s.MkVar(2, false))
	equationMust(t, s.MkVar(3, false))
	strictCoeff := []numeral.Int{numeral.NewInt(1), numeral.NewInt(1)}
	strictIDs := []uint32{2, 3}
	_, err = s.MkIntEquation(strictCoeff, strictIDs)
	equationMust(t, err)
	for i := range strictCoeff {
		equationMust(t, m.Integers().Del(&strictCoeff[i]))
	}
	equationMust(t, m.SetInt32(&input, 2))
	_, err = s.AssertLower(2, &input, true)
	equationMust(t, err)
	equationMust(t, s.Propagate())
	found, strict, _, err := s.Upper(3, &out)
	equationMust(t, err)
	got, err := m.String(&out)
	equationMust(t, err)
	if !found || !strict || got != "-2" {
		t.Fatalf("strict propagated upper = (%v,%v,%s)", found, strict, got)
	}
}
