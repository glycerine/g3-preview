package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLARBasicDeltaActualClampPreservesTableauEquations(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(41782, 367))
	for world := 0; world < 160; world++ {
		a := big.NewRat(int64(rng.IntN(9)+1), int64(rng.IntN(7)+1))
		b := big.NewRat(int64(rng.IntN(11)+1), int64(rng.IntN(13)+1))
		if world%2 == 0 {
			a.Neg(a)
		}
		if world%3 == 0 {
			b.Neg(b)
		}
		s := lpNewRemovalModel(t, h, a.RatString(), "7", false)
		p := h.pair(b.RatString(), "0")
		_, err := s.AddTerm([]RationalSparseEntry{{Index: 0, Value: p.X}}, &p.Y)
		h.must(err)
		if world%2 == 0 {
			s.Settings.SimplexStrategy = StrategyTableauCosts
		}
		c := s.Core
		h.must(s.BasicColumnsWithChangedCost.Resize(4))
		lo := h.pair("3/7", "1/5")
		h.must(s.Bounds.Low.Set(0, lo))
		h.must(s.Bounds.Types.Set(0, ColumnLowBound))
		h.must(c.RPrimal.InfSet.Insert(0))
		delta := h.pair("17", "19")
		changed, err := c.RPrimal.MakeColumnFeasible(0, delta)
		h.must(err)
		if !changed {
			t.Fatal("missing actual clamp")
		}
		expected := [][2]*big.Rat{{new(big.Rat).Mul(a, big.NewRat(3, 7)), new(big.Rat).Mul(a, big.NewRat(1, 5))}, {new(big.Rat).Mul(b, big.NewRat(3, 7)), new(big.Rat).Mul(b, big.NewRat(1, 5))}}
		for row := 0; row < 2; row++ {
			bound := h.pair(expected[row][0].RatString(), expected[row][1].RatString())
			if row == 0 {
				h.must(h.m.SetString(&bound.X, new(big.Rat).Add(expected[row][0], big.NewRat(1, 13)).RatString()))
			} else {
				h.must(h.m.SetString(&bound.Y, new(big.Rat).Sub(expected[row][1], big.NewRat(1, 17)).RatString()))
			}
			h.must(s.Bounds.Low.Set(uint32(row+2), bound))
			h.must(s.Bounds.Types.Set(uint32(row+2), ColumnLowBound))
			h.must(c.RPrimal.InfSet.Insert(uint32(row + 2)))
		}
		h.must(s.ChangeBasicXByDeltaOnColumn(0, delta))
		for row := 0; row < 2; row++ {
			h.check(&c.RX.Data[row+2], expected[row][0], expected[row][1])
		}
		if !c.RPrimal.InfSet.Contains(0) || !c.RPrimal.InfSet.Contains(2) || c.RPrimal.InfSet.Contains(3) {
			t.Fatal("per-column membership", c.RPrimal.InfSet.index)
		}
		for _, j := range []uint32{2, 3} {
			if s.BasicColumnsWithChangedCost.Contains(j) != (world%2 == 0) {
				t.Fatal("cost change mode", world, j)
			}
		}
		h.check(delta, big.NewRat(3, 7), big.NewRat(1, 5))
		// A*x=0 follows independently: each basis value equals coefficient*x0;
		// x1 stays zero and each tableau's own basis coefficient remains one.
		h.check(&c.RX.Data[1], big.NewRat(0, 1), big.NewRat(0, 1))
		if s.work != nil || s.rationalWork != nil {
			t.Fatal("retained completed expressions")
		}
		h.must(s.Close())
	}
}

func TestLARBasicDeltaIndependentNonunitLU(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(9182, 3671))
	for world := 0; world < 100; world++ {
		a := big.NewRat(int64(rng.IntN(17)-8), int64(rng.IntN(11)+1))
		basis := big.NewRat(int64(rng.IntN(9)+1), int64(rng.IntN(7)+1))
		if world%2 == 0 {
			basis.Neg(basis)
		}
		s := lpPreparedLARPivot(t, h, a.RatString(), "2/3", basis.RatString())
		c := s.Core
		h.must(c.RX.Data[2].Copy(h.m, h.pair("7/11", "-3/13")))
		h.must(c.RPrimal.InfSet.Insert(2))
		delta := h.pair("5/17", "-7/19")
		h.must(s.ChangeBasicXByDeltaOnColumn(0, delta))
		coefficient := new(big.Rat).Quo(a, basis)
		h.check(&c.RX.Data[2], new(big.Rat).Sub(big.NewRat(7, 11), new(big.Rat).Mul(coefficient, big.NewRat(5, 17))), new(big.Rat).Sub(big.NewRat(-3, 13), new(big.Rat).Mul(coefficient, big.NewRat(-7, 19))))
		lpSparseRationalCheck(t, h, &s.ColumnBuffer.Data[0], coefficient)
		if c.RPrimal.InfSet.Contains(2) != (a.Sign() == 0) {
			t.Fatal("source only visits nonzero solved indices")
		}
		if s.BasicColumnsWithChangedCost.Size() != 0 {
			t.Fatal("LU recorded tableau cost marks")
		}
		h.must(s.Close())
	}
}

