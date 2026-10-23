package simplex

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestArithmeticRowConflictSignedBasesAndRelaxationBudget(t *testing.T) {
	for _, scale := range []int64{-7, 7} {
		for _, sign := range []int64{-1, 1} {
			for _, integer := range []bool{false, true} {
				s := NewSolver(nil)
				_, err := s.AddRationalRow(0, []uint32{0, 1, 2}, integerRow(scale, -2*scale, 3*scale))
				if err != nil {
					t.Fatal(err)
				}
				set := func(column uint32, lower bool, value *big.Rat, literal int32) {
					bound := RationalInfinitesimal(value)
					dependencies := []BoundDependency{{Literal: literal, Coefficient: *big.NewRat(1, 1)}}
					if lower {
						err = s.SetLowerWithDependencies(column, bound, dependencies)
					} else {
						err = s.SetUpperWithDependencies(column, bound, dependencies)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				set(1, sign < 0, big.NewRat(sign, 1), 12)
				set(2, sign > 0, big.NewRat(0, 1), 14)
				set(0, sign > 0, big.NewRat(10*sign, 1), 10)
				// Put nonbases at the endpoints blocking repair, preserving
				// this basic row for the exact source-order explanation.
				s.SetValue(1, RationalInfinitesimal(big.NewRat(sign, 1)))
				for i, value := range []int64{3, 4, 5, 9} {
					bound := RationalInfinitesimal(big.NewRat(sign*value, 1))
					dependencies := []BoundDependency{{Literal: int32(16 + 2*i), Coefficient: *big.NewRat(1, 1)}}
					s.AddArithmeticBoundAtom(1, sign < 0, bound, dependencies)
					bound.Main.SetInt64(999)
					dependencies[0].Literal = 99
					dependencies[0].Coefficient.SetInt64(99)
				}
				s.AddArithmeticBoundAtom(2, sign > 0, RationalInfinitesimal(big.NewRat(-sign, 2)), []BoundDependency{{Literal: 26, Coefficient: *big.NewRat(1, 1)}})
				status, err := s.MakeFeasible()
				if err != nil || status != lbool.False {
					t.Fatal(status, err)
				}
				integers := map[uint32]bool{}
				if integer {
					integers[0] = true
				}
				if err = s.ExplainArithmeticInfeasibleRow(integers, false); err != nil {
					t.Fatal(err)
				}
				unrelaxed, known := s.BoundConflictDependencies()
				if !known || len(unrelaxed) != 3 {
					t.Fatal(unrelaxed, known)
				}
				for i, want := range []int32{12, 14, 10} {
					if unrelaxed[i].Literal != want || unrelaxed[i].Coefficient.Cmp(big.NewRat([]int64{-2, 3, 1}[i], 1)) != 0 {
						t.Fatal("unrelaxed row explanation", unrelaxed)
					}
				}
				if err = s.ExplainArithmeticInfeasibleRow(integers, true); err != nil {
					t.Fatal(err)
				}
				dependencies, known := s.BoundConflictDependencies()
				if !known || len(dependencies) != 3 {
					t.Fatal(dependencies, known)
				}
				zLiteral := int32(26)
				if integer {
					zLiteral = 14
				}
				for i, want := range []int32{18, zLiteral, 10} {
					if dependencies[i].Literal != want || dependencies[i].Coefficient.Cmp(big.NewRat([]int64{2, 3, 1}[i], 1)) != 0 {
						t.Fatal(scale, sign, integer, "source explanation", dependencies)
					}
				}
				endpoint := func(column uint32, lower bool, literal int32) Infinitesimal {
					original, _ := s.BoundDependencies(column, lower)
					if len(original) == 1 && original[0].Literal == literal {
						value, _ := s.Upper(column)
						if lower {
							value, _ = s.Lower(column)
						}
						return value
					}
					for _, atom := range s.arithmeticBoundAtoms[arithmeticBoundKey{column, lower}] {
						if len(atom.dependencies) == 1 && atom.dependencies[0].Literal == literal {
							return atom.value.Clone()
						}
					}
					t.Fatal("explanation references no active endpoint", literal)
					return Infinitesimal{}
				}
				// Reconstruct the implied endpoint from the selected premises,
				// independently of the explanation's relaxation arithmetic.
				implied := endpoint(1, sign < 0, dependencies[0].Literal)
				implied.Mul(big.NewRat(2, 1))
				z := endpoint(2, sign > 0, dependencies[1].Literal)
				z.Mul(big.NewRat(3, 1))
				implied.Sub(&z)
				bound := endpoint(0, sign > 0, dependencies[2].Literal)
				if sign > 0 && implied.Cmp(&bound) >= 0 || sign < 0 && implied.Cmp(&bound) <= 0 {
					t.Fatal("selected endpoints do not imply a contradiction", implied, bound)
				}
			}
		}
	}
}

func TestArithmeticRowConflictStrictGapAndUnknownPremises(t *testing.T) {
	for _, known := range []bool{false, true} {
		s := NewSolver(nil)
		_, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(2, -3))
		if err != nil {
			t.Fatal(err)
		}
		zero := RationalInfinitesimal(new(big.Rat))
		if known {
			err = s.SetUpperWithDependencies(1, zero, []BoundDependency{{Literal: 12, Coefficient: *big.NewRat(1, 1)}})
		} else {
			err = s.SetUpper(1, zero)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = s.SetLowerWithDependencies(0, NewInfinitesimal(nil, big.NewRat(1, 1)), []BoundDependency{{Literal: 10, Coefficient: *big.NewRat(1, 1)}}); err != nil {
			t.Fatal(err)
		}
		if status, err := s.MakeFeasible(); err != nil || status != lbool.False {
			t.Fatal(status, err)
		}
		if err = s.ExplainArithmeticInfeasibleRow(nil, true); err != nil {
			t.Fatal(err)
		}
		dependencies, explained := s.BoundConflictDependencies()
		if explained != known {
			t.Fatal("candidate premise became logical", explained, known)
		}
		if known && (len(dependencies) != 2 || dependencies[0].Literal != 12 || dependencies[0].Coefficient.Cmp(big.NewRat(-3, 2)) != 0 || dependencies[1].Literal != 10) {
			t.Fatal("zero-slack normalized coefficients", dependencies)
		}
	}
}
