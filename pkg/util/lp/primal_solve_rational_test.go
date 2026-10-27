// Source: lp_primal_core_solver.hpp:864-966 and tableau.h:99-208.
package lp

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalSolveLUIndependentBoundedOptima(t *testing.T) {
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
		steps, e := m.Solve()
		h.must(e)
		if m.Status != StatusOptimal || steps != 2 || m.enteringStep != nil {
			t.Fatal("Solve convergence", world, steps, m.Status)
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
		h.must(h.m.Mul(&f.costs.Data[en], &f.x.Data[en], got))
		lpSparseRationalCheck(t, h, got, actual)
		if off, e := m.AMultXIsOff(); e != nil || off {
			t.Fatal("equation", e, off)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalSolveTableauIndependentBoundedOptima(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 3)
		f.settings.SimplexStrategy = StrategyTableauCosts
		cap := big.NewRat(int64(1+world%17), int64(1+world%7))
		a := big.NewRat(int64(1+world%11), int64(1+world%5))

		h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
		h.must(f.a.Set(0, 1, lpScalarBoundValue(h, a.RatString())))
		h.must(f.a.Set(0, 2, lpScalarBoundValue(h, new(big.Rat).Neg(cap).RatString())))
		f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
		h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, cap.RatString())))
		for _, p := range []*numeral.Rational{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
			h.must(h.m.Copy(p, lpScalarBoundValue(h, "1")))
		}
		profit := big.NewRat(int64(1+world%9), 11)
		h.must(h.m.SetString(&f.costs.Data[1], profit.RatString()))
		m, e := NewRationalPrimalCoreSolver(f.input())
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
		lpSparseRationalCheck(t, h, &f.x.Data[1], optimum)
		lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
		cost := lpScalarBoundValue(h, "0")
		h.must(m.GetCost(cost))
		lpSparseRationalCheck(t, h, cost, new(big.Rat).Mul(optimum, profit))
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

func TestLPRationalSolveRowsIndependentFeasibilityAndInfeasibility(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		c := big.NewRat(int64(2+world%13), 7)
		f, m := lpRationalRowIterationFixture(t, h, new(big.Rat).Neg(c), new(big.Rat), big.NewRat(-1, 1), ColumnLowBound)
		infeasible := world%2 == 0
		if infeasible {
			f.types[1] = ColumnBoxed
			h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, new(big.Rat).Quo(c, big.NewRat(2, 1)).RatString())))
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
		lpSparseRationalCheck(t, h, &f.x.Data[1], c)
		lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

var lpErrRationalSolveCancel = errors.New("exact Solve cancellation service failure")

type lpRationalSolveCancelSequence struct {
	values         []bool
	calls, errorAt int
}

func (l *lpRationalSolveCancelSequence) CancelFlag() (bool, error) {
	l.calls++
	if l.errorAt == l.calls {
		return false, lpErrRationalSolveCancel
	}
	if l.calls <= len(l.values) {
		return l.values[l.calls-1], nil
	}
	return false, nil
}

func lpRationalSolveBoundedFixture(t *testing.T, h *lpPairHarness, strategy SimplexStrategy) (*lpRationalCoreFixture, *RationalPrimalCoreSolver) {
	f := lpNewRationalCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = strategy
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
	h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
	h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "-1")))
	f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
	h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "1")))
	for _, p := range []*numeral.Rational{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
		h.must(h.m.Copy(p, lpScalarBoundValue(h, "1")))
	}
	h.must(h.m.SetInt32(&f.costs.Data[1], 1))
	m, e := NewRationalPrimalCoreSolver(f.input())
	h.must(e)
	m.Names = lpFeasibleNames{}
	if f.settings.UseTableau() {
		h.must(m.InitInfSet())
		h.must(m.InitReducedCostsTableau())
	}
	return f, m
}

