package realclosure

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRealClosurePolynomialRationalCoefficientOwnership(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	store := realValueStore{numbers: service.Manager}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	one, err := store.newRational(1, 1)
	must(err)
	large, err := store.newRational(1<<62, 3)
	must(err)
	zero, err := store.newRational(0, 1)
	must(err)
	if zero != nil {
		t.Fatal("zero coefficient must use the source nil representation")
	}
	var first, second realPolynomial
	must(store.setPolynomial(&first, []*realValue{nil, one, large}))
	if one.refs != 2 || large.refs != 2 || len(first.coefficients) != 3 {
		t.Fatal("polynomial did not retain its coefficient values")
	}
	must(store.setPolynomial(&second, []*realValue{one, large}))
	must(store.release(one))
	must(store.release(large))
	if one.refs != 2 || large.refs != 2 {
		t.Fatal("creator ownership did not transfer to the two polynomials")
	}
	must(store.resetPolynomial(&first))
	if one.deleted || large.deleted || one.refs != 1 || large.refs != 1 {
		t.Fatal("first polynomial deleted shared coefficients")
	}
	must(store.resetPolynomial(&second))
	if !one.deleted || !large.deleted || one.refs != 0 || large.refs != 0 {
		t.Fatal("last polynomial failed to delete its coefficients")
	}
	if err := store.retain(large); err == nil {
		t.Fatal("deleted coefficient was retained")
	}
	must(service.Close())
	if used := budget.Used(); used != 0 {
		t.Fatalf("real-closure rational values retained %d bytes", used)
	}
}
