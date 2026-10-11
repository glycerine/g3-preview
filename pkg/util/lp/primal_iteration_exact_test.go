package lp

import (
	"math/big"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExactIterationCountIndependentRNG(t *testing.T) {
	for _, count := range []int{0, 1, 2, 300, 301, 401} {
		for _, percent := range []uint32{0, 1, 30, 100, 1000, ^uint32(0)} {
			for _, tentative := range []bool{false, true} {
				h := newLPPairHarness(t)
				f, m := lpAdvanceDifferentFixture(t, h)
				f.nb = make([]uint32, count)
				m.Settings.PercentOfEnteringToCheck = percent
				m.Status = StatusUnknown
				if tentative {
					m.Status = StatusTentativeUnbounded
				}
				seed := uint32(count) + percent + 97
				m.Settings.RandomSeed(seed)
				rng := z3rand.New(seed)
				want := uint32(count)
				if !tentative {
					if want > 300 {
						want = uint32((uint64(want)*uint64(percent))&0xffffffff) / 100
					}
					if want != 0 {
						want = rng.Next() % want
						if want < 1 {
							want = 1
						}
					}
				}
				got, e := m.NumberOfNonBasicColumnsToTryForEnter()
				h.must(e)
				if got != want || m.Settings.RandomNext() != rng.Next() {
					t.Fatal(count, percent, tentative, got, want)
				}
				m.stopped = true
				if e := m.DecideOnStatusWhenCannotFindEntering(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				if _, e := m.NumberOfNonBasicColumnsToTryForEnter(); e != ErrExactVectorPending {
					t.Fatal(e)
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPExactIterationPricingAndAdvance(t *testing.T) {
	for _, phase := range []string{"pivot", "flip", "unbounded", "optimal", "infeasible", "switch-assert", "pricing-error", "advance-error", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
			m.ColumnsNZ = []uint32{1, 1, 1}
			m.IterationsNoCostGrowth = 31
			m.TotalIterations = 17
			h.must(h.m.SetInt32(&m.D.Data[1], 1))
			want := error(nil)
			status := StatusFeasible
			switch phase {
			case "pivot":
				f.types[0] = ColumnLowBound
				h.must(f.x.Data[0].Copy(h.m, h.pair("1", "0")))
				h.must(f.b.Data[0].Copy(h.m, h.pair("2", "0")))
			case "flip":
				f.types[1] = ColumnBoxed
				h.must(f.up.Data[1].Copy(h.m, h.pair("1", "0")))
				h.must(f.low.Data[1].Copy(h.m, h.pair("-1", "0")))
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
				h.check(&f.x.Data[1], big.NewRat(14, 3), new(big.Rat))
				h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
				if f.basis[0] != 1 {
					t.Fatal("basis")
				}
			}
			if phase == "flip" {
				h.check(&f.x.Data[1], big.NewRat(1, 1), new(big.Rat))
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
func TestLPExactIterationIndependentBoundedOptima(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		n := 1 + world%4
		en := uint32(n)
		f := lpNewPairCoreFixture(t, h, uint32(n), en+1)
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
			h.must(f.a.Set(uint32(i), uint32(i), &h.pair(diagonal.RatString(), "0").X))
			h.must(f.a.Set(uint32(i), en, &h.pair(coeff.RatString(), "0").X))
			h.must(f.x.Data[i].Copy(h.m, h.pair(initial[i].RatString(), "0")))
			h.must(f.b.Data[i].Copy(h.m, h.pair(rhs.RatString(), "0")))
			f.types[i] = ColumnLowBound
			limit := new(big.Rat).Quo(rhs, coeff)
			if best == nil || limit.Cmp(best) < 0 {
				best = new(big.Rat).Set(limit)
			}
		}
		f.types[en] = ColumnLowBound
		profit := big.NewRat(int64(world%11+1), 13)
		h.must(h.m.SetString(&f.costs.Data[en], profit.RatString()))
		m, e := NewPairPrimalCoreSolver(f.input())
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
		h.check(&f.x.Data[en], best, new(big.Rat))
		for i := range ds {
			want := new(big.Rat).Sub(initial[i], new(big.Rat).Mul(ds[i], best))
			h.check(&f.x.Data[i], want, new(big.Rat))
			if want.Sign() < 0 {
				t.Fatal("oracle feasibility")
			}
		}
		actual := new(big.Rat).Mul(best, profit)
		lpSparseRationalCheck(t, h, &f.costs.Data[en], profit)
		got := h.pair("0", "0")
		h.must(got.Scale(h.m, &f.x.Data[en], &f.costs.Data[en]))
		h.check(got, actual, new(big.Rat))
		if off, e := m.AMultXIsOff(); e != nil || off {
			t.Fatal("equation", e, off)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
