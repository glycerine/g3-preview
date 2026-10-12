package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanNormalizeQueryAfterNonUnitSolve(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	euclideanAssertSmall(t, s, []int32{2, 3}, []uint32{0, 1}, 4, nullEquationIndex)
	consistent, e := s.solve()
	euclidMust(t, e)
	if !consistent {
		t.Fatal("source fixture is inconsistent")
	}
	aPrime, cPrime := numeral.NewInt(0), numeral.NewInt(0)
	var js []uint32
	one := []numeral.Int{numeral.NewInt(1)}
	zero := numeral.NewInt(0)
	euclidMust(t, s.normalizeQuery(one, []uint32{0}, &zero, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "3", "-2")
	if len(js) != 0 {
		t.Fatalf("unexpected justifications %v", js)
	}
	// Unsorted input IDs follow the source sort before substitution. The
	// asserted polynomial reduces completely to zero.
	as := []numeral.Int{numeral.NewInt(3), numeral.NewInt(2)}
	c := numeral.NewInt(4)
	euclidMust(t, s.normalizeQuery(as, []uint32{1, 0}, &c, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "0", "0")
	if len(js) != 0 {
		t.Fatalf("unexpected justifications %v", js)
	}
	euclidMust(t, s.Close())
	for i := range one {
		euclidMust(t, z.Del(&one[i]))
	}
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	for _, value := range []*numeral.Int{&zero, &c, &aPrime, &cPrime} {
		euclidMust(t, z.Del(value))
	}
}

func TestEuclideanNormalizeQueryJustificationAndConstantOnly(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	_, e = s.mkVar(false)
	euclidMust(t, e)
	j, e := s.mkJustification()
	euclidMust(t, e)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 1, j)
	consistent, e := s.solve()
	euclidMust(t, e)
	if !consistent {
		t.Fatal("unit fixture is inconsistent")
	}
	coefficient, constant := numeral.NewInt(2), numeral.NewInt(3)
	aPrime, cPrime := numeral.NewInt(0), numeral.NewInt(0)
	js := []uint32{99}
	euclidMust(t, s.normalizeQuery([]numeral.Int{coefficient}, []uint32{0},
		&constant, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "0", "1")
	if !equalUint32Slices(js, []uint32{j}) {
		t.Fatalf("justification IDs = %v, want [%d]", js, j)
	}
	// Empty input clears the previous justification result and copies c.
	euclidMust(t, s.normalizeQuery(nil, nil, &constant, &aPrime, &cPrime, &js))
	checkEuclideanQueryResult(t, z, &aPrime, &cPrime, "0", "3")
	if len(js) != 0 {
		t.Fatalf("empty query retained justifications %v", js)
	}
	euclidMust(t, s.Close())
	for _, value := range []*numeral.Int{&coefficient, &constant, &aPrime, &cPrime} {
		euclidMust(t, z.Del(value))
	}
}

func checkEuclideanQueryResult(t *testing.T, z *numeral.IntegerManager, a, c *numeral.Int, wantA, wantC string) {
	t.Helper()
	gotA, e := z.String(a)
	euclidMust(t, e)
	gotC, e := z.String(c)
	euclidMust(t, e)
	if gotA != wantA || gotC != wantC {
		t.Fatalf("normalized query = (%s,%s), want (%s,%s)", gotA, gotC, wantA, wantC)
	}
}