func TestLPRationalSolveLimitsAndCancellationOrder(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for _, phase := range []string{"early-feasible", "head-cancel", "head-error", "zero-limit", "one-limit", "exit-cancel", "transient-cancel", "exit-error", "second-head-cancel"} {
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				var f *lpRationalCoreFixture
				var m *RationalPrimalCoreSolver
				if strategy == StrategyTableauRows && phase != "early-feasible" {
					f, m = lpRationalRowIterationFixture(t, h, big.NewRat(-2, 1), new(big.Rat), big.NewRat(-1, 1), ColumnLowBound)
					f.types[1] = ColumnBoxed
					h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, "1")))
				} else {
					f, m = lpRationalSolveBoundedFixture(t, h, strategy)
				}
				limit := &lpRationalSolveCancelSequence{}
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
					wantStatus, wantCount, wantCalls, wantErr = StatusUnknown, 1, 1, lpErrRationalSolveCancel
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
						wantStatus, wantCount, wantCalls, wantErr = StatusUnknown, 1, 2, lpErrRationalSolveCancel
					} else {
						wantStatus, wantCount, wantCalls, wantErr = StatusFeasible, 2, 2, lpErrRationalSolveCancel
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

func TestLPRationalSolveReportingAndWriteFailure(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", strategy, fail), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalSolveBoundedFixture(t, h, strategy)
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

func TestLPRationalSolvePreservedFallbackStatuses(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for _, status := range []Status{StatusTentativeUnbounded, StatusUnstable} {
			t.Run(fmt.Sprintf("%d/%d", strategy, status), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = strategy
				h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
				h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
				f.types[0] = ColumnFree
				f.types[1] = ColumnBoxed
				h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, "1")))
				h.must(h.m.SetInt32(&f.costs.Data[1], 1))
				m, e := NewRationalPrimalCoreSolver(f.input())
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
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(1, 1))
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPRationalSolveUnboundedIndependentRays(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for world := 0; world < 80; world++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = strategy
			coefficient := big.NewRat(int64(1+world%13), int64(1+world%7))
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, coefficient.RatString())))
			f.types[0], f.types[1] = ColumnFree, ColumnLowBound
			h.must(h.m.SetInt32(&f.costs.Data[1], int32(1+world%11)))
			m, e := NewRationalPrimalCoreSolver(f.input())
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
			lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
			lpSparseRationalCheck(t, h, &f.x.Data[1], new(big.Rat))
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalSolveFeasibilitySearchIndependentSolutions(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for world := 0; world < 80; world++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = strategy
			gap := big.NewRat(int64(1+world%17), int64(1+world%7))
			a := big.NewRat(int64(1+world%11), int64(1+world%5))
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, new(big.Rat).Neg(a).RatString())))
			h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "-1")))
			f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
			negative := new(big.Rat).Neg(gap)
			h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, negative.RatString())))
			for _, p := range []*numeral.Rational{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
				h.must(h.m.Copy(p, lpScalarBoundValue(h, negative.RatString())))
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
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
			lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
			lpSparseRationalCheck(t, h, &f.x.Data[1], new(big.Rat).Quo(gap, a))
			if off, e := m.AMultXIsOff(); e != nil || off {
				t.Fatal("feasible equation", off, e)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalSolveInfeasibleIndependentBounds(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for world := 0; world < 80; world++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = strategy
			gap := big.NewRat(int64(1+world%17), int64(1+world%7))
			a := big.NewRat(int64(1+world%11), int64(1+world%5))
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, a.RatString())))
			h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "-1")))
			f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
			negative := new(big.Rat).Neg(gap)
			h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, negative.RatString())))
			for _, p := range []*numeral.Rational{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
				h.must(h.m.Copy(p, lpScalarBoundValue(h, negative.RatString())))
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
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
			lpSparseRationalCheck(t, h, &f.x.Data[0], negative)
			lpSparseRationalCheck(t, h, &f.x.Data[1], new(big.Rat))
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalSolveTableauStaleCostCorrection(t *testing.T) {
	for _, profit := range []int32{0, 1} {
		h := newLPPairHarness(t)
		f, m := lpRationalSolveBoundedFixture(t, h, StrategyTableauCosts)
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
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(int64(profit), 1))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalSolveStoppedAndInitializationPrefixes(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for _, phase := range []string{"stopped", "short-heading"} {
			if phase == "short-heading" && strategy == StrategyTableauRows {
				continue // A feasible row strategy does not read the heading.
			}
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalSolveBoundedFixture(t, h, strategy)
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

func TestLPRationalSolvePhysicalOwnership(t *testing.T) {
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
					f, m := lpRationalSolveBoundedFixture(t, h, strategy)
					huge := strings.Repeat("1234567890", 20)
					cap := lpScalarBoundValue(h, huge+"7/13")
					for _, p := range []*numeral.Rational{&f.x.Data[0], &f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
						h.must(h.m.Copy(p, cap))
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

// Cancellation is an observable service boundary. Mutating a model at that
// boundary checks the source's later factorization and final-validation reads.
type lpRationalSolveMutatingLimit struct {
	calls, at int
	mutate    func()
}

func (l *lpRationalSolveMutatingLimit) CancelFlag() (bool, error) {
	l.calls++
	if l.calls == l.at {
		l.mutate()
	}
	return false, nil
}

func TestLPRationalSolveSingularFallbackAfterBoundFlip(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 3)
		f.settings.SimplexStrategy = strategy
		for i := uint32(0); i < 2; i++ {
			h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
		}
		h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "1")))
		f.types[0], f.types[1], f.types[2] = ColumnFree, ColumnFree, ColumnBoxed
		h.must(h.m.SetInt32(&f.up.Data[2], 1))
		h.must(h.m.SetInt32(&f.costs.Data[2], 1))
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		if strategy == StrategyTableauCosts {
			h.must(m.InitReducedCostsTableau())
		}
		m.Names = lpFeasibleNames{}
		m.Status = StatusTentativeUnbounded
		limit := &lpRationalSolveMutatingLimit{at: 1}
		limit.mutate = func() {
			// The completed initial basis is valid. The service boundary changes
			// its unused second column to the first. The entering direction and
			// resulting model still satisfy A*x=b, but refactorization is singular.
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
			v, err := f.a.ColumnValue(f.a.Columns[1][0])
			h.must(err)
			h.must(h.m.SetInt32(v, 0))
		}
		f.settings.SetResourceLimit(limit)
		count, e := m.Solve()
		if sourceDebugEnabled && strategy == StrategyTableauCosts {
			if e != ErrAssertion || count != 1 || m.Status != StatusTentativeUnbounded || limit.calls != 1 {
				t.Fatal(strategy, count, m.Status, limit.calls, e)
			}
		} else if e != nil {
			t.Fatal(strategy, count, m.Status, limit.calls, e)
		}
		calls := 1
		if strategy == StrategyTableauCosts {
			calls = 2
		}
		if !(sourceDebugEnabled && strategy == StrategyTableauCosts) && (count != 1 || m.Status != StatusFloatingPointError || limit.calls != calls || m.Factorization == nil || m.Factorization.GetStatus() == LUOK) {
			t.Fatal(strategy, count, m.Status, limit.calls)
		}
		lpSparseRationalCheck(t, h, &f.x.Data[2], big.NewRat(1, 1))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalSolveFinalValidationReadsLiveBounds(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts} {
		for _, metadata := range []bool{false, true} {
			if strategy == StrategyLU && metadata {
				continue
			}
			h := newLPPairHarness(t)
			f, m := lpRationalSolveBoundedFixture(t, h, strategy)
			at := 2
			if strategy == StrategyTableauCosts {
				at = 4
			}
			savedTypes := f.types
			limit := &lpRationalSolveMutatingLimit{at: at}
			limit.mutate = func() {
				if metadata {
					f.types = nil
					return
				}
				h.must(h.m.SetInt32(&f.low.Data[2], 0))
				h.must(h.m.SetInt32(&f.up.Data[2], 0))
			}
			f.settings.SetResourceLimit(limit)
			count, err := m.Solve()
			f.types = savedTypes
			want := error(nil)
			if sourceDebugEnabled {
				want = ErrAssertion
				if metadata {
					want = ErrBounds
				}
			}
			if count != 2 || err != want || m.Status != StatusOptimal || limit.calls != at {
				t.Fatal(strategy, metadata, count, err, want, m.Status, limit.calls)
			}
			if err != nil {
				if _, e := m.Solve(); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalSolveTableauLiveStaleCostCorrection(t *testing.T) {
	for _, profit := range []int32{0, 1} {
		h := newLPPairHarness(t)
		f, m := lpRationalSolveBoundedFixture(t, h, StrategyTableauCosts)
		h.must(h.m.SetInt32(&f.costs.Data[1], profit))
		h.must(m.InitReducedCostsTableau())
		h.must(m.BackupAndNormalizeCosts())
		// Initialization sees a valid objective. The source service boundary
		// then installs stale infeasibility costs before terminal selection.
		f.settings.BackupCosts = false
		limit := &lpRationalSolveMutatingLimit{at: 1}
		limit.mutate = func() {
			h.must(h.m.SetInt32(&f.costs.Data[0], 1))
			h.must(h.m.SetInt32(&f.costs.Data[1], 0))
			m.UsingInfeasCosts = true
			h.must(m.fillReducedCostsTableau())
		}
		f.settings.SetResourceLimit(limit)
		var count uint32
		var err error
		lpCaptureFeasibleStdout(t, func() { count, err = m.Solve() })
		if sourceDebugEnabled {
			// Status selection asserts on switching costs before correction.
			if err != ErrAssertion || count != 1 || limit.calls != 1 {
				t.Fatal("live stale-cost assertion prefix", count, limit.calls, err)
			}
		} else {
			h.must(err)
			wantCount := uint32(1)
			if profit != 0 {
				wantCount = 3
			}
			if count != wantCount || m.Status != StatusOptimal || m.UsingInfeasCosts {
				t.Fatal("live cost correction", count, m.Status, m.UsingInfeasCosts)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(int64(profit), 1))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