func TestLARBasicDeltaSourcePrefixesAndStoppedGuards(t *testing.T) {
	for _, op := range []string{"nil", "column", "row", "basis", "basis-short", "link", "cost-domain", "inf-domain", "zero-delta", "LU-column"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			delta := h.pair("1", "2")
			want := ErrBounds
			var err error
			switch op {
			case "nil":
				err = s.ChangeBasicXByDeltaOnColumn(0, nil)
			case "column":
				err = s.ChangeBasicXByDeltaOnColumn(99, delta)
			case "row":
				c.RA.Columns[0][0].Row = 99
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
			case "basis-short":
				c.RBasis = nil
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
			case "basis":
				c.RBasis[0] = 99
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
			case "link":
				c.RA.Columns[0][0].Offset = 99
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
			case "cost-domain":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
				h.check(&c.RX.Data[2], big.NewRat(2, 1), big.NewRat(4, 1))
			case "inf-domain":
				h.must(c.RPrimal.InfSet.Resize(0))
				h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				h.must(s.Bounds.Low.Set(2, h.pair("100", "0")))
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
				h.check(&c.RX.Data[2], big.NewRat(2, 1), big.NewRat(4, 1))
				if !c.RPrimal.PairCoreSolverBase.stopped || s.calls.pending() {
					t.Fatal("membership ownership")
				}
			case "zero-delta":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				h.must(s.BasicColumnsWithChangedCost.Resize(3))
				delta = h.pair("0", "0")
				h.must(c.RPrimal.InfSet.Insert(2))
				err = s.ChangeBasicXByDeltaOnColumn(0, delta)
				want = nil
				if !s.BasicColumnsWithChangedCost.Contains(2) || c.RPrimal.InfSet.Contains(2) {
					t.Fatal("source zero delta still visits linked rows")
				}
			case "LU-column":
				h.must(s.Close())
				s = lpPreparedLARPivot(t, h, "2", "3", "5")
				err = s.ChangeBasicXByDeltaOnColumn(99, delta)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			if err != want {
				t.Fatal(op, err, want)
			}
			if err != nil && s.ChangeBasicXByDeltaOnColumn(0, delta) != ErrExactVectorPending {
				t.Fatal("lost stopped outer owner")
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARBasicDeltaPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"tableau", "costs", "LU"} {
			failures, own, child := 0, 0, 0
			success := false
			for ordinal := int64(0); ordinal < 260; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					x := strings.Repeat("1234567890", 100) + "/7"
					y := strings.Repeat("9876543210", 100) + "/13"
					var s *LARSolver
					if op == "LU" {
						s = lpPreparedLARPivot(t, h, x, "3", "1"+strings.Repeat("0", 999)+"3")
					} else {
						s = lpNewRemovalModel(t, h, x, y, false)
					}
					c := s.Core
					delta := h.pair(x, y)
					if op == "costs" {
						s.Settings.SimplexStrategy = StrategyTableauCosts
						h.must(s.BasicColumnsWithChangedCost.Resize(3))
					}
					h.budget.FailAfter(ordinal)
					err := s.ChangeBasicXByDeltaOnColumn(0, delta)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						coefficient := mustLPRat(t, x)
						if op == "LU" {
							coefficient.Quo(coefficient, mustLPRat(t, "1"+strings.Repeat("0", 999)+"3"))
						} else {
							coefficient.Neg(coefficient)
						}
						h.check(&c.RX.Data[2], new(big.Rat).Neg(new(big.Rat).Mul(coefficient, mustLPRat(t, x))), new(big.Rat).Neg(new(big.Rat).Mul(coefficient, mustLPRat(t, y))))
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if s.calls.pending() {
							own++
						}
						if c.RPrimal.PairCoreSolverBase.stopped || s.ColumnBuffer.calls.pending() || (c.RPrimal.Factorization != nil && c.RPrimal.Factorization.stopped) {
							child++
							if s.calls.pending() {
								t.Fatal("outer adopted child numerical failure")
							}
						}
						if s.ChangeBasicXByDeltaOnColumn(0, delta) != ErrExactVectorPending {
							t.Fatal("lost outer stop")
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || own == 0 || !success {
				t.Fatal("incomplete expression campaign", direct, op, failures, own, success)
			}
			if op == "LU" && child == 0 {
				t.Fatal("no actual LU child refusal", direct)
			}
		}
	}
}
