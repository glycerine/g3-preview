// Source: src/util/lp/lp_primal_core_solver_tableau.h:23-41.
// Independent exact-rational models supplement retained source assertions.
package lp

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPRationalTableauEnterIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	seen := map[string]int{}
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		n := 1 + world%4
		en := uint32(n)
		f := lpNewRationalCoreFixture(t, h, uint32(n), en+1)
		f.settings.SimplexStrategy = StrategyTableauCosts
		direction := int64(1)
		if world%2 == 0 {
			direction = -1
		}
		ds := make([]*big.Rat, n)
		bestX := (*big.Rat)(nil)
		leaving := -1
		var candidates []int
		for i := 0; i < n; i++ {
			ds[i] = big.NewRat(int64(1+(world+i)%7), int64(3+i))
			if (world+i)%3 == 0 {
				ds[i].Neg(ds[i])
			}
			diagonal := big.NewRat(1, 1)
			h.must(f.a.Set(uint32(i), uint32(i), lpScalarBoundValue(h, diagonal.RatString())))
			coeff := new(big.Rat).Mul(diagonal, ds[i])
			h.must(f.a.Set(uint32(i), en, lpScalarBoundValue(h, coeff.RatString())))
			kind := ColumnType((world/3 + i) % 5)
			if world%11 == 0 {
				kind = ColumnFree
			}
			f.types[i] = kind
			lx := big.NewRat(int64(1+i+world%9), 7)
			if kind == ColumnFixed {
				lx.SetInt64(0)
			}
			abs := new(big.Rat).Abs(ds[i])
			bx := new(big.Rat).Mul(abs, lx)
			h.must(h.m.Copy(&f.up.Data[i], lpScalarBoundValue(h, bx.RatString())))
			h.must(h.m.Copy(&f.low.Data[i], lpScalarBoundValue(h, new(big.Rat).Neg(bx).RatString())))
			rising := ds[i].Sign()*int(direction) < 0
			eligible := kind == ColumnBoxed || kind == ColumnFixed || (kind == ColumnLowBound && !rising) || (kind == ColumnUpperBound && rising)
			if eligible {
				cmp := -1
				if bestX != nil {
					cmp = lx.Cmp(bestX)
				}
				if cmp < 0 {
					bestX = new(big.Rat).Set(lx)
					leaving = i
					candidates = []int{i}
				} else if cmp == 0 {
					candidates = append(candidates, i)
				}
				// All actual rows have two cells: equal ratios remain tied in stored order.
			}
		}
		f.types[en] = ColumnFree
		if world%5 == 0 {
			f.types[en] = ColumnBoxed
			h.must(h.m.Copy(&f.low.Data[en], lpScalarBoundValue(h, "-1/100")))
			h.must(h.m.Copy(&f.up.Data[en], lpScalarBoundValue(h, "1/100")))
			bx := big.NewRat(1, 100)
			if bestX == nil || bx.Cmp(bestX) < 0 {
				bestX = bx
				leaving = int(en)
			}
		}
		h.must(h.m.SetInt32(&f.costs.Data[en], int32(-direction)))
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, en+1)
		m.RowsNZ = make([]uint32, n)
		for i := range m.RowsNZ {
			m.RowsNZ[i] = uint32(i + 1)
		}
		h.must(m.InitReducedCostsTableau())
		m.NonBasisList = []uint32{en}
		m.SignOfEnteringDelta = int32(direction)
		m.LookForFeasibleSolutionOnly = world/2%2 == 0
		m.IterationsNoCostGrowth = 53
		m.Status = StatusOptimal
		seed := uint32(world + 97)
		m.Settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		if leaving >= 0 && leaving != int(en) && len(candidates) > 1 {
			leaving = candidates[rng.Next()%uint32(len(candidates))]
		}
		h.must(m.AdvanceOnEnteringTableau(int32(en)))
		if m.Settings.RandomNext() != rng.Next() {
			t.Fatal("RNG prefix", world, leaving)
		}
		if m.enteringStep != nil {
			t.Fatal("local step retained on success")
		}
		if leaving < 0 {
			seen["unbounded"]++
			if m.Status != StatusUnbounded || m.IterationsNoCostGrowth != 53 || f.basis[0] != 0 {
				t.Fatal("unbounded prefix", world)
			}
			for j := range f.x.Data {
				lpSparseRationalCheck(t, h, &f.x.Data[j], new(big.Rat))
			}
		} else {
			sx := new(big.Rat).Mul(bestX, big.NewRat(direction, 1))
			for i := range ds {
				lpSparseRationalCheck(t, h, &f.x.Data[i], new(big.Rat).Neg(new(big.Rat).Mul(ds[i], sx)))
			}
			lpSparseRationalCheck(t, h, &f.x.Data[en], sx)
			wantCounter := uint32(0)
			if leaving == int(en) {
				seen["flip"]++
				if m.Status != StatusOptimal || f.basis[0] != 0 || m.NonBasisList[0] != en {
					t.Fatal("flip metadata", world)
				}
				if m.LookForFeasibleSolutionOnly {
					wantCounter = 53
				}
			} else {
				seen["pivot"]++
				if m.Status != StatusOptimal || f.basis[leaving] != en {
					t.Fatal("pivot metadata", world)
				}
				if bestX.Sign() == 0 {
					wantCounter = 53
					seen["zero"]++
				}
				if !m.LookForFeasibleSolutionOnly {
					if m.NonBasisList[0] != uint32(leaving) {
						t.Fatal("list")
					}
					for j := range f.costs.Data {
						want := new(big.Rat)
						if j == leaving {
							want.Quo(big.NewRat(direction, 1), ds[leaving])
						}
						lpSparseRationalCheck(t, h, &m.D.Data[j], want)
					}
				} else if m.NonBasisList[0] != en {
					t.Fatal("early list")
				}
			}
			if m.IterationsNoCostGrowth != wantCounter {
				t.Fatal("counter", world, m.IterationsNoCostGrowth, wantCounter)
			}
		}
		if m.Factorization != nil {
			t.Fatal("tableau created LU factorization")
		}
		if leaving >= 0 && leaving != int(en) && m.LookForFeasibleSolutionOnly {
			for j := range f.costs.Data {
				want := new(big.Rat)
				if j == leaving {
					want.Quo(big.NewRat(direction, 1), ds[leaving])
				}
				lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	for _, kind := range []string{"pivot", "flip", "unbounded", "zero"} {
		if seen[kind] == 0 {
			t.Fatal("unexercised", kind)
		}
	}
}

// A fixed companion column encodes the capacity in a homogeneous row, so the
// independently checked optimum does not rely on nonpivot RHS updates.
func TestLPRationalTableauIterationIndependentOptima(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 3)
		f.settings.SimplexStrategy = StrategyTableauCosts
		cap := big.NewRat(int64(1+world%17), int64(1+world%7))
		coeff := big.NewRat(int64(1+world%11), int64(1+world%5))
		h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
		h.must(f.a.Set(0, 1, lpScalarBoundValue(h, coeff.RatString())))
		h.must(f.a.Set(0, 2, lpScalarBoundValue(h, new(big.Rat).Neg(cap).RatString())))
		f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
		h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, cap.RatString())))
		for _, p := range []*numeral.Rational{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
			h.must(h.m.Copy(p, lpScalarBoundValue(h, "1")))
		}
		h.must(h.m.SetInt32(&f.costs.Data[1], 1))
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 0))
		h.must(m.InitInfSet())
		h.must(m.InitReducedCostsTableau())
		h.must(m.InitRunTableau())
		m.NonBasisList = []uint32{2, 1}
		m.TotalIterations = 73
		m.IterationsNoCostGrowth = 29
		h.must(m.OneIterationTableau())
		optimum := new(big.Rat).Quo(cap, coeff)
		lpSparseRationalCheck(t, h, &f.x.Data[1], optimum)
		lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
		if f.basis[0] != 1 || m.InfSet.Size() != 0 || m.Factorization != nil || m.IterationsNoCostGrowth != 0 {
			t.Fatal("pivot", world)
		}
		for repeat := 0; repeat < 3; repeat++ {
			h.must(m.OneIterationTableau())
			if m.Status != StatusOptimal || m.TotalIterations != 73 || m.enteringStep != nil {
				t.Fatal("terminal iteration", world, m.Status)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[1], optimum)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauIterationDispatchAndStops(t *testing.T) {
	for _, phase := range []string{"unbounded", "optimal", "infeasible", "pricing-error", "advance-error", "pending-step", "stopped", "membership", "membership-error"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.NonBasisList = []uint32{1}
			m.BasisSortCounter = 7
			m.TotalIterations = 41
			h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 0))
			h.must(h.m.SetInt32(&m.D.Data[1], -1))
			savedOffset := uint32(0)
			want := error(nil)
			status := StatusUnknown
			switch phase {
			case "unbounded":
				status = StatusUnbounded
			case "optimal":
				h.must(h.m.SetInt32(&m.D.Data[1], 0))
				status = StatusOptimal
			case "infeasible":
				h.must(h.m.SetInt32(&m.D.Data[1], 0))
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "-1")))
				h.must(m.InfSet.Insert(0))
				m.UsingInfeasCosts = true
				status = StatusInfeasible
			case "pricing-error":
				f.types = f.types[:1]
				want = ErrBounds
			case "advance-error":
				h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
				savedOffset = f.a.Columns[1][0].Offset
				f.a.Columns[1][0].Offset = 9
				want = ErrBounds
			case "pending-step":
				m.enteringStep = NewIndexedRationalVector(h.m, 1)
				want = ErrExactVectorPending
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
			case "membership", "membership-error":
				h.must(h.m.SetInt32(&m.D.Data[1], 0))
				h.must(m.InfSet.Insert(1))
				m.UsingInfeasCosts = true
				m.Names = lpFeasibleNames{}
				status = StatusInfeasible
				if sourceDebugEnabled {
					want = ErrAssertion
					if phase == "membership-error" {
						m.Names = nil
						want = ErrBounds
					}
				}
			}
			var got error
			output := lpCaptureFeasibleStdout(t, func() { got = m.OneIterationTableau() })
			if phase == "advance-error" {
				if m.enteringStep == nil {
					t.Fatal("did not reach advancement")
				}
				f.a.Columns[1][0].Offset = savedOffset
			}
			if got != want {
				t.Fatal(phase, got, want, output)
			}
			if got == nil && m.Status != status {
				t.Fatal("status", phase, m.Status, status)
			}
			if m.TotalIterations != 41 {
				t.Fatal("iteration count belongs to outer loop")
			}
			if got != nil {
				if e = m.OneIterationTableau(); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalTableauEnterPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, profile := range []string{"pivot", "flip", "unbounded"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 500; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", profile, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 2)
					f.settings.SimplexStrategy = StrategyTableauCosts
					h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
					h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "2/7")))
					huge := strings.Repeat("1234567890", 100)
					f.types[0] = ColumnBoxed
					f.types[1] = ColumnFree
					h.must(h.m.Copy(&f.low.Data[0], lpScalarBoundValue(h, "-"+huge+"1/11")))
					h.must(h.m.Copy(&f.up.Data[0], lpScalarBoundValue(h, huge+"1/11")))
					if profile == "flip" {
						f.types[1] = ColumnBoxed
						h.must(h.m.Copy(&f.low.Data[1], lpScalarBoundValue(h, "-1/7")))
						h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, "1/7")))
					} else if profile == "unbounded" {
						f.types[0] = ColumnFree
					}
					h.must(h.m.SetInt32(&f.costs.Data[1], 1))
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					h.must(m.InitReducedCostsTableau())
					m.SignOfEnteringDelta = 1
					m.NonBasisList = []uint32{1}
					h.budget.FailAfter(ordinal)
					err := m.AdvanceOnEnteringTableau(1)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						if m.enteringStep != nil {
							t.Fatal("local retained")
						}
						switch profile {
						case "pivot":
							expected := new(big.Rat).Mul(mustLPRat(t, huge+"1/11"), big.NewRat(7, 2))
							lpSparseRationalCheck(t, h, &f.x.Data[1], expected)
							if f.basis[0] != 1 {
								t.Fatal("pivot basis")
							}
						case "flip":
							lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(1, 7))
						case "unbounded":
							if m.Status != StatusUnbounded {
								t.Fatal("status")
							}
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.enteringStep == nil {
							t.Fatal("stopped child lost local")
						}
						if e = m.AdvanceOnEnteringTableau(1); e != ErrExactVectorPending {
							t.Fatal("retry", e)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted child capture")
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if t.Failed() {
					return
				}
				if done {
					break
				}
			}
			if !success || (direct && profile == "pivot" && failures == 0) {
				t.Fatal("campaign", profile, direct, failures)
			}
		}
	}
}

