// Source: src/util/lp/lp_dual_core_solver.h/.hpp; concrete mpq/mpq.
package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPRationalDualCoreIndependentModels(t *testing.T) {
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 3, 6)
		f.settings.SimplexStrategy = StrategyLU
		f.settings.SimplexStrategy = StrategyLU
		f.basis = []uint32{2, 0, 4}
		diag := []*big.Rat{big.NewRat(2, 3), big.NewRat(-4, 7), big.NewRat(8, 11)}
		y := []*big.Rat{new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(world+1)), uint(81+world%31)), big.NewInt(13)), big.NewRat(int64(world-70), 17), big.NewRat(-3, 19)}
		for i, j := range f.basis {
			h.must(f.a.Set(uint32(i), j, lpScalarBoundValue(h, diag[i].RatString())))
			cost := new(big.Rat).Mul(y[i], diag[i])
			h.must(h.m.Copy(&f.costs.Data[j], lpScalarBoundValue(h, cost.RatString())))
			h.must(h.m.SetInt32(&f.b.Data[i], int32(i+world+1)))
		}
		for _, j := range []uint32{1, 3, 5} {
			for i := 0; i < 3; i++ {
				h.must(f.a.Set(uint32(i), j, lpScalarBoundValue(h, big.NewRat(int64(i+int(j)+1), 23).RatString())))
			}
			h.must(h.m.SetInt32(&f.costs.Data[j], int32(j)))
			f.types[j] = ColumnBoxed
			h.must(h.m.SetInt32(&f.low.Data[j], -2))
			h.must(h.m.SetInt32(&f.up.Data[j], 3))
			h.must(h.m.SetInt32(&f.x.Data[j], 7))
		}
		can := []bool{false, world%2 == 0, false, true, false, false}
		m, err := NewRationalDualCoreSolver(f.input(), &can)
		h.must(err)
		for i := range y {
			lpSparseRationalCheck(t, h, &m.Y.Data[i], y[i])
		}
		lpSparseRationalCheck(t, h, &m.HarrisTolerance, new(big.Rat))
		h.must(m.RecalculateD())
		d := map[uint32]*big.Rat{}
		for _, j := range []uint32{1, 3, 5} {
			want := big.NewRat(int64(j), 1)
			for i := 0; i < 3; i++ {
				want.Sub(want, new(big.Rat).Mul(y[i], big.NewRat(int64(i+int(j)+1), 23)))
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			d[j] = want
		}
		expected := []uint32{3}
		if world%2 == 0 {
			expected = []uint32{3, 1}
		}
		if !reflect.DeepEqual(f.nb, expected) {
			t.Fatal(world, f.nb, expected)
		}
		h.must(m.StartWithInitialBasisAndMakeItDualFeasible())
		for i, j := range f.basis {
			want := big.NewRat(int64(i+world+1), 1)
			for _, k := range []uint32{1, 3, 5} {
				x := int64(7)
				if can[k] {
					x = -2
					if d[k].Sign() < 0 {
						x = 3
					}
				}
				lpSparseRationalCheck(t, h, &f.x.Data[k], big.NewRat(x, 1))
				want.Sub(want, new(big.Rat).Mul(big.NewRat(x, 1), big.NewRat(int64(i+int(k)+1), 23)))
			}
			want.Quo(want, diag[i])
			lpSparseRationalCheck(t, h, &f.x.Data[j], want)
			lpSparseRationalCheck(t, h, &m.Betas.Data[i], big.NewRat(1, 1))
		}
		h.must(m.RecalculateXBAndD())
		can = []bool{false, false, false, false, false, true}
		old := f.heading[1]
		h.must(m.RestoreNonBasis())
		if !reflect.DeepEqual(f.nb, []uint32{5}) || f.heading[5] != -1 || f.heading[1] != old {
			t.Fatal("excluded heading changed", f.nb, f.heading)
		}
		h.must(m.InitAWaveByZeros())
		for i := range m.AWave.Data {
			lpSparseRationalCheck(t, h, &m.AWave.Data[i], new(big.Rat))
		}
		h.must(m.Close())
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualCoreDoneOrder(t *testing.T) {
	for _, status := range []Status{StatusOptimal, StatusUnknown, StatusCancelled, StatusFloatingPointError, StatusDualUnbounded} {
		for _, count := range []uint32{0, 3, 4} {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			can := []bool{false, true}
			m, e := NewRationalDualCoreSolver(f.input(), &can)
			h.must(e)
			m.Status = status
			m.TotalIterations = count
			f.settings.MaxTotalIterations = 3
			done, e := m.Done()
			h.must(e)
			want := status == StatusOptimal || count > 3
			end := status
			if status != StatusOptimal && count > 3 {
				end = StatusIterationsExhausted
			}
			if done != want || m.Status != end {
				t.Fatal(status, count, done, m.Status)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalDualCorePartialOwners(t *testing.T) {
	for _, phase := range []string{"nil-flags", "nil-matrix", "short-flags", "tableau", "short-costs", "base-stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			can := []bool{false, true, true}
			flags := &can
			in := f.input()
			saved := f.costs.Data
			switch phase {
			case "nil-flags":
				flags = nil
			case "nil-matrix":
				in.A = nil
			case "short-flags":
				can = can[:2]
			case "tableau":
				f.settings.SimplexStrategy = StrategyTableauRows
			case "short-costs":
				f.costs.Data = nil
			case "base-stopped":
				h.budget.FailAfter(0)
			}
			m, e := NewRationalDualCoreSolver(in, flags)
			h.budget.FailAfter(-1)
			f.costs.Data = saved
			if e == nil && phase != "base-stopped" {
				t.Fatal("missing rejection")
			}
			if m != nil {
				h.must(m.Close())
			}
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualCorePhysicalOwnership(t *testing.T) {
	for _, op := range []string{"construct", "recalculate", "start", "adjust", "place", "feasible"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 3)
					f.settings.SimplexStrategy = StrategyLU
					huge := strings.Repeat("1234567890", 12)
					h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "2/3")))
					h.must(f.a.Set(0, 1, lpScalarBoundValue(h, huge+"7/13")))
					h.must(h.m.SetString(&f.costs.Data[0], huge+"3/17"))
					h.must(h.m.SetString(&f.b.Data[0], huge+"5/19"))
					f.types[1] = ColumnBoxed
					h.must(h.m.SetInt32(&f.low.Data[1], -2))
					h.must(h.m.SetString(&f.up.Data[1], huge+"1/23"))
					can := []bool{false, true, false}
					var m *RationalDualCoreSolver
					var e error
					if op == "construct" {
						h.budget.FailAfter(ordinal)
						m, e = NewRationalDualCoreSolver(f.input(), &can)
					} else {
						m, e = NewRationalDualCoreSolver(f.input(), &can)
						h.must(e)
						h.must(h.m.SetInt32(&m.D.Data[1], -1))
						if op == "feasible" {
							h.must(h.m.Copy(&f.x.Data[1], &f.up.Data[1]))
						}
						h.budget.FailAfter(ordinal)
						switch op {
						case "recalculate":
							e = m.RecalculateXBAndD()
						case "start":
							e = m.StartWithInitialBasisAndMakeItDualFeasible()
						case "adjust":
							e = m.AdjustXBForChangedXNAndInitBetas()
						case "place":
							e = m.SetNonBasicXToCorrectBounds()
						case "feasible":
							var ok bool
							ok, e = m.ColumnIsDualFeasible(1)
							if e == nil && !ok {
								t.Fatal("feasible boxed upper bound rejected")
							}
						}
					}
					h.budget.FailAfter(-1)
					if e == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
					}
					if m != nil {
						h.must(m.Close())
					}
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || direct && failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalDualCoreLiveMetadataPrefixes(t *testing.T) {
	for _, phase := range []string{"flags", "heading", "wave", "betas", "costs", "rhs", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 4)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(1, 1, lpScalarBoundValue(h, "1")))
			can := []bool{false, false, true, true}
			m, e := NewRationalDualCoreSolver(f.input(), &can)
			h.must(e)
			heading, costs, rhs := f.heading, f.costs.Data, f.b.Data
			wave, betas := m.AWave.Data, m.Betas.Data
			switch phase {
			case "flags":
				can = nil
				e = m.RestoreNonBasis()
				if len(f.nb) != 0 {
					t.Fatal(f.nb)
				}
			case "heading":
				f.heading = nil
				e = m.RestoreNonBasis()
			case "wave":
				m.AWave.Data = m.AWave.Data[:1]
				e = m.InitAWaveByZeros()
			case "betas":
				m.Betas.Data = m.Betas.Data[:1]
				e = m.InitBetas()
			case "costs":
				f.costs.Data = nil
				e = m.RecalculateD()
			case "rhs":
				f.b.Data = nil
				e = m.RecalculateXBAndD()
			case "stopped":
				m.stopped = true
				_, e = m.Done()
			}
			if e == nil {
				t.Fatal("missing rejection")
			}
			if retry := m.InitBetas(); retry != ErrExactVectorPending {
				t.Fatal("stopped retry", retry)
			}
			f.heading, f.costs.Data, f.b.Data = heading, costs, rhs
			m.AWave.Data, m.Betas.Data = wave, betas
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualCorePreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"construct", "start", "recalculate"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "2/3")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "5/7")))
			can := []bool{false, true}
			var m *RationalDualCoreSolver
			var e error
			if op != "construct" {
				m, e = NewRationalDualCoreSolver(f.input(), &can)
				h.must(e)
			}
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, before, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				switch op {
				case "construct":
					m, e = NewRationalDualCoreSolver(f.input(), &can)
				case "start":
					e = m.StartWithInitialBasisAndMakeItDualFeasible()
				case "recalculate":
					e = m.RecalculateXBAndD()
				}
				if m != nil && (m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0) {
					t.Fatal("adopted caller comparison")
				}
			} else if e != nil && !lpExactPriorityPhysicalRefusal(e) {
				t.Fatal(e)
			}
			if m != nil {
				h.must(m.Close())
			}
			f.close()
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !matched {
			t.Fatal("no actual older comparison", op)
		}
	}
}
