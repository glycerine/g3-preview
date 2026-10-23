package simplex

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestArithmeticRowConflictPreservesEqualityPremises(t *testing.T) {
	s := NewSolver(nil)
	if _, err := s.AddRationalRow(0, []uint32{0, 1, 2}, integerRow(1, -2, 3)); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUpperWithDependencies(1, RationalInfinitesimal(big.NewRat(1, 1)), []BoundDependency{{Equality: 7, Coefficient: *big.NewRat(5, 7)}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLowerWithDependencies(2, RationalInfinitesimal(big.NewRat(0, 1)), []BoundDependency{{Equality: 8, Coefficient: *big.NewRat(11, 13)}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLowerWithDependencies(0, RationalInfinitesimal(big.NewRat(10, 1)), []BoundDependency{{Literal: 12, Coefficient: *big.NewRat(1, 1)}}); err != nil {
		t.Fatal(err)
	}
	s.SetValue(1, RationalInfinitesimal(big.NewRat(1, 1)))
	status, err := s.MakeFeasible()
	if err != nil || status != lbool.False {
		t.Fatal("expected infeasible equality-supported row", status, err)
	}
	if err := s.ExplainArithmeticInfeasibleRow(nil, false); err != nil {
		t.Fatal(err)
	}
	dependencies, known := s.BoundConflictDependencies()
	if !known || len(dependencies) != 3 || dependencies[0].Equality != 7 || dependencies[1].Equality != 8 || dependencies[2].Literal != 12 || dependencies[0].Coefficient.Cmp(big.NewRat(-10, 7)) != 0 || dependencies[1].Coefficient.Cmp(big.NewRat(33, 13)) != 0 {
		t.Fatal("row conflict lost typed equality premises or multipliers", dependencies)
	}
	dependencies[0].Equality = 99
	dependencies[0].Coefficient.SetInt64(99)
	retained, _ := s.BoundConflictDependencies()
	if retained[0].Equality != 7 || retained[0].Coefficient.Cmp(big.NewRat(-10, 7)) != 0 {
		t.Fatal("returned premise values alias the solver's owned explanation")
	}
}
