package euclid

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanNormalizeQueryCopyRefusalRetainsOwnedBuffer(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e = s.mkVar(false)
		euclidMust(t, e)
	}
	value := strings.Repeat("1234567890", 20)
	coeff := []numeral.Int{{}, {}}
	for i := range coeff {
		euclidMust(t, z.SetString(&coeff[i], value))
	}
	constant, aPrime, cPrime := numeral.NewInt(7), numeral.NewInt(0), numeral.NewInt(0)
	var js []uint32
	z.Budget().FailAfter(2)
	e = s.normalizeQuery(coeff, []uint32{1, 0}, &constant, &aPrime, &cPrime, &js)
	z.Budget().FailAfter(-1)
	if e != nil {
		var failure umemory.Failure
		if !errors.As(e, &failure) || failure.Kind != umemory.FailureMemout {
			t.Fatal("unexpected normalization refusal", e)
		}
		if s.normRow == nil || s.stop == nil {
			t.Fatal("stopped normalization lost its owning workspace")
		}
		pending := s.normRow
		if retry := s.normalizeQuery(coeff, []uint32{1, 0}, &constant, &aPrime, &cPrime, &js); retry != s.stop || s.normRow != pending {
			t.Fatal("stopped workspace was overwritten by a later query", retry)
		}
		first, read := s.asBuffer.BorrowSlot(1)
		euclidMust(t, read)
		second, read := s.asBuffer.BorrowSlot(0)
		euclidMust(t, read)
		zero, read := z.IsZero(second)
		euclidMust(t, read)
		if z.IsFailureRef(first.Cell) {
			if !zero {
				t.Fatal("failed first copy reached the second buffer slot")
			}
		} else {
			got, read := z.String(first)
			euclidMust(t, read)
			if got != value {
				t.Fatal("source first buffer copy was not retained", got)
			}
			if !zero {
				got, read = z.String(second)
				euclidMust(t, read)
				if got != value {
					t.Fatal("later refusal damaged second copied buffer slot", got)
				}
			}
		}
	} else {
		if s.normRow != nil || s.stop != nil {
			t.Fatal("completed normalization retained a workspace")
		}
		checkEuclideanQueryResult(t, z, &aPrime, &cPrime, value, "7")
	}
	for i := range coeff {
		got, read := z.String(&coeff[i])
		euclidMust(t, read)
		if got != value {
			t.Fatal("borrowed input changed", i, got)
		}
	}
	euclidMust(t, s.Close())
	for i := range coeff {
		euclidMust(t, z.Del(&coeff[i]))
	}
	for _, v := range []*numeral.Int{&constant, &aPrime, &cPrime} {
		euclidMust(t, z.Del(v))
	}
}

func TestEuclideanNormalizeQuerySubstitutionPublishesConstantBeforeRationalRefusal(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	_, e = s.mkVar(false)
	euclidMust(t, e)
	j, e := s.mkJustification()
	euclidMust(t, e)
	minusOne, five := numeral.NewInt(-1), numeral.NewInt(5)
	proof := numeral.NewRational()
	euclidMust(t, m.SetString(&proof, "1"+strings.Repeat("0", 200)+"1/1"+strings.Repeat("0", 200)+"3"))
	_, e = s.addEquation([]numeral.Int{minusOne}, []uint32{0}, &five,
		[]numeral.Rational{proof}, []uint32{j}, true)
	euclidMust(t, e)
	euclidMust(t, s.promoteSolvedEquation(0, 0))
	coefficient, constant := numeral.NewInt(2), numeral.NewInt(3)
	aPrime, cPrime := numeral.NewInt(0), numeral.NewInt(0)
	var js []uint32
	z.Budget().FailAfter(0)
	e = s.normalizeQuery([]numeral.Int{coefficient}, []uint32{0},
		&constant, &aPrime, &cPrime, &js)
	z.Budget().FailAfter(-1)
	var failure umemory.Failure
	if !errors.Is(e, arena.ErrInjectedFailure) &&
		(!errors.As(e, &failure) || failure.Kind != umemory.FailureMemout) {
		t.Fatal("expected rational substitution allocation refusal", e)
	}
	got, read := z.String(&cPrime)
	euclidMust(t, read)
	if got != "13" || s.normRow == nil || s.stop == nil || len(s.normRow.js) != 0 {
		t.Fatalf("source constant-before-rational prefix lost: c_prime=%s, row=%v", got, s.normRow)
	}
	got, read = z.String(&constant)
	euclidMust(t, read)
	if got != "3" {
		t.Fatalf("borrowed input constant changed: %s", got)
	}
	euclidMust(t, s.Close())
	for _, value := range []*numeral.Int{&minusOne, &five, &coefficient, &constant, &aPrime, &cPrime} {
		euclidMust(t, z.Del(value))
	}
	euclidMust(t, m.Del(&proof))
}

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
