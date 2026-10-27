package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLPPairReducedCostIndependentSystems(t *testing.T) {
	rng := rand.New(rand.NewPCG(82311, 7119))
	for _, op := range []string{"fill", "init"} {
		for world := 0; world < 120; world++ {
			h := newLPPairHarness(t)
			rows := 1 + world%4
			cols := rows + 3
			f := lpNewPairCoreFixture(t, h, uint32(rows), uint32(cols))
			perm := rng.Perm(cols)
			for i := range f.basis {
				f.basis[i] = uint32(perm[i])
			}
			f.settings.SimplexStrategy = StrategyLU
			f.settings.DensityThreshold = 1e100
			if world%2 == 0 {
				f.settings.DensityThreshold = 0
			}
			y := make([]*big.Rat, rows)
			for i := range y {
				y[i] = big.NewRat(int64(world%17-i-8), int64(7+i))
			}
			a := make([][]*big.Rat, rows)
			for i := range a {
				a[i] = make([]*big.Rat, cols)
				for j := range a[i] {
					a[i][j] = new(big.Rat)
				}
				for k, j := range perm {
					v := big.NewRat(int64(rng.IntN(7)-3), 11)
					if k == i {
						v.SetInt64(int64(13 + rows + i))
					}
					a[i][j] = v
					if v.Sign() != 0 {
						h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
					}
				}
			}
			cost := make([]*big.Rat, cols)
			for j := range cost {
				cost[j] = big.NewRat(int64(world%23+j-11), 13)
			}
			for _, j := range f.basis {
				cost[j] = new(big.Rat)
				for i := range y {
					cost[j].Add(cost[j], new(big.Rat).Mul(y[i], a[i][j]))
				}
			}
			for j := range cost {
				h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
			}
			m, e := NewPairCoreSolverBase(f.input())
			h.must(e)
			for i := range y {
				h.must(h.m.SetString(&m.Y.Data[i], y[i].RatString()))
			}
			for j := range cost {
				h.must(h.m.SetInt32(&m.D.Data[j], 97))
			}
			m.D.Index = []uint32{uint32(cols - 1), 0}
			m.Y.Index = []uint32{0}
			if op == "fill" {
				h.must(m.FillReducedCostsFromYByRows())
			} else {
				h.must(m.InitReducedCostsForOneIteration())
			}
			for j := range cost {
				want := new(big.Rat)
				if f.heading[j] < 0 {
					want.Set(cost[j])
					for i := range y {
						want.Sub(want, new(big.Rat).Mul(y[i], a[i][j]))
					}
				}
				lpSparseRationalCheck(t, h, &m.D.Data[j], want)
				lpSparseRationalCheck(t, h, &f.costs.Data[j], cost[j])
			}
			for i := range y {
				lpSparseRationalCheck(t, h, &m.Y.Data[i], y[i])
			}
			if len(m.D.Index) != 2 || m.D.Index[0] != uint32(cols-1) || len(m.Y.Index) != 1 {
				t.Fatal("changed dense vector indices")
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPPairReducedCostLiveAliases(t *testing.T) {
	for _, phase := range []string{"cost-shift", "y-d", "coefficient-d"} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 4)
		for i := uint32(0); i < 2; i++ {
			h.must(f.a.Set(i, i, &h.pair("1", "0").X))
			// Nonbasic row order matters when Y aliases the first destination.
			h.must(f.a.Set(i, 2, &h.pair("2", "0").X))
			h.must(f.a.Set(i, 3, &h.pair("3", "0").X))
		}
		m, e := NewPairCoreSolverBase(f.input())
		h.must(e)
		for j := range f.costs.Data {
			h.must(h.m.SetInt32(&f.costs.Data[j], int32(7+j)))
		}
		for j := range m.D.Data {
			h.must(h.m.SetInt32(&m.D.Data[j], int32(17+j)))
		}
		h.must(h.m.SetInt32(&m.Y.Data[0], 2))
		h.must(h.m.SetInt32(&m.Y.Data[1], 5))
		originalY := m.Y.Data
		originalCosts := m.Costs
		switch phase {
		case "cost-shift":
			// Descending writes cascade: Costs[3]=D[4]=21, then Costs[2]
			// rereads newly assigned D[3]=21 instead of its old value20.
			h.must(m.D.resizeData(5))
			h.must(h.m.SetInt32(&m.D.Data[4], 21))
			costs := m.D.Data[1:5]
			m.Costs = &costs
		case "y-d":
			m.Y.Data = m.D.Data[2:4]
		case "coefficient-d":
			// Share the rational value, not owning scalar identity. Copy-on-write
			// must retain the original coefficient after D initialization.
			h.must(h.m.Copy(&m.D.Data[2], &f.a.Rows[1][1].Value))
		}
		h.must(m.FillReducedCostsFromYByRows())
		switch phase {
		case "cost-shift":
			lpSparseRationalCheck(t, h, &m.D.Data[2], big.NewRat(7, 1))
			lpSparseRationalCheck(t, h, &m.D.Data[3], big.NewRat(0, 1))
			lpSparseRationalCheck(t, h, &m.D.Data[4], big.NewRat(21, 1))
		case "y-d":
			// D=[0,0,9,10]; row1 y=D3=10: D2=-11, D3=-20.
			// Row0 y=D2=-11: D2=11, then D3=-53 from reread y=11.
			lpSparseRationalCheck(t, h, &m.D.Data[2], big.NewRat(11, 1))
			lpSparseRationalCheck(t, h, &m.D.Data[3], big.NewRat(-53, 1))
		case "coefficient-d":
			lpSparseRationalCheck(t, h, &m.D.Data[2], big.NewRat(-5, 1))
			lpSparseRationalCheck(t, h, &f.a.Rows[1][1].Value, big.NewRat(2, 1))
		}
		m.Y.Data = originalY
		m.Costs = originalCosts
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairReducedCostPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"fill", "init"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 2, 4)
					f.settings.SimplexStrategy = StrategyLU
					huge := strings.Repeat("1234567890", 12)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &h.pair(fmt.Sprint(3+i), "0").X))
						h.must(f.a.Set(i, 2, &h.pair(huge+"7/13", "0").X))
						h.must(f.a.Set(i, 3, &h.pair(huge+"9/17", "0").X))
					}
					m, e := NewPairCoreSolverBase(f.input())
					h.must(e)
					for i := range m.Y.Data {
						h.must(h.m.SetString(&m.Y.Data[i], huge+"1/19"))
					}
					for j := range f.costs.Data {
						h.must(h.m.SetString(&f.costs.Data[j], huge+"3/23"))
						h.must(h.m.SetInt32(&m.D.Data[j], 97))
					}
					h.budget.FailAfter(ordinal)
					var err error
					if op == "fill" {
						err = m.FillReducedCostsFromYByRows()
					} else {
						err = m.InitReducedCostsForOneIteration()
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.Factorization.stopped {
							t.Fatal("adopted LU receipt")
						}
						if op == "init" && m.Factorization.stopped {
							for j := range m.D.Data {
								lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
							}
						}
						if e = m.FillReducedCostsFromYByRows(); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if e = m.InitReducedCostsForOneIteration(); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPPairReducedCostLeavesOlderComparisonOwned(t *testing.T) {
	for _, op := range []string{"fill", "init"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
				h.must(f.a.Set(0, 1, &h.pair("5/7", "0").X))
				m, err := NewPairPrimalCoreSolver(f.input())
				h.must(err)
				m.UsingInfeasCosts = true
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.low.Data[0].X, &v.X))
				h.must(h.m.Copy(&f.x.Data[0].X, &v.Y))
				h.must(h.m.SetInt32(&f.costs.Data[0], 17))
				h.must(h.m.Copy(&m.Y.Data[0], &v.Y))
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
				h.budget.FailAfter(ordinal)
				_, err = h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if op == "fill" {
						err = m.FillReducedCostsFromYByRows()
					} else {
						err = m.InitReducedCostsForOneIteration()
					}
					if m.calls.pending() || m.PairCoreSolverBase.calls.pending() || m.work != nil {
						t.Fatal("adopted older comparison", err)
					}
					lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(17, 1))
					if err == nil {
						if op != "init" {
							t.Fatal("expected large product refusal")
						}
						lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(-85, 21))
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(m.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}

func TestLPPairReducedCostReachedGuardsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"heading", "d", "cost", "y", "cell", "zero-row", "basic-no-cost", "missing-lu", "stopped-lu"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 2)
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			if phase == "stopped-lu" {
				f.settings.SimplexStrategy = StrategyLU
			}
			h.must(h.m.SetInt32(&f.costs.Data[0], 17))
			h.must(h.m.SetInt32(&f.costs.Data[1], 5))
			m, e := NewPairCoreSolverBase(f.input())
			h.must(e)
			h.must(h.m.SetInt32(&m.Y.Data[0], 3))
			for j := range m.D.Data {
				h.must(h.m.SetInt32(&m.D.Data[j], 97))
			}
			want := ErrBounds
			switch phase {
			case "heading":
				f.heading = nil
			case "d":
				h.must(m.D.resizeData(0))
			case "cost":
				h.must(f.costs.resizeData(0))
			case "y":
				h.must(m.Y.resizeData(0))
			case "cell":
				f.a.Rows[0][1].Column = 99
			case "zero-row":
				f.a.Rows[0][1].Column = 99
				h.must(h.m.SetInt32(&m.Y.Data[0], 0))
				want = nil
			case "basic-no-cost":
				f.heading[1] = 0
				h.must(f.costs.resizeData(0))
				want = nil
			case "stopped-lu":
				stopWant := ErrBounds
				if sourceDebugEnabled {
					stopWant = ErrAssertion
				}
				if e = m.Factorization.SolveBy(nil); e != stopWant {
					t.Fatal(e)
				}
				want = ErrLUStopped
			}
			var err error
			if phase == "missing-lu" || phase == "stopped-lu" {
				err = m.InitReducedCostsForOneIteration()
			} else {
				err = m.FillReducedCostsFromYByRows()
			}
			if err != want {
				t.Fatal(err, want)
			}
			if phase == "missing-lu" || phase == "stopped-lu" {
				lpSparseRationalCheck(t, h, &m.Y.Data[0], big.NewRat(17, 1))
				for j := range m.D.Data {
					lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
				}
				if m.calls.pending() || m.pivotWork != nil {
					t.Fatal("adopted child failure")
				}
			}
			if phase == "y" || phase == "cell" || phase == "zero-row" {
				lpSparseRationalCheck(t, h, &m.D.Data[0], big.NewRat(0, 1))
				lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(5, 1))
			}
			if err != nil {
				if e = m.InitReducedCostsForOneIteration(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			f.a.Rows[0][1].Column = 1
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
