package realclosure

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosureRationalDerivativeKeepsCoefficientOwnership(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	store := &realValueStore{numbers: service.Manager}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	one, err := store.newRational(1, 1)
	must(err)
	minusOne, err := store.newRational(-1, 1)
	must(err)
	three, err := store.newRational(3, 1)
	must(err)
	// d(x³-x)/dx = 3x²-1, with the source nil zero coefficient.
	derived, err := store.derivativeRationalPolynomial([]*realValue{nil, minusOne, nil, one})
	must(err)
	if len(derived) != 3 || derived[1] != nil || derived[0] == nil || derived[2] == nil {
		t.Fatalf("unexpected derivative coefficients: %v", derived)
	}
	constant, err := store.numbers.Equal(&derived[0].rational, &minusOne.rational)
	must(err)
	quadratic, err := store.numbers.Equal(&derived[2].rational, &three.rational)
	must(err)
	if !constant || !quadratic || one.refs != 1 || minusOne.refs != 1 {
		t.Fatal("derivative changed its input or produced incorrect coefficients")
	}
	must(store.releaseCoefficients(derived))
	zero, err := store.derivativeRationalPolynomial([]*realValue{one})
	must(err)
	if len(zero) != 0 {
		t.Fatal("constant polynomial has nonzero derivative")
	}
	must(store.release(one))
	must(store.release(minusOne))
	must(store.release(three))
	must(service.Close())
	if used := budget.Used(); used != 0 {
		t.Fatalf("real-closure derivative retained %d bytes", used)
	}
}