func TestLPRationalTableauEnterRejectedIndexOwnsLocal(t *testing.T) {
	for _, index := range []int32{-1, 9} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 0, 1)
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		m.Status = StatusOptimal
		if e = m.AdvanceOnEnteringTableau(index); e != ErrBounds {
			t.Fatal(e)
		}
		if m.enteringStep == nil || m.Status != StatusOptimal {
			t.Fatal("source prefix")
		}
		if e = m.AdvanceOnEnteringTableau(0); e != ErrExactVectorPending {
			t.Fatal(e)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauEnterPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, iteration := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			f.types[0] = ColumnFree
			f.types[1] = ColumnBoxed
			h.must(h.m.Copy(&f.low.Data[1], lpScalarBoundValue(h, "-1/7")))
			h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, "1/7")))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.NonBasisList = []uint32{1}
			m.SignOfEnteringDelta = 1
			m.ColumnsNZ = []uint32{1, 1, 1}
			h.must(h.m.SetInt32(&m.D.Data[1], 1))
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, err := h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if iteration {
					err = m.OneIterationTableau()
				} else {
					err = m.AdvanceOnEnteringTableau(1)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) {
					t.Fatal(err)
				}
				if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			h.must(m.Close())
			f.close()
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !matched {
			t.Fatal("no actual caller comparison", iteration)
		}
	}
}
