package realclosure

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosureBoundedRationalSturmTarskiEndpoints(t *testing.T) {
	budget := arena.NewBudget(1 << 25)
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	bounds, err := numeral.NewBinaryRationalManager(service.Manager.Z, numeral.BinaryRationalConfig{Diagnostics: service})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	store := &realValueStore{numbers: service.Manager}
	one, err := store.newRational(1, 1)
	must(err)
	minusOne, err := store.newRational(-1, 1)
	must(err)
	p, q := []*realValue{minusOne, nil, one}, []*realValue{nil, one}
	q2, err := store.multiplyRationalPolynomials(q, q)
	must(err)
	lower, upper := numeral.NewBinaryRational(), numeral.NewBinaryRational()
	interval, err := NewBinaryInterval(bounds)
	must(err)
	check := func(label string, wantRoots, wantQ, wantQ2 int) {
		t.Helper()
		for _, usePseudo := range []bool{true, false} {
			roots, err := store.rootCountBinaryIntervalRational(p, interval, usePseudo)
			must(err)
			first, err := store.tarskiBinaryIntervalRational(p, q, interval, usePseudo)
			must(err)
			second, err := store.tarskiBinaryIntervalRational(p, q2, interval, usePseudo)
			must(err)
			if roots != wantRoots || first != wantQ || second != wantQ2 {
				t.Fatalf("%s use_prem=%v: roots=%d, q=%d, q²=%d; want %d,%d,%d", label, usePseudo, roots, first, second, wantRoots, wantQ, wantQ2)
			}
		}
	}
	must(bounds.SetInt32(&lower, -2))
	must(bounds.SetInt32(&upper, 0))
	must(interval.SetFinite(&lower, &upper, true, false))
	check("(-2,0]", 1, -1, 1)
	must(interval.ResetLower())
	check("(-oo,0]", 1, -1, 1)
	must(bounds.SetInt32Power(&lower, 1, 1)) // 1/2
	must(bounds.SetInt32(&upper, 2))
	must(interval.SetFinite(&lower, &upper, false, false))
	check("[1/2,2]", 1, 1, 1)
	must(interval.ResetUpper())
	check("[1/2,+oo)", 1, 1, 1)
	must(store.releaseCoefficients(q2))
	must(store.release(one))
	must(store.release(minusOne))
	must(interval.Close())
	must(bounds.Del(&lower))
	must(bounds.Del(&upper))
	must(bounds.CloseChecked())
	must(service.Close())
	if used := budget.Used(); used != 0 {
		t.Fatalf("bounded real-closure query retained %d bytes", used)
	}
}
