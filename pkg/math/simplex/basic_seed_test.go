package simplex

import (
	"math/big"
	"testing"
)

// Source simplex_def.h set_value/update_value: a basic column's self-row
// cancels the requested delta. Nonbasic updates still move dependent bases.
func TestSimplexBasicSeedPreservesCanonicalRow(t *testing.T) {
	s := NewSolver(nil)
	s.SetValue(1, NewInfinitesimal(big.NewRat(3, 1), big.NewRat(-2, 1)))
	_, err := s.AddIntegerRow(2, []uint32{2, 1}, []big.Int{*big.NewInt(2), *big.NewInt(-3)})
	if err != nil {
		t.Fatal(err)
	}
	before := s.Value(2)
	s.SetValue(2, NewInfinitesimal(big.NewRat(100, 1), big.NewRat(7, 1)))
	after := s.Value(2)
	if after.Cmp(&before) != 0 {
		t.Fatal("seeding a basic column changed its row value", after.String(), before.String())
	}
	seed := NewInfinitesimal(big.NewRat(4, 1), big.NewRat(6, 1))
	s.SetValue(1, seed)
	expected := seed.Clone()
	expected.Mul(big.NewRat(3, 2))
	after = s.Value(2)
	if after.Cmp(&expected) != 0 {
		t.Fatal("nonbasic seed did not move its dependent base", after.String(), expected.String())
	}
}
