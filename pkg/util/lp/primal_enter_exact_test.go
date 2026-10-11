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

func TestLPExactEnterSourceAndChildPrefixes(t *testing.T) {
	for _, phase := range []string{"negative", "column-range", "missing-factor", "empty-column", "types", "rows", "late-list", "stopped", "breakpoints"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
			en := int32(1)
			want := ErrBounds
			local := true
			f.types[0] = ColumnLowBound
			h.must(f.x.Data[0].Copy(h.m, h.pair("1", "0")))
			h.must(f.b.Data[0].Copy(h.m, h.pair("2", "0")))
			switch phase {
			case "negative":
				en = -1
				local = false
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "column-range":
				en = 99
				local = false
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "missing-factor":
				h.must(m.Factorization.Close())
				m.Factorization = nil
				local = false
			case "empty-column":
				p, e := f.a.ColumnValue(f.a.Columns[1][0])
				h.must(e)
				h.must(h.m.SetInt32(p, 0))
			case "types":
				f.types = nil
			case "rows":
				m.RowsNZ = nil
			case "late-list":
				m.NonBasisList = nil
				if sourceDebugEnabled {
					want = ErrBounds
				}
			case "stopped":
				m.stopped = true
				local = false
				want = ErrExactVectorPending
			case "breakpoints":
				h.must(f.x.Data[0].Copy(h.m, h.pair("-1", "0")))
				h.must(f.b.Data[0].Copy(h.m, h.pair("-2", "0")))
				h.must(m.InitInfSet())
				h.must(m.InitInfeasibilityCosts())
				h.must(m.InitReducedCostsForOneIteration())
				h.must(m.BreakpointQueue.Resize(4))
				m.SignOfEnteringDelta = -1
				m.LookForFeasibleSolutionOnly = true
				m.Settings.UseBreakpointsInFeasibilitySearch = true
				want = nil
				local = false
				if sourceDebugEnabled {
					want = ErrAssertion
					local = true
				}
			}
			if e := m.AdvanceOnEnteringPrecise(en); e != want {
				t.Fatal(phase, e, want)
			}
			if (m.enteringStep != nil) != local {
				t.Fatal("step construction/destruction prefix", phase, local)
			}
			if phase == "breakpoints" && !sourceDebugEnabled {
				h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
				h.check(&f.x.Data[1], big.NewRat(-14, 3), new(big.Rat))
				if m.Status != StatusFeasible || m.CurrentXIsInfeasible() {
					t.Fatal("breakpoint transition")
				}
			}
			if want != nil {
				if e := m.AdvanceOnEntering(en); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactEnterPreservesOlderComparison(t *testing.T) {
	for _, iteration := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
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
					err = m.OneIteration()
				} else {
					err = m.AdvanceOnEnteringPrecise(1)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) {
					t.Fatal(err)
				}
				if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
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

func TestLPExactEnterIndependentModels(t *testing.T) {
	seen := map[string]int{}
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		n := 1 + world%4
		en := uint32(n)
		f := lpNewPairCoreFixture(t, h, uint32(n), en+1)
		f.settings.SimplexStrategy = StrategyLU
		direction := int64(1)
		if world%2 == 0 {
			direction = -1
		}
		ds := make([]*big.Rat, n)
		bestX, bestY := (*big.Rat)(nil), (*big.Rat)(nil)
		leaving := -1
		for i := 0; i < n; i++ {
			ds[i] = big.NewRat(int64(1+(world+i)%7), int64(3+i))
			if (world+i)%3 == 0 {
				ds[i].Neg(ds[i])
			}
			diagonal := big.NewRat(int64(2+i), 1)
			h.must(f.a.Set(uint32(i), uint32(i), &h.pair(diagonal.RatString(), "0").X))
			coeff := new(big.Rat).Mul(diagonal, ds[i])
			h.must(f.a.Set(uint32(i), en, &h.pair(coeff.RatString(), "0").X))
			kind := ColumnType((world/3 + i) % 5)
			if world%11 == 0 {
				kind = ColumnFree
			}
			f.types[i] = kind
			lx, ly := big.NewRat(int64(1+i+world%9), 7), big.NewRat(int64((world+i)%3), 11)
			if kind == ColumnFixed {
				lx.SetInt64(0)
				ly.SetInt64(0)
			}
			abs := new(big.Rat).Abs(ds[i])
			bx, by := new(big.Rat).Mul(abs, lx), new(big.Rat).Mul(abs, ly)
			h.must(f.up.Data[i].Copy(h.m, h.pair(bx.RatString(), by.RatString())))
			h.must(f.low.Data[i].Copy(h.m, h.pair(new(big.Rat).Neg(bx).RatString(), new(big.Rat).Neg(by).RatString())))
			rising := ds[i].Sign()*int(direction) < 0
			eligible := kind == ColumnBoxed || kind == ColumnFixed || (kind == ColumnLowBound && !rising) || (kind == ColumnUpperBound && rising)
			if eligible {
				cmp := -1
				if bestX != nil {
					cmp = lx.Cmp(bestX)
					if cmp == 0 {
						cmp = ly.Cmp(bestY)
					}
				}
				if cmp < 0 {
					bestX, bestY = new(big.Rat).Set(lx), new(big.Rat).Set(ly)
					leaving = i
				}
				// Equal limits select the smaller supplied row-count estimate, i+1.
			}
		}
		f.types[en] = ColumnFree
		if world%5 == 0 {
			f.types[en] = ColumnBoxed
			h.must(f.low.Data[en].Copy(h.m, h.pair("-1/100", "-1/1000")))
			h.must(f.up.Data[en].Copy(h.m, h.pair("1/100", "1/1000")))
			bx, by := big.NewRat(1, 100), big.NewRat(1, 1000)
			if bestX == nil || bx.Cmp(bestX) < 0 {
				bestX, bestY = bx, by
				leaving = int(en)
			}
		}
		h.must(h.m.SetInt32(&f.costs.Data[en], int32(-direction)))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, en+1)
		m.RowsNZ = make([]uint32, n)
		for i := range m.RowsNZ {
			m.RowsNZ[i] = uint32(i + 1)
		}
		h.must(m.InitReducedCostsForOneIteration())
		m.NonBasisList = []uint32{en}
		m.SignOfEnteringDelta = int32(direction)
		m.LookForFeasibleSolutionOnly = world/2%2 == 0
		m.IterationsNoCostGrowth = 53
		m.Status = StatusOptimal
		seed := uint32(world + 97)
		m.Settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		rng.Next()
		if leaving >= 0 && leaving != int(en) {
			rng.Next()
		}
		if world%2 == 0 {
			h.must(m.AdvanceOnEnteringPrecise(int32(en)))
		} else {
			h.must(m.AdvanceOnEntering(int32(en)))
		}
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
				h.check(&f.x.Data[j], new(big.Rat), new(big.Rat))
			}
		} else {
			sx, sy := new(big.Rat).Mul(bestX, big.NewRat(direction, 1)), new(big.Rat).Mul(bestY, big.NewRat(direction, 1))
			for i := range ds {
				h.check(&f.x.Data[i], new(big.Rat).Neg(new(big.Rat).Mul(ds[i], sx)), new(big.Rat).Neg(new(big.Rat).Mul(ds[i], sy)))
			}
			h.check(&f.x.Data[en], sx, sy)
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
				if m.Status != StatusFeasible || f.basis[leaving] != en {
					t.Fatal("pivot metadata", world)
				}
				if bestX.Sign() == 0 && bestY.Sign() == 0 {
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
		for i := range ds {
			lpSparseRationalCheck(t, h, &m.Ed.Data[i], ds[i])
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

func TestLPExactEnterPhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, phase := range []string{"pivot", "flip", "unbounded", "iteration-pivot", "iteration-flip", "iteration-unbounded"} {
			success, borrowedFailure := false, false
			for ordinal := int64(0); ordinal < 1600; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%s/%d", direct, phase, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpAdvanceDifferentFixture(t, h)
					if strings.HasSuffix(phase, "pivot") {
						f.types[0] = ColumnLowBound
						h.must(f.x.Data[0].Copy(h.m, h.pair(strings.Repeat("1234567890", 12)+"1/13", "1/17")))
						h.must(f.b.Data[0].Copy(h.m, &f.x.Data[0]))
						h.must(f.b.Data[0].Scale(h.m, &f.b.Data[0], &h.pair("2", "0").X))
					}
					if strings.HasSuffix(phase, "flip") {
						f.types[1] = ColumnBoxed
						h.must(f.up.Data[1].Copy(h.m, h.pair(strings.Repeat("9876543210", 12)+"1/19", "1/23")))
						h.must(f.low.Data[1].Copy(h.m, h.pair("-1", "0")))
					}
					h.must(h.m.SetString(&m.D.Data[1], strings.Repeat("3141592653", 12)+"1/29"))
					h.budget.FailAfter(ordinal)
					var err error
					if strings.HasPrefix(phase, "iteration-") {
						m.ColumnsNZ = []uint32{1, 1, 1}
						err = m.OneIteration()
					} else {
						err = m.AdvanceOnEntering(1)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						if m.enteringStep != nil {
							t.Fatal("success local")
						}
					} else {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.enteringStep != nil && (m.calls.pending() || m.PairCoreSolverBase.calls.pending()) {
							borrowedFailure = true
						}
						if e := m.AdvanceOnEnteringPrecise(1); e != ErrExactVectorPending {
							t.Fatal("retry", e)
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
			if !success || (direct && !strings.HasSuffix(phase, "unbounded") && !borrowedFailure) {
				t.Fatal("campaign", direct, phase, borrowedFailure)
			}
		}
	}
}
