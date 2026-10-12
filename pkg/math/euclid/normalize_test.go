package euclid

import (
	"slices"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanAssertEquationNormalizeAndOccurrences(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	j, e := s.mkJustification()
	euclidMust(t, e)
	as := []numeral.Int{numeral.NewInt(-6), numeral.NewInt(9)}
	c := numeral.NewInt(12)
	euclidMust(t, s.assertEquation(as, []uint32{0, 1}, &c, j))
	eq := s.equations[0]
	if eq == nil || s.inconsistent != nullEquationIndex ||
		!slices.Equal(s.occurrences[0], []uint32{0}) || !slices.Equal(s.occurrences[1], []uint32{0}) {
		t.Fatal("normalized equation publication or occurrence lists")
	}
	for i, want := range []string{"-2", "3"} {
		got, e := z.String(&eq.as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatalf("coefficient[%d] = %s, want %s", i, got, want)
		}
	}
	// Source normalize_eq divides coefficients and justifications, while the
	// constant is only checked for divisibility and retains its original value.
	constant, e := z.String(&eq.c)
	euclidMust(t, e)
	b, e := m.String(&eq.bs[0])
	euclidMust(t, e)
	if constant != "12" || b != "1/3" {
		t.Fatalf("source normalization constant=%s justification=%s", constant, b)
	}
	bad := []numeral.Int{numeral.NewInt(4)}
	badC := numeral.NewInt(6)
	euclidMust(t, s.assertEquation(bad, []uint32{0}, &badC, nullEquationIndex))
	if s.inconsistent != 1 || s.equations[1] == nil || !slices.Equal(s.occurrences[0], []uint32{0, 1}) {
		t.Fatal("gcd divisibility contradiction or source occurrence publication")
	}
	before := len(s.equations)
	euclidMust(t, s.assertEquation(bad, []uint32{0}, &badC, nullEquationIndex))
	if len(s.equations) != before {
		t.Fatal("assertion proceeded after inconsistency")
	}
	euclidMust(t, s.Close())
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	euclidMust(t, z.Del(&c))
	euclidMust(t, z.Del(&bad[0]))
	euclidMust(t, z.Del(&badC))
}

func TestEuclideanNormalizeUnitGCDPreservesEquation(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	as := []numeral.Int{numeral.NewInt(-2), numeral.NewInt(3)}
	c := numeral.NewInt(5)
	euclidMust(t, s.assertEquation(as, []uint32{0, 1}, &c, nullEquationIndex))
	if s.inconsistent != nullEquationIndex || !slices.Equal(s.equations[0].xs, []uint32{0, 1}) {
		t.Fatal("unit gcd changed equation")
	}
	for i, want := range []string{"-2", "3"} {
		got, e := z.String(&s.equations[0].as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatalf("unit-gcd coefficient[%d] = %s", i, got)
		}
	}
	euclidMust(t, s.Close())
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	euclidMust(t, z.Del(&c))
}

func TestEuclideanConstantOnlyTrivialAndContradictoryEquations(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	zero := numeral.NewInt(0)
	euclidMust(t, s.assertEquation(nil, nil, &zero, nullEquationIndex))
	if s.equations[0] != nil || s.inconsistent != nullEquationIndex {
		t.Fatal("zero equality was not discarded")
	}
	two := numeral.NewInt(2)
	euclidMust(t, s.assertEquation(nil, nil, &two, nullEquationIndex))
	if s.equations[1] == nil || s.inconsistent != 1 {
		t.Fatal("constant contradiction was not retained")
	}
	euclidMust(t, s.Close())
	euclidMust(t, m.Integers().Del(&zero))
	euclidMust(t, m.Integers().Del(&two))
}

func TestEuclideanAssertEquationSubstitutesThenNormalizes(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
		_, e = s.mkJustification()
		euclidMust(t, e)
	}
	solAs := []numeral.Int{numeral.NewInt(-1), numeral.NewInt(1)}
	solC := numeral.NewInt(1)
	solB := []numeral.Rational{numeral.NewRational()}
	euclidMust(t, m.SetString(&solB[0], "1"))
	_, e = s.addEquation(solAs, []uint32{0, 1}, &solC, solB, []uint32{0}, true)
	euclidMust(t, e)
	euclidMust(t, s.promoteSolvedEquation(0, 0))
	as := []numeral.Int{numeral.NewInt(2)}
	c := numeral.NewInt(2)
	euclidMust(t, s.assertEquation(as, []uint32{0}, &c, 1))
	eq := s.equations[1]
	if eq == nil || s.inconsistent != nullEquationIndex || !slices.Equal(eq.xs, []uint32{1}) ||
		!slices.Equal(eq.js, []uint32{0, 1}) || !slices.Equal(s.occurrences[1], []uint32{1}) {
		t.Fatal("assertion did not substitute, normalize and publish occurrences")
	}
	gotA, e := z.String(&eq.as[0])
	euclidMust(t, e)
	gotC, e := z.String(&eq.c)
	euclidMust(t, e)
	b0, e := m.String(&eq.bs[0])
	euclidMust(t, e)
	b1, e := m.String(&eq.bs[1])
	euclidMust(t, e)
	if gotA != "1" || gotC != "4" || b0 != "1" || b1 != "1/2" {
		t.Fatalf("asserted equation = %s*x1+%s; justifications %s,%s", gotA, gotC, b0, b1)
	}
	euclidMust(t, s.Close())
	for i := range solAs {
		euclidMust(t, z.Del(&solAs[i]))
	}
	euclidMust(t, z.Del(&solC))
	euclidMust(t, m.Del(&solB[0]))
	euclidMust(t, z.Del(&as[0]))
	euclidMust(t, z.Del(&c))
}
