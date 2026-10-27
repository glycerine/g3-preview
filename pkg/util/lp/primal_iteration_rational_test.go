package lp

import (
	"math/big"
	"testing"
)

func TestLPRationalIterationPricingAndAdvance(t *testing.T) {
	for _, phase := range []string{"pivot", "flip", "unbounded", "optimal", "infeasible", "switch-assert", "budget-error", "pricing-error", "advance-error", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalAdvanceDifferentFixture(t, h)
			m.ColumnsNZ = []uint32{1, 1, 1}
			m.IterationsNoCostGrowth = 31
			m.TotalIterations = 17
			h.must(h.m.SetInt32(&m.D.Data[1], 1))
			want := error(nil)
			status := StatusFeasible
			switch phase {
			case "pivot":
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "1")))
				h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "2")))
			case "flip":
				f.types[1] = ColumnBoxed
				h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, "1")))
				h.must(h.m.Copy(&f.low.Data[1], lpScalarBoundValue(h, "-1")))
				status = StatusUnknown
			case "unbounded":
				status = StatusUnbounded
			case "optimal":
				h.must(h.m.SetInt32(&m.D.Data[1], 0))
				status = StatusOptimal
			case "infeasible":
				h.must(h.m.SetInt32(&m.D.Data[1], 0))
				h.must(m.InfSet.Insert(0))
				m.UsingInfeasCosts = true
				status = StatusInfeasible
			case "switch-assert":
				h.must(h.m.SetInt32(&m.D.Data[1], 0))
				h.must(m.InfSet.Insert(0))
				status = StatusInfeasible
				if sourceDebugEnabled {
					want = ErrAssertion
					status = StatusUnknown
				}
			case "budget-error":
				m.Settings = nil
				want = ErrBounds
				status = StatusUnknown
			case "pricing-error":
				f.types = f.types[:1]
				want = ErrBounds
				status = StatusUnknown
			case "advance-error":
				h.must(m.Factorization.Close())
				m.Factorization = nil
				want = ErrBounds
				status = StatusUnknown
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
				status = StatusUnknown
			}
			if e := m.OneIteration(); e != want {
				t.Fatal(e, want)
			}
			if m.Status != status || m.TotalIterations != 17 {
				t.Fatal("iteration owns status but not total count", m.Status, status, m.TotalIterations)
			}
			if phase == "pivot" {
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(14, 3))
				lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
				if f.basis[0] != 1 {
					t.Fatal("basis")
				}
			}
			if phase == "flip" {
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(1, 1))
				if f.basis[0] != 0 {
					t.Fatal("flip basis")
				}
			}
			if want != nil {
				if e := m.OneIteration(); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

// Independent standard-form maximum: each diagonal slack yields the upper
// bound x <= rhs_i / a_i. The cost is positive, so the smallest bound is the
// unique maximizing x. Drive the actual repeated iteration path, including
// InitRun, pricing after exchange and the no-entering optimal decision.
func TestLPRationalIterationIndependentBoundedOptima(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		n := 1 + world%4
		en := uint32(n)
		f := lpNewRationalCoreFixture(t, h, uint32(n), en+1)
		f.settings.SimplexStrategy = StrategyLU
		f.settings.DensityThreshold = 0
		if world%2 == 0 {
			f.settings.DensityThreshold = 1e100
		}
		var best *big.Rat
		ds, initial := make([]*big.Rat, n), make([]*big.Rat, n)
		for i := 0; i < n; i++ {
			diagonal := big.NewRat(int64(2+i), 1)
			ds[i] = big.NewRat(int64(1+(world+i)%7), int64(3+i))
			initial[i] = big.NewRat(int64(5+(world*3+i*7)%31), 11)
			coeff := new(big.Rat).Mul(diagonal, ds[i])
			rhs := new(big.Rat).Mul(diagonal, initial[i])
			h.must(f.a.Set(uint32(i), uint32(i), lpScalarBoundValue(h, diagonal.RatString())))
			h.must(f.a.Set(uint32(i), en, lpScalarBoundValue(h, coeff.RatString())))
			h.must(h.m.Copy(&f.x.Data[i], lpScalarBoundValue(h, initial[i].RatString())))
			h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, rhs.RatString())))
			f.types[i] = ColumnLowBound
			limit := new(big.Rat).Quo(rhs, coeff)
			if best == nil || limit.Cmp(best) < 0 {
				best = new(big.Rat).Set(limit)
			}
		}
		f.types[en] = ColumnLowBound
		profit := big.NewRat(int64(world%11+1), 13)
		h.must(h.m.SetString(&f.costs.Data[en], profit.RatString()))
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitRun())
		steps := 0
		for m.Status != StatusOptimal && steps < 8 {
			h.must(m.OneIteration())
			steps++
			if m.Status == StatusUnbounded || m.Status == StatusInfeasible || m.Status == StatusFloatingPointError {
				t.Fatal("bounded optimum rejected", world, m.Status)
			}
		}
		if m.Status != StatusOptimal || steps != 2 || m.enteringStep != nil {
			t.Fatal("iteration convergence", world, steps, m.Status)
		}
		lpSparseRationalCheck(t, h, &f.x.Data[en], best)
		for i := range ds {
			want := new(big.Rat).Sub(initial[i], new(big.Rat).Mul(ds[i], best))
			lpSparseRationalCheck(t, h, &f.x.Data[i], want)
			if want.Sign() < 0 {
				t.Fatal("oracle feasibility")
			}
		}
		actual := new(big.Rat).Mul(best, profit)
		lpSparseRationalCheck(t, h, &f.costs.Data[en], profit)
		got := lpScalarBoundValue(h, "0")
		h.must(h.m.Mul(&f.x.Data[en], &f.costs.Data[en], got))
		lpSparseRationalCheck(t, h, got, actual)
		if off, e := m.AMultXIsOff(); e != nil || off {
			t.Fatal("equation", e, off)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
