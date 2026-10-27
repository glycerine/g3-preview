// Source: lp_primal_core_solver.hpp:864-966 and tableau.h:99-208.
package lp

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactSolveLUIndependentBoundedOptima(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

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
		steps, e := m.Solve()
		h.must(e)
		if m.Status != StatusOptimal || steps != 2 || m.enteringStep != nil {
			t.Fatal("Solve convergence", world, steps, m.Status)
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

func TestLPExactSolveTableauIndependentBoundedOptima(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 3)
		f.settings.SimplexStrategy = StrategyTableauCosts
		cap := big.NewRat(int64(1+world%17), int64(1+world%7))
		a := big.NewRat(int64(1+world%11), int64(1+world%5))
		ey := big.NewRat(int64(world%5-2), 13)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.a.Set(0, 1, &h.pair(a.RatString(), "0").X))
		h.must(f.a.Set(0, 2, &h.pair(new(big.Rat).Neg(cap).RatString(), "0").X))
		f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
		h.must(f.x.Data[0].Copy(h.m, h.pair(cap.RatString(), new(big.Rat).Mul(cap, ey).RatString())))
		for _, p := range []*RationalPair{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
			h.must(p.Copy(h.m, h.pair("1", ey.RatString())))
		}
		profit := big.NewRat(int64(1+world%9), 11)
		h.must(h.m.SetString(&f.costs.Data[1], profit.RatString()))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		m.Names = lpFeasibleNames{}
		h.must(m.InitInfSet())
		h.must(m.InitReducedCostsTableau())
		limit := &lpTestResourceLimit{}
		f.settings.SetResourceLimit(limit)
		steps, e := m.Solve()
		h.must(e)
		optimum := new(big.Rat).Quo(cap, a)
		if steps != 2 || m.Status != StatusOptimal || m.Factorization != nil || limit.calls != 4 {
			t.Fatal("tableau solve", world, steps, m.Status, limit.calls)
		}
		h.check(&f.x.Data[1], optimum, new(big.Rat).Mul(optimum, ey))
		h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
		cost := h.pair("0", "0")
		h.must(m.GetCost(cost))
		h.check(cost, new(big.Rat).Mul(optimum, profit), new(big.Rat).Mul(new(big.Rat).Mul(optimum, profit), ey))
		off, e := m.AMultXIsOff()
		h.must(e)
		if off {
			t.Fatal("AX", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactSolveRowsIndependentFeasibilityAndInfeasibility(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		c := big.NewRat(int64(2+world%13), 7)
		f, m := lpRowIterationFixture(t, h, new(big.Rat).Neg(c), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
		infeasible := world%2 == 0
		if infeasible {
			f.types[1] = ColumnBoxed
			h.must(f.up.Data[1].Copy(h.m, h.pair(new(big.Rat).Quo(c, big.NewRat(2, 1)).RatString(), "0")))
		}
		only := world/2%2 == 0
		m.LookForFeasibleSolutionOnly = only
		limit := &lpTestResourceLimit{}
		f.settings.SetResourceLimit(limit)
		steps, e := m.Solve()
		h.must(e)
		wantStatus, wantSteps, wantCalls := StatusOptimal, uint32(1), 2
		if infeasible {
			wantStatus = StatusInfeasible
			wantSteps = 2
			wantCalls = 4
		}
		if steps != wantSteps || m.Status != wantStatus || limit.calls != wantCalls || m.Factorization != nil {
			t.Fatal(world, steps, m.Status, limit.calls, only)
		}
		h.check(&f.x.Data[1], c, new(big.Rat))
		h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

var lpErrExactSolveCancel = errors.New("exact Solve cancellation service failure")

type lpExactSolveCancelSequence struct {
	values         []bool
	calls, errorAt int
}

func (l *lpExactSolveCancelSequence) CancelFlag() (bool, error) {
	l.calls++
	if l.errorAt == l.calls {
		return false, lpErrExactSolveCancel
	}
	if l.calls <= len(l.values) {
		return l.values[l.calls-1], nil
	}
	return false, nil
}

func lpExactSolveBoundedFixture(t *testing.T, h *lpPairHarness, strategy SimplexStrategy) (*lpPairCoreFixture, *PairPrimalCoreSolver) {
	f := lpNewPairCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = strategy
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.a.Set(0, 1, &h.pair("1", "0").X))
	h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
	f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
	h.must(f.x.Data[0].Copy(h.m, h.pair("1", "0")))
	for _, p := range []*RationalPair{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
		h.must(p.Copy(h.m, h.pair("1", "0")))
	}
	h.must(h.m.SetInt32(&f.costs.Data[1], 1))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	m.Names = lpFeasibleNames{}
	if f.settings.UseTableau() {
		h.must(m.InitInfSet())
		h.must(m.InitReducedCostsTableau())
	}
	return f, m
}

func TestLPExactSolveLimitsAndCancellationOrder(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for _, phase := range []string{"early-feasible", "head-cancel", "head-error", "zero-limit", "one-limit", "exit-cancel", "transient-cancel", "exit-error", "second-head-cancel"} {
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				var f *lpPairCoreFixture
				var m *PairPrimalCoreSolver
				if strategy == StrategyTableauRows && phase != "early-feasible" {
					f, m = lpRowIterationFixture(t, h, big.NewRat(-2, 1), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
					f.types[1] = ColumnBoxed
					h.must(f.up.Data[1].Copy(h.m, h.pair("1", "0")))
				} else {
					f, m = lpExactSolveBoundedFixture(t, h, strategy)
				}
				limit := &lpExactSolveCancelSequence{}
				f.settings.SetResourceLimit(limit)
				m.TotalIterations = 91
				f.settings.Stats.TotalIterations = 17
				wantStatus, wantCount, wantCalls, wantErr := StatusOptimal, uint32(2), 2, error(nil)
				if strategy != StrategyLU {
					wantCalls = 4
				}
				if strategy == StrategyTableauRows {
					wantStatus = StatusInfeasible
				}
				switch phase {
				case "early-feasible":
					m.LookForFeasibleSolutionOnly = true
					wantStatus, wantCount, wantCalls = StatusFeasible, 0, 0
					limit.values = []bool{true}
				case "head-cancel":
					limit.values = []bool{true}
					wantStatus, wantCount, wantCalls = StatusTimeExhausted, 1, 1
				case "head-error":
					limit.errorAt = 1
					wantStatus, wantCount, wantCalls, wantErr = StatusUnknown, 1, 1, lpErrExactSolveCancel
				case "zero-limit":
					f.settings.MaxTotalIterations = 0
					wantStatus, wantCount = StatusUnknown, 1
					wantCalls = 1
					if strategy == StrategyLU {
						wantStatus = StatusFeasible
					}
					if strategy != StrategyLU {
						wantCalls = 2
					}
				case "one-limit":
					f.settings.MaxTotalIterations = 1
				case "exit-cancel":
					limit.values = []bool{false, true, true}
					if strategy != StrategyLU {
						wantStatus, wantCount, wantCalls = StatusCancelled, 1, 3
					} else {
						wantStatus, wantCount, wantCalls = StatusTimeExhausted, 2, 2
					}
				case "transient-cancel":
					limit.values = []bool{false, true, false}
					if strategy != StrategyLU {
						wantStatus, wantCount, wantCalls = StatusUnknown, 1, 3
					} else {
						wantStatus, wantCount, wantCalls = StatusTimeExhausted, 2, 2
					}
				case "exit-error":
					limit.errorAt = 2
					if strategy != StrategyLU {
						wantStatus, wantCount, wantCalls, wantErr = StatusUnknown, 1, 2, lpErrExactSolveCancel
					} else {
						wantStatus, wantCount, wantCalls, wantErr = StatusFeasible, 2, 2, lpErrExactSolveCancel
					}
				case "second-head-cancel":
					limit.values = []bool{false, false, true}
					if strategy != StrategyLU {
						wantStatus, wantCount, wantCalls = StatusTimeExhausted, 2, 3
					}
				}
				count, err := m.Solve()
				if count != wantCount || err != wantErr || m.Status != wantStatus || limit.calls != wantCalls || f.settings.Stats.TotalIterations != 17+wantCount {
					t.Fatal(count, err, m.Status, limit.calls, wantCount, wantErr, wantStatus, wantCalls)
				}
				if err != nil {
					if _, retry := m.Solve(); retry != ErrExactVectorPending {
						t.Fatal("retry", retry)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactSolveReportingAndWriteFailure(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", strategy, fail), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpExactSolveBoundedFixture(t, h, strategy)
				f.settings.PrintStatistics = true
				f.settings.ReportFrequency = 1
				var out bytes.Buffer
				f.settings.MessageOut = &out
				if fail {
					f.settings.MessageOut = &lpResidualFailWriter{remaining: 3}
				}
				limit := &lpTestResourceLimit{}
				f.settings.SetResourceLimit(limit)
				count, err := m.Solve()
				if fail {
					if err != lpErrResidualWrite || count != 1 || limit.calls != 0 || m.Status != StatusUnknown {
						t.Fatal(count, err, limit.calls, m.Status)
					}
				} else {
					h.must(err)
					prefix := "feas "
					if strategy != StrategyLU {
						prefix = "feas t "
					}
					if count != 2 || !bytes.HasPrefix(out.Bytes(), []byte(prefix)) || bytes.Count(out.Bytes(), []byte(prefix)) != 2 {
						t.Fatal("report", out.String(), count)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactSolvePreservedFallbackStatuses(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for _, status := range []Status{StatusTentativeUnbounded, StatusUnstable} {
			t.Run(fmt.Sprintf("%d/%d", strategy, status), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = strategy
				h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
				h.must(f.a.Set(0, 1, &h.pair("1", "0").X))
				f.types[0] = ColumnFree
				f.types[1] = ColumnBoxed
				h.must(f.up.Data[1].Copy(h.m, h.pair("1", "0")))
				h.must(h.m.SetInt32(&f.costs.Data[1], 1))
				m, e := NewPairPrimalCoreSolver(f.input())
				h.must(e)
				m.Status = status
				m.Names = lpFeasibleNames{}
				if strategy == StrategyTableauCosts {
					h.must(m.InitReducedCostsTableau())
				}
				f.settings.SetResourceLimit(&lpTestResourceLimit{})
				count, err := m.Solve()
				want := error(nil)
				if sourceDebugEnabled && (status == StatusUnstable || strategy == StrategyTableauCosts) {
					want = ErrAssertion
				}
				if err != want {
					t.Fatal(count, err, want)
				}
				if want != nil {
					if count != 1 || m.Status != status {
						t.Fatal("assert prefix", count, m.Status)
					}
				} else if count != 2 || m.Status != StatusOptimal || m.Factorization == nil {
					t.Fatal("fallback", count, m.Status)
				}
				h.check(&f.x.Data[1], big.NewRat(1, 1), new(big.Rat))
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactSolveUnboundedIndependentRays(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for world := 0; world < 80; world++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = strategy
			coefficient := big.NewRat(int64(1+world%13), int64(1+world%7))
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair(coefficient.RatString(), "0").X))
			f.types[0], f.types[1] = ColumnFree, ColumnLowBound
			h.must(h.m.SetInt32(&f.costs.Data[1], int32(1+world%11)))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			m.Names = lpFeasibleNames{}
			if strategy == StrategyTableauCosts {
				h.must(m.InitReducedCostsTableau())
			}
			limit := &lpTestResourceLimit{}
			f.settings.SetResourceLimit(limit)
			count, e := m.Solve()
			h.must(e)
			wantCalls := 1
			if strategy == StrategyTableauCosts {
				wantCalls = 2
			}
			if count != 1 || m.Status != StatusUnbounded || limit.calls != wantCalls {
				t.Fatal(strategy, world, count, m.Status, limit.calls)
			}
			// Independent feasible rays: (-a*t,t) satisfies the equation for
			// every positive t, respects the entering lower bound and makes
			// the positive objective grow without limit.
			for _, distance := range []int64{1, 17, 1000000} {
				entering := big.NewRat(distance, 1)
				basic := new(big.Rat).Neg(new(big.Rat).Mul(coefficient, entering))
				if new(big.Rat).Add(basic, new(big.Rat).Mul(coefficient, entering)).Sign() != 0 || entering.Sign() <= 0 {
					t.Fatal("invalid independent ray")
				}
			}
			h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
			h.check(&f.x.Data[1], new(big.Rat), new(big.Rat))
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactSolveFeasibilitySearchIndependentSolutions(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for world := 0; world < 80; world++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = strategy
			gap := big.NewRat(int64(1+world%17), int64(1+world%7))
			a := big.NewRat(int64(1+world%11), int64(1+world%5))
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair(new(big.Rat).Neg(a).RatString(), "0").X))
			h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
			f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
			negative := new(big.Rat).Neg(gap)
			h.must(f.x.Data[0].Copy(h.m, h.pair(negative.RatString(), "0")))
			for _, p := range []*RationalPair{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
				h.must(p.Copy(h.m, h.pair(negative.RatString(), "0")))
			}
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			m.Names = lpFeasibleNames{}
			m.LookForFeasibleSolutionOnly = true
			if strategy == StrategyTableauCosts {
				h.must(m.InitInfSet())
				h.must(m.InitReducedCostsTableau())
			}
			f.settings.SetResourceLimit(&lpTestResourceLimit{})
			count, e := m.Solve()
			h.must(e)
			if count != 1 || !m.CurrentXIsFeasible() {
				t.Fatal("feasibility search", strategy, world, count, m.Status)
			}
			h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
			h.check(&f.x.Data[1], new(big.Rat).Quo(gap, a), new(big.Rat))
			if off, e := m.AMultXIsOff(); e != nil || off {
				t.Fatal("feasible equation", off, e)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactSolveInfeasibleIndependentBounds(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for world := 0; world < 80; world++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = strategy
			gap := big.NewRat(int64(1+world%17), int64(1+world%7))
			a := big.NewRat(int64(1+world%11), int64(1+world%5))
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair(a.RatString(), "0").X))
			h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
			f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
			negative := new(big.Rat).Neg(gap)
			h.must(f.x.Data[0].Copy(h.m, h.pair(negative.RatString(), "0")))
			for _, p := range []*RationalPair{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
				h.must(p.Copy(h.m, h.pair(negative.RatString(), "0")))
			}
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			m.Names = lpFeasibleNames{}
			m.LookForFeasibleSolutionOnly = world%2 == 0
			if strategy == StrategyTableauCosts {
				h.must(m.InitInfSet())
				h.must(m.InitReducedCostsTableau())
			}
			f.settings.SetResourceLimit(&lpTestResourceLimit{})
			count, e := m.Solve()
			h.must(e)
			// The equation requires x0+a*x1=-gap. Its left side is
			// nonnegative under both lower bounds, so no solution exists.
			if gap.Sign() <= 0 || a.Sign() <= 0 || count != 1 || m.Status != StatusInfeasible {
				t.Fatal("infeasibility", strategy, world, count, m.Status)
			}
			h.check(&f.x.Data[0], negative, new(big.Rat))
			h.check(&f.x.Data[1], new(big.Rat), new(big.Rat))
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactSolveTableauStaleCostCorrection(t *testing.T) {
	for _, profit := range []int32{0, 1} {
		h := newLPPairHarness(t)
		f, m := lpExactSolveBoundedFixture(t, h, StrategyTableauCosts)
		h.must(h.m.SetInt32(&f.costs.Data[1], profit))
		h.must(m.BackupAndNormalizeCosts())
		// A stale infeasibility cost hides the profitable nonbasic column.
		// Native Solve must double-check and restore the actual objective.
		// Source debug rejects this input in InitRunTableau before the loop.
		h.must(h.m.SetInt32(&f.costs.Data[0], 1))
		h.must(h.m.SetInt32(&f.costs.Data[1], 0))
		m.UsingInfeasCosts = true
		h.must(m.fillReducedCostsTableau())
		f.settings.BackupCosts = false
		f.settings.SetResourceLimit(&lpTestResourceLimit{})
		var count uint32
		var err error
		lpCaptureFeasibleStdout(t, func() { count, err = m.Solve() })
		if sourceDebugEnabled {
			if err != ErrAssertion || count != 0 {
				t.Fatal("debug stale-cost initialization", count, err)
			}
		} else {
			h.must(err)
			wantCount := uint32(1)
			if profit != 0 {
				wantCount = 3
			}
			if count != wantCount || m.Status != StatusOptimal || m.UsingInfeasCosts {
				t.Fatal("native cost correction", count, m.Status, m.UsingInfeasCosts)
			}
			h.check(&f.x.Data[1], big.NewRat(int64(profit), 1), new(big.Rat))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactSolveStoppedAndInitializationPrefixes(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for _, phase := range []string{"stopped", "short-heading"} {
			if phase == "short-heading" && strategy == StrategyTableauRows {
				continue // A feasible row strategy does not read the heading.
			}
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpExactSolveBoundedFixture(t, h, strategy)
				m.TotalIterations = 91
				limit := &lpTestResourceLimit{}
				f.settings.SetResourceLimit(limit)
				heading := *m.Heading
				wantCount := uint32(0)
				wantCalls := 0
				switch phase {
				case "stopped":
					m.stopped = true
					wantCount = 91
				case "short-heading":
					*m.Heading = heading[:0]
					if strategy == StrategyTableauCosts && !sourceDebugEnabled {
						wantCount, wantCalls = 1, 1
					}
				}
				count, e := m.Solve()
				if e == nil || count != wantCount || limit.calls != wantCalls {
					t.Fatal("initialization prefix", count, e, limit.calls)
				}
				if _, retry := m.SolveWithTableau(); retry != ErrExactVectorPending {
					t.Fatal("stopped direct tableau retry", retry)
				}
				*m.Heading = heading
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactSolvePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%d/%t/%d", strategy, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpExactSolveBoundedFixture(t, h, strategy)
					huge := strings.Repeat("1234567890", 20)
					cap := h.pair(huge+"7/13", "0")
					for _, p := range []*RationalPair{&f.x.Data[0], &f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
						h.must(p.Copy(h.m, cap))
					}
					h.must(h.m.SetString(&f.costs.Data[1], huge+"3/17"))
					if strategy == StrategyTableauCosts {
						h.must(m.InitReducedCostsTableau())
					}
					f.settings.SetResourceLimit(&lpTestResourceLimit{})
					h.budget.FailAfter(ordinal)
					count, err := m.Solve()
					h.budget.FailAfter(-1)
					if count != m.TotalIterations {
						t.Fatal("lost iteration prefix", count, m.TotalIterations)
					}
					if err == nil {
						success, done = true, true
						if count != 2 || m.Status != StatusOptimal {
							t.Fatal("physical success", count, m.Status)
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if retry, e := m.Solve(); e != ErrExactVectorPending || retry != count {
							t.Fatal("stopped retry", retry, count, e)
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
			if !success || (direct && failures == 0) {
				t.Fatal("physical campaign", strategy, direct, success, failures)
			}
			t.Log("physical campaign", strategy, direct, "refusals", failures)
		}
	}
}
