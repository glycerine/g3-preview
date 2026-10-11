// Source: src/util/lp/lp_primal_core_solver.h:446-449,483-511.
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

func lpRowIterationFixture(t *testing.T, h *lpPairHarness, initial, bound, a *big.Rat, epsilonInitial, epsilonBound *big.Rat, kind ColumnType) (*lpPairCoreFixture, *PairPrimalCoreSolver) {
	f := lpNewPairCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = StrategyTableauRows
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.a.Set(0, 1, &h.pair(a.RatString(), "0").X))
	h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
	f.types[0], f.types[1], f.types[2] = kind, ColumnFree, ColumnFixed
	h.must(f.x.Data[0].Copy(h.m, h.pair(initial.RatString(), epsilonInitial.RatString())))
	for _, p := range []*RationalPair{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
		h.must(p.Copy(h.m, h.pair(initial.RatString(), epsilonInitial.RatString())))
	}
	for _, p := range []*RationalPair{&f.low.Data[0], &f.up.Data[0]} {
		h.must(p.Copy(h.m, h.pair(bound.RatString(), epsilonBound.RatString())))
	}
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	m.Names = lpFeasibleNames{}
	h.must(m.InitInfSet())
	h.must(m.InitRunTableau())
	return f, m
}

func TestLPExactRowIterationIndependentFeasibleModels(t *testing.T) {
	kinds := []ColumnType{ColumnFixed, ColumnLowBound, ColumnUpperBound, ColumnBoxed}
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		kind := kinds[world%4]
		bound := big.NewRat(int64(world%17-8), 7)
		by := big.NewRat(int64(world%7-3), 11)
		gap := big.NewRat(int64(1+world%11), 13)
		gy := big.NewRat(int64(1+world%5), 17)
		above := world/4%2 == 0
		if kind == ColumnLowBound {
			above = false
		}
		if kind == ColumnUpperBound {
			above = true
		}
		initial, iy := new(big.Rat).Sub(bound, gap), new(big.Rat).Sub(by, gy)
		if above {
			initial.Add(bound, gap)
			iy.Add(by, gy)
		}
		a := big.NewRat(int64(1+world%9), int64(3+world%7))
		if world/8%2 == 0 {
			a.Neg(a)
		}
		f, m := lpRowIterationFixture(t, h, initial, bound, a, iy, by, kind)
		m.Status = StatusUnknown
		m.TotalIterations = 73
		m.IterationsNoCostGrowth = 37
		m.BasisSortCounter = 19
		m.SignOfEnteringDelta = -91
		m.NonBasisList = []uint32{2, 1}
		m.BlandModeTableau = world/16%2 == 0
		oldBland := m.BlandModeTableau
		h.must(m.OneIterationTableauRows())
		theta := new(big.Rat).Quo(new(big.Rat).Sub(initial, bound), a)
		ty := new(big.Rat).Quo(new(big.Rat).Sub(iy, by), a)
		h.check(&f.x.Data[0], bound, by)
		h.check(&f.x.Data[1], theta, ty)
		h.check(&f.x.Data[2], initial, iy)
		if m.Status != StatusOptimal || f.basis[0] != 1 || m.InfSet.Size() != 0 {
			t.Fatal("iteration", world, m.Status, f.basis, m.InfSet.Indices())
		}
		if m.TotalIterations != 73 || m.IterationsNoCostGrowth != 37 || m.BasisSortCounter != 19 || m.SignOfEnteringDelta != -91 || m.NonBasisList[1] != 1 {
			t.Fatal("unrelated loop state", world)
		}
		if m.LeftBasisTableau.Contains(0) == oldBland || m.LeftBasisRepeated != 0 {
			t.Fatal("source repetition tracking", world)
		}
		if m.rowIterationBound != nil || m.rowIterationCoefficient != nil || m.rowIterationTheta != nil || m.pairWork != nil || m.Factorization != nil {
			t.Fatal("retained local/LU")
		}
		for j := range m.D.Data {
			lpSparseRationalCheck(t, h, &m.D.Data[j], new(big.Rat))
		}
		h.must(m.OneIterationTableauRows())
		h.check(&f.x.Data[1], theta, ty)
		if m.Status != StatusOptimal || m.TotalIterations != 73 {
			t.Fatal("terminal iteration")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactRowIterationTwoStepInfeasibility(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		// x0-e+c=0, x0>=0, e in [0,u], c>u proves infeasibility.
		c := big.NewRat(int64(2+world%13), 7)
		u := new(big.Rat).Quo(c, big.NewRat(2, 1))
		f, m := lpRowIterationFixture(t, h, new(big.Rat).Neg(c), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
		f.types[1] = ColumnBoxed
		h.must(f.up.Data[1].Copy(h.m, h.pair(u.RatString(), "0")))
		m.Status = StatusUnknown
		h.must(m.OneIterationTableauRows())
		h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
		h.check(&f.x.Data[1], c, new(big.Rat))
		if m.Status != StatusUnknown || f.basis[0] != 1 || m.InfSet.Size() != 1 || !m.InfSet.Contains(1) {
			t.Fatal("post-pivot entering membership", world)
		}
		h.must(m.OneIterationTableauRows())
		if m.Status != StatusInfeasible || m.InfRowIndexForTableau != 0 || !m.LeftBasisTableau.Contains(0) || !m.LeftBasisTableau.Contains(1) {
			t.Fatal("infeasibility", world)
		}
		h.check(&f.x.Data[1], c, new(big.Rat))
		if m.rowIterationBound != nil || m.rowIterationCoefficient != nil || m.rowIterationTheta != nil {
			t.Fatal("early-return local")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactRowIterationRepetitionThreshold(t *testing.T) {
	for _, profile := range []string{"first", "equal", "exceed", "wrap", "already-bland"} {
		t.Run(profile, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRowIterationFixture(t, h, big.NewRat(-1, 1), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
			m.BlandModeThreshold = 3
			m.LeftBasisRepeated = 2
			if profile != "first" {
				h.must(m.LeftBasisTableau.Insert(0))
			}
			wantCount := uint32(2)
			wantBland := false
			switch profile {
			case "equal":
				wantCount = 3
			case "exceed":
				m.LeftBasisRepeated = 3
				wantCount = 4
				wantBland = true
			case "wrap":
				m.LeftBasisRepeated = ^uint32(0)
				wantCount = 0
			case "already-bland":
				m.BlandModeTableau = true
				wantBland = true
			}
			h.must(m.OneIterationTableauRows())
			if m.LeftBasisRepeated != wantCount || m.BlandModeTableau != wantBland || !m.LeftBasisTableau.Contains(0) {
				t.Fatal("repetition", m.LeftBasisRepeated, m.BlandModeTableau)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactRowIterationPhysicalOwnership(t *testing.T) {
	for _, profile := range []string{"step", "bound", "entering-bound", "coefficient", "minneg"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 600; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", profile, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					huge := strings.Repeat("1234567890", 100)
					initial := new(big.Rat).Neg(mustLPRat(t, huge+"1/11"))
					bound := new(big.Rat)
					a := big.NewRat(-2, 7)
					iy := new(big.Rat).Neg(mustLPRat(t, huge+"3/13"))
					by := new(big.Rat)
					switch profile {
					case "bound":
						initial.SetInt64(0)
						iy.SetInt64(0)
						bound.Set(mustLPRat(t, huge+"5/7"))
						by.Set(mustLPRat(t, huge+"7/13"))
					case "coefficient":
						initial.SetInt64(-1)
						iy.SetInt64(0)
						a.Neg(mustLPRat(t, huge+"1/7"))
					case "minneg":
						initial.SetInt64(-2147483648)
						iy.SetInt64(0)
						bound.SetInt64(1)
						a.SetInt64(-1)
					}
					f, m := lpRowIterationFixture(t, h, initial, bound, a, iy, by, ColumnLowBound)
					if profile == "minneg" {
						for _, p := range []*RationalPair{&f.x.Data[0], &f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
							h.must(h.m.SetInt32(&p.X, -2147483648))
						}
					}
					if profile == "entering-bound" {
						f.types[1] = ColumnBoxed
						h.must(f.up.Data[1].Copy(h.m, h.pair("1", "0")))
					}
					m.TotalIterations = 73
					m.IterationsNoCostGrowth = 37
					h.budget.FailAfter(ordinal)
					err := m.OneIterationTableauRows()
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						theta := new(big.Rat).Quo(new(big.Rat).Sub(initial, bound), a)
						ty := new(big.Rat).Quo(new(big.Rat).Sub(iy, by), a)
						h.check(&f.x.Data[0], bound, by)
						h.check(&f.x.Data[1], theta, ty)
						status := StatusOptimal
						if profile == "entering-bound" {
							status = StatusUnknown
							if !m.InfSet.Contains(1) {
								t.Fatal("missing entering infeasibility")
							}
						}
						if m.Status != status || f.basis[0] != 1 || m.rowIterationTheta != nil || m.rowIterationCoefficient != nil || m.rowIterationBound != nil {
							t.Fatal("success state", m.Status, status)
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.rowIterationBound == nil {
							t.Fatal("lost automatic local")
						}
						if retry := m.OneIterationTableauRows(); retry != ErrExactVectorPending {
							t.Fatal("retry", retry)
						}
						if m.calls.pending() && m.PairCoreSolverBase.calls.pending() {
							t.Fatal("adopted child numeric receipt")
						}
					}
					if m.TotalIterations != 73 || m.IterationsNoCostGrowth != 37 {
						t.Fatal("outer-loop counters changed")
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
			if !success || (direct && failures == 0) {
				t.Fatal("campaign", profile, direct, failures)
			}
		}
	}
}

func TestLPExactRowIterationReachedPrefixes(t *testing.T) {
	for _, profile := range []string{"stopped", "pending-bound", "pending-coefficient", "pending-theta", "leaving", "left-domain", "heading", "selector", "no-entering", "row-cost-assert"} {
		t.Run(profile, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRowIterationFixture(t, h, big.NewRat(-1, 1), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
			savedHeading := append([]int32(nil), f.heading...)
			savedTypes := append([]ColumnType(nil), f.types...)
			m.Status = StatusUnknown
			m.InfRowIndexForTableau = 91
			want := error(nil)
			wantBound, wantCoeff, wantTheta := true, false, false
			wantHistory := false
			switch profile {
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
				wantBound = false
			case "pending-bound":
				m.rowIterationBound = NewIndexedPairVector(h.m, 1)
				want = ErrExactVectorPending
			case "pending-coefficient":
				m.rowIterationCoefficient = NewIndexedRationalVector(h.m, 1)
				want = ErrExactVectorPending
				wantBound = false
				wantCoeff = true
			case "pending-theta":
				m.rowIterationTheta = NewIndexedPairVector(h.m, 1)
				want = ErrExactVectorPending
				wantBound = false
				wantTheta = true
			case "leaving":
				f.types = nil
				want = ErrBounds
			case "left-domain":
				h.must(m.LeftBasisTableau.Resize(0))
				want = ErrBounds
			case "heading":
				f.heading = nil
				want = ErrBounds
				wantHistory = true
				wantCoeff = true
			case "selector":
				f.heading[0] = 9
				want = ErrBounds
				wantHistory = true
				wantCoeff = true
			case "no-entering":
				f.types[1] = ColumnFixed
				wantBound = false
				wantHistory = true
			case "row-cost-assert":
				// Rows mode deliberately requires ordinary membership updates, rather
				// than the cost-update path asserted out by UpdateInfCostForColumnTableau.
				m.UsingInfeasCosts = true
				wantHistory = true
				if sourceDebugEnabled {
					want = ErrAssertion
					wantCoeff = true
					wantTheta = true
				} else {
					wantBound = false
				}
			}
			err := m.OneIterationTableauRows()
			f.heading, f.types = savedHeading, savedTypes
			if err != want {
				t.Fatal(err, want)
			}
			if (m.rowIterationBound != nil) != wantBound || (m.rowIterationCoefficient != nil) != wantCoeff || (m.rowIterationTheta != nil) != wantTheta {
				t.Fatal("automatic lifetime prefix", m.rowIterationBound != nil, m.rowIterationCoefficient != nil, m.rowIterationTheta != nil)
			}
			if m.LeftBasisTableau.Contains(0) != wantHistory {
				t.Fatal("history publication")
			}
			if profile == "no-entering" {
				if m.Status != StatusInfeasible || m.InfRowIndexForTableau != 0 {
					t.Fatal("infeasible status")
				}
			} else if err != nil && m.Status != StatusUnknown {
				t.Fatal("status published before error")
			}
			if err != nil {
				if retry := m.OneIterationTableauRows(); retry != ErrExactVectorPending {
					t.Fatal("retry", retry)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactRowAdvanceReachedPrefixes(t *testing.T) {
	for _, profile := range []string{"stopped", "nil", "entering", "heading", "inf-domain", "success"} {
		t.Run(profile, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRowIterationFixture(t, h, big.NewRat(-1, 1), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
			theta := h.pair("1", "0")
			entering := int32(1)
			want := error(nil)
			savedHeading := f.heading
			switch profile {
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
			case "nil":
				theta = nil
				want = ErrBounds
			case "entering":
				entering = 9
				want = ErrBounds
			case "heading":
				f.heading = nil
				want = ErrBounds
			case "inf-domain":
				f.types[1] = ColumnBoxed
				h.must(f.up.Data[1].Copy(h.m, h.pair("1/2", "0")))
				h.must(m.InfSet.Resize(1))
				want = ErrBounds
			}
			err := m.AdvanceOnEnteringAndLeavingTableauRows(entering, 0, theta)
			f.heading = savedHeading
			if err != want {
				t.Fatal(err, want)
			}
			if profile == "heading" {
				h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
				if f.basis[0] != 0 || m.InfSet.Contains(0) {
					t.Fatal("update before heading guard")
				}
			}
			if profile == "inf-domain" {
				if f.basis[0] != 1 {
					t.Fatal("entering membership checked before basis publication")
				}
			}
			if profile == "success" {
				if f.basis[0] != 1 || m.InfSet.Size() != 0 {
					t.Fatal("success")
				}
				h.check(theta, big.NewRat(1, 1), new(big.Rat))
			}
			if err != nil {
				if retry := m.AdvanceOnEnteringAndLeavingTableauRows(1, 0, theta); retry != ErrExactVectorPending {
					t.Fatal("retry", retry)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactRowIterationPreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"iteration", "advance"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			huge := mustLPRat(t, strings.Repeat("1234567890", 100)+"1/11")
			f, m := lpRowIterationFixture(t, h, new(big.Rat).Neg(huge), new(big.Rat), big.NewRat(-1, 1), new(big.Rat), new(big.Rat), ColumnLowBound)
			theta := h.pair(huge.RatString(), "0")
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
				if op == "iteration" {
					err = m.OneIterationTableauRows()
				} else {
					err = m.AdvanceOnEnteringAndLeavingTableauRows(1, 0, theta)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) {
					t.Fatal(op, err)
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
			t.Fatal("no actual older comparison", op)
		}
	}
}

func TestLPExactRowIterationPermutedBasisAndMinimumLeaving(t *testing.T) {
	for world := 0; world < 80; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 5)
		f.settings.SimplexStrategy = StrategyTableauRows
		f.basis = []uint32{3, 1}
		for j := range f.types {
			f.types[j] = ColumnFree
		}
		f.types[1], f.types[3], f.types[4] = ColumnLowBound, ColumnLowBound, ColumnFixed
		d, c := big.NewRat(int64(1+world%17), 7), big.NewRat(int64(1+world%11), 5)
		h.must(f.a.Set(0, 3, &h.pair("1", "0").X))
		h.must(f.a.Set(1, 1, &h.pair("1", "0").X))
		h.must(f.a.Set(0, 0, &h.pair("-2", "0").X))
		h.must(f.a.Set(1, 2, &h.pair("-3", "0").X))
		h.must(f.a.Set(0, 4, &h.pair(d.RatString(), "0").X))
		h.must(f.a.Set(1, 4, &h.pair(c.RatString(), "0").X))
		for _, p := range []*RationalPair{&f.x.Data[4], &f.low.Data[4], &f.up.Data[4]} {
			h.must(p.Copy(h.m, h.pair("1", "1/13")))
		}
		h.must(f.x.Data[3].Copy(h.m, h.pair(new(big.Rat).Neg(d).RatString(), new(big.Rat).Quo(new(big.Rat).Neg(d), big.NewRat(13, 1)).RatString())))
		h.must(f.x.Data[1].Copy(h.m, h.pair(new(big.Rat).Neg(c).RatString(), new(big.Rat).Quo(new(big.Rat).Neg(c), big.NewRat(13, 1)).RatString())))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitInfSet())
		h.must(m.InfSet.Clear())
		h.must(m.InfSet.Insert(3))
		h.must(m.InfSet.Insert(1))
		h.must(m.InitRunTableau())
		m.Status = StatusUnknown
		h.must(m.OneIterationTableauRows())
		if f.basis[0] != 3 || f.basis[1] != 2 || m.Status != StatusUnknown || m.InfSet.Size() != 1 || !m.InfSet.Contains(3) {
			t.Fatal("minimum leaving and actual heading", world)
		}
		h.check(&f.x.Data[1], new(big.Rat), new(big.Rat))
		h.check(&f.x.Data[2], new(big.Rat).Quo(c, big.NewRat(3, 1)), new(big.Rat).Quo(c, big.NewRat(39, 1)))
		h.must(m.OneIterationTableauRows())
		if f.basis[0] != 0 || f.basis[1] != 2 || m.Status != StatusOptimal || m.InfSet.Size() != 0 {
			t.Fatal("second row", world)
		}
		h.check(&f.x.Data[0], new(big.Rat).Quo(d, big.NewRat(2, 1)), new(big.Rat).Quo(d, big.NewRat(26, 1)))
		h.check(&f.x.Data[3], new(big.Rat), new(big.Rat))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactRowIterationThresholdAffectsCurrentSelection(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 1, 4)
	f.settings.SimplexStrategy = StrategyTableauRows
	f.types[0], f.types[1], f.types[2], f.types[3] = ColumnLowBound, ColumnFree, ColumnFree, ColumnFixed
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
	h.must(f.a.Set(0, 1, &h.pair("-2", "0").X))
	h.must(f.a.Set(0, 3, &h.pair("-1", "0").X))
	h.must(f.x.Data[0].Copy(h.m, h.pair("-1", "0")))
	for _, p := range []*RationalPair{&f.x.Data[3], &f.low.Data[3], &f.up.Data[3]} {
		h.must(p.Copy(h.m, h.pair("-1", "0")))
	}
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	h.must(m.InitInfSet())
	h.must(m.InitRunTableau())
	h.must(m.LeftBasisTableau.Insert(0))
	m.BlandModeThreshold = 0
	m.LeftBasisRepeated = 0
	m.Settings.RandomSeed(17)
	rng := z3rand.New(17)
	h.must(m.OneIterationTableauRows())
	if !m.BlandModeTableau || m.LeftBasisRepeated != 1 || f.basis[0] != 1 || m.Settings.RandomNext() != rng.Next() {
		t.Fatal("threshold did not affect current selector")
	}
	h.check(&f.x.Data[1], big.NewRat(1, 2), new(big.Rat))
	h.check(&f.x.Data[2], new(big.Rat), new(big.Rat))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
