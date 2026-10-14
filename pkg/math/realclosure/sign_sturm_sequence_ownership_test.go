package realclosure

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// Source: src/math/realclosure/realclosure.cpp:406-451, 3637-3674.
func TestRealClosureRationalSturmSequenceOwnsBorrowedCoefficients(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	service, err := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	store := &realValueStore{numbers: service.Manager}
	one, err := store.newRational(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	minusOne, err := store.newRational(-1, 1)
	if err != nil {
		t.Fatal(err)
	}
	three, err := store.newRational(3, 1)
	if err != nil {
		t.Fatal(err)
	}
	// P=x^3-x and P'=3x^2-1. The two nonzero remainders extend the
	// sequence after the two borrowed input rows.
	rows, err := store.sturmSequenceRational([]*realValue{nil, minusOne, nil, one}, []*realValue{minusOne, nil, three}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || one.refs != 2 || minusOne.refs != 3 || three.refs != 2 {
		t.Fatal("sequence did not retain borrowed source coefficients", len(rows), one.refs, minusOne.refs, three.refs)
	}
	for _, value := range []*realValue{one, minusOne, three} {
		if err := store.release(value); err != nil {
			t.Fatal(err)
		}
		if value.deleted {
			t.Fatal("live sequence lost a caller coefficient")
		}
	}
	if err := store.closePolynomialSequence(rows); err != nil {
		t.Fatal(err)
	}
	if !one.deleted || !minusOne.deleted || !three.deleted {
		t.Fatal("sequence cleanup retained source coefficients")
	}
	for i := range rows {
		if rows[i].coefficients != nil {
			t.Fatal("sequence cleanup retained a polynomial", i)
		}
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if used := budget.Used(); used != 0 {
		t.Fatal("sequence retained numeric storage", used)
	}
}
