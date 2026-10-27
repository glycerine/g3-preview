// Source: lp_primal_core_solver_tableau.h:323-346 and its called assertions.
package lp

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpInitTableauFixture(t *testing.T, h *lpPairHarness, infeasible bool) (*lpPairCoreFixture, *PairPrimalCoreSolver) {
	f := lpNewPairCoreFixture(t, h, 1, 2)
	f.settings.SimplexStrategy = StrategyTableauCosts
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.a.Set(0, 1, &h.pair("-1", "0").X))
	f.types[0] = ColumnLowBound
	f.types[1] = ColumnFree
	value := "2/7"
	if infeasible {
		value = "-2/7"
	}
	for j := range f.x.Data {
		h.must(f.x.Data[j].Copy(h.m, h.pair(value, "1/13")))
	}
	h.must(h.m.SetString(&f.costs.Data[1], "5/11"))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	m.Names = lpFeasibleNames{}
	h.must(m.InitInfSet())
	h.must(m.InitReducedCostsTableau())
	return f, m
}

func TestLPExactInitTableauIndependentStateModels(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		inf := world%2 == 0
		f, m := lpInitTableauFixture(t, h, inf)
		rows := world/2%2 == 0
		if rows {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		only := world/4%2 == 0
		m.LookForFeasibleSolutionOnly = only
		backup := world/8%2 == 0
		f.settings.BackupCosts = backup
		breakpoints := world/16%2 == 0
		f.settings.UseBreakpointsInFeasibilitySearch = breakpoints
		if inf {
			m.UsingInfeasCosts = false
			h.must(m.InitReducedCostsTableau())
		}
		h.must(m.CostsBackup.resizeData(1))
		h.must(h.m.SetInt32(&m.CostsBackup.Data[0], 17))
		h.must(m.BreakpointQueue.Resize(7))
		h.must(m.LeftBasisTableau.Resize(7))
		h.must(m.LeftBasisTableau.Insert(6))
		m.BlandModeTableau = true
		m.LeftBasisRepeated = 13
		m.BasisSortCounter = 19
		m.TotalIterations = 37
		m.IterationsNoCostGrowth = 41
		m.ColumnNormUpdateCounter = 43
		m.ColumnsNZ = []uint32{91}
		m.RowsNZ = []uint32{92}
		h.must(h.m.SetString(&m.EpsilonOfReducedCost, "3/17"))
		h.must(m.InitRunTableau())
		wantInfSize := uint32(0)
		if inf {
			wantInfSize = 1
		}
		early := !inf && only
		if m.BasisSortCounter != 0 || m.TotalIterations != 0 || m.IterationsNoCostGrowth != 0 {
			t.Fatal("counters", world)
		}
		if m.UsingInfeasCosts != inf || m.InfSet.Contains(0) != inf || m.InfSet.Size() != wantInfSize {
			t.Fatal("cost/membership recomputed", world)
		}
		epsilon := new(big.Rat)
		if early {
			epsilon = big.NewRat(3, 17)
		}
		lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, epsilon)
		size := 7
		if breakpoints && !early {
			size = 2
		}
		if len(m.BreakpointQueue.priorities.Data) != size {
			t.Fatal("conditional queue", world)
		}
		if backup && !only && !early {
			if len(m.CostsBackup.Data) != 2 {
				t.Fatal("backup extent")
			}
			for j := range f.costs.Data {
				want := big.NewRat(0, 1)
				if inf && j == 0 {
					want.SetInt64(1)
					if breakpoints {
						want.SetInt64(-1)
					}
				}
				if !inf && j == 1 {
					want.SetFrac64(5, 11)
				}
				lpSparseRationalCheck(t, h, &m.CostsBackup.Data[j], want)
			}
		} else {
			if len(m.CostsBackup.Data) != 1 {
				t.Fatal("early backup extent")
			}
			lpSparseRationalCheck(t, h, &m.CostsBackup.Data[0], big.NewRat(17, 1))
		}
		if rows && !early {
			if m.BlandModeTableau || m.LeftBasisRepeated != 0 || m.LeftBasisTableau.Size() != 0 || m.LeftBasisTableau.DataSize() != 2 {
				t.Fatal("row initialization", world)
			}
		} else if !m.BlandModeTableau || m.LeftBasisRepeated != 13 || !m.LeftBasisTableau.Contains(6) {
			t.Fatal("row state changed", world)
		}
		if m.Factorization != nil || m.ColumnNormUpdateCounter != 43 || m.ColumnsNZ[0] != 91 || m.RowsNZ[0] != 92 {
			t.Fatal("precise tableau initialized LU estimates")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactTableauBasisIndependentSingletonModels(t *testing.T) {
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 3)
		expectedShape := true
		for j := 0; j < 2; j++ {
			count := (world / (j + 1)) % 3
			if count != 1 {
				expectedShape = false
			}
			value := "1"
			if (world+j)%4 == 0 {
				value = "2/7"
			}
			for i := 0; i < count; i++ {
				h.must(f.a.Set(uint32(i), uint32(j), &h.pair(value, "0").X))
			}
		}
		// Source shape predicate visits all basis entries; pivot predicate can stop
		// on an earlier nonunit/multiple column before a later empty column.
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		got, e := m.BasisColumnsAreSetCorrectly()
		h.must(e)
		if got != expectedShape {
			t.Fatal("shape", world)
		}
		wantErr := error(nil)
		expectedPivot := false
		for _, j := range f.basis {
			count := (world / (int(j) + 1)) % 3
			if count == 0 {
				wantErr = ErrBounds
				if sourceDebugEnabled {
					wantErr = ErrAssertion
				}
				break
			}
			if count > 1 || (world+int(j))%4 == 0 {
				expectedPivot = true
				break
			}
		}
		pivot, e := m.NeedToPivotToBasisTableau()
		if e != wantErr || (e == nil && pivot != expectedPivot) {
			t.Fatal("pivot", world, pivot, e, expectedPivot, wantErr)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactTableauBasisRowIdentityIsNotChecked(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 2, 2)
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.a.Set(1, 1, &h.pair("1", "0").X))
	f.basis = []uint32{1, 0}
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	ok, e := m.BasisColumnsAreSetCorrectly()
	h.must(e)
	if !ok {
		t.Fatal("shape")
	}
	pivot, e := m.NeedToPivotToBasisTableau()
	h.must(e)
	if pivot {
		t.Fatal("source does not inspect cell row identity")
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactInitTableauReachedPrefixes(t *testing.T) {
	for _, profile := range []string{"stopped", "ax", "ax-metadata", "shape", "shape-index", "row-clear", "membership", "membership-name", "queue", "reduced-cost", "pivot"} {
		t.Run(profile, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpInitTableauFixture(t, h, false)
			m.BasisSortCounter = 19
			m.TotalIterations = 37
			m.IterationsNoCostGrowth = 41
			h.must(h.m.SetString(&m.EpsilonOfReducedCost, "3/17"))
			savedX := f.x.Data
			savedBasis := append([]uint32(nil), f.basis...)
			want := error(nil)
			earlyFailure := false
			switch profile {
			case "stopped":
				m.stopped = true
				want = ErrExactVectorPending
				earlyFailure = true
			case "ax-metadata":
				f.x.Data = nil
				if sourceDebugEnabled {
					want = ErrAssertion
					earlyFailure = true
				}
			case "shape-index":
				f.basis[0] = 9
				if sourceDebugEnabled {
					want = ErrBounds
					earlyFailure = true
				}
			case "row-clear":
				f.settings.SimplexStrategy = StrategyTableauRows
				h.must(m.LeftBasisTableau.Resize(7))
				h.must(m.LeftBasisTableau.Insert(6))
				h.must(m.LeftBasisTableau.Resize(2))
				want = ErrBounds
			case "ax":
				h.must(f.b.Data[0].Copy(h.m, h.pair("1", "0")))
				if sourceDebugEnabled {
					want = ErrAssertion
					earlyFailure = true
				}
			case "shape":
				h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
				h.must(f.b.Data[0].Copy(h.m, &f.x.Data[0]))
				if sourceDebugEnabled {
					want = ErrAssertion
					earlyFailure = true
				}
			case "membership", "membership-name":
				h.must(m.InfSet.Insert(1))
				m.Names = lpFeasibleNames{}
				if profile == "membership-name" {
					m.Names = nil
				}
				if sourceDebugEnabled {
					want = ErrAssertion
					if m.Names == nil {
						want = ErrBounds
					}
				}
				// Prevent later reduced-cost checks from obscuring this native profile.
				m.LookForFeasibleSolutionOnly = false
				f.settings.SimplexStrategy = StrategyTableauRows
			case "queue":
				f.settings.UseBreakpointsInFeasibilitySearch = true
				m.BreakpointQueue.decreaseArg = NewIndexedPairVector(h.m, 1)
				want = ErrExactQueuePending
			case "reduced-cost":
				h.must(h.m.SetInt32(&m.D.Data[1], 9))
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "pivot":
				// Keep AX and singleton shape valid; rows strategy skips reduced costs.
				v, e := f.a.ColumnValue(f.a.Columns[0][0])
				h.must(e)
				h.must(h.m.SetInt32(v, 2))
				h.must(f.b.Data[0].Copy(h.m, &f.x.Data[0]))
				f.settings.SimplexStrategy = StrategyTableauRows
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			var err error
			out := lpCaptureFeasibleStdout(t, func() { err = m.InitRunTableau() })
			f.x.Data = savedX
			f.basis = savedBasis
			if profile == "row-clear" {
				h.must(m.LeftBasisTableau.Resize(7))
			}
			if err != want {
				t.Fatal(profile, err, want, out)
			}
			if earlyFailure {
				if m.TotalIterations != 37 || m.BasisSortCounter != 19 {
					t.Fatal("early counter mutation")
				}
			} else if m.TotalIterations != 0 || m.BasisSortCounter != 0 {
				t.Fatal("counter prefix")
			}
			if err != nil {
				if retry := m.InitRunTableau(); retry != ErrExactVectorPending {
					t.Fatal("retry", retry)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactInitTableauPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		success, failures := false, 0
		for ordinal := int64(0); ordinal < 250; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f, m := lpInitTableauFixture(t, h, false)
				huge := strings.Repeat("1234567890", 100) + "1/11"
				h.must(h.m.SetString(&f.costs.Data[1], huge))
				h.must(m.InitReducedCostsTableau())
				f.settings.BackupCosts = true
				f.settings.UseBreakpointsInFeasibilitySearch = true
				m.TotalIterations = 71
				m.BasisSortCounter = 19
				h.budget.FailAfter(ordinal)
				err := m.InitRunTableau()
				h.budget.FailAfter(-1)
				if err == nil {
					success, done = true, true
					lpSparseRationalCheck(t, h, &m.CostsBackup.Data[1], mustLPRat(t, huge))
					if len(m.BreakpointQueue.priorities.Data) != 2 {
						t.Fatal("queue not resized")
					}
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if m.calls.pending() && m.PairCoreSolverBase.calls.pending() {
						t.Fatal("adopted child capture")
					}
					if retry := m.InitRunTableau(); retry != ErrExactVectorPending {
						t.Fatal("retry", retry)
					}
				}
				if m.TotalIterations != 0 || m.BasisSortCounter != 0 {
					t.Fatal("failure precedes counters unexpectedly")
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
			t.Fatal("campaign", direct, failures)
		}
	}
}

func TestLPExactTableauBasisReachedMetadata(t *testing.T) {
	for _, profile := range []string{"stopped", "basis-extent", "basis-index", "cell", "cross-link", "shape-index", "shape-short-circuit", "empty"} {
		t.Run(profile, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpInitTableauFixture(t, h, false)
			savedBasis := append([]uint32(nil), f.basis...)
			savedCell := f.a.Columns[0][0]
			var err error
			switch profile {
			case "stopped":
				m.stopped = true
				if _, err = m.BasisColumnIsSetCorrectly(0); err != ErrExactVectorPending {
					t.Fatal(err)
				}
				if _, err = m.BasisColumnsAreSetCorrectly(); err != ErrExactVectorPending {
					t.Fatal(err)
				}
				if err = m.InitTableauRows(); err != ErrExactVectorPending {
					t.Fatal(err)
				}
				m.PairCoreSolverBase.stopped = true
				if _, err = m.NeedToPivotToBasisTableau(); err != ErrExactVectorPending {
					t.Fatal(err)
				}
			case "basis-extent":
				f.basis = nil
				if _, err = m.NeedToPivotToBasisTableau(); err != ErrBounds {
					t.Fatal(err)
				}
			case "basis-index":
				f.basis[0] = 9
				if _, err = m.NeedToPivotToBasisTableau(); err != ErrBounds {
					t.Fatal(err)
				}
			case "cross-link":
				savedOffset := f.a.Rows[0][0].Offset
				f.a.Rows[0][0].Offset = 9
				pivot, e := m.NeedToPivotToBasisTableau()
				if sourceDebugEnabled {
					if e != ErrAssertion {
						t.Fatal(e)
					}
				} else if e != nil || pivot {
					t.Fatal(pivot, e)
				}
				f.a.Rows[0][0].Offset = savedOffset
			case "cell":
				f.a.Columns[0][0].Offset = 9
				want := ErrBounds
				if _, err = m.NeedToPivotToBasisTableau(); err != want {
					t.Fatal(err, want)
				}
			case "shape-index":
				f.basis[0] = 9
				if _, err = m.BasisColumnsAreSetCorrectly(); err != ErrBounds {
					t.Fatal(err)
				}
			case "shape-short-circuit":
				f.basis = []uint32{0, 9}
				saved := f.a.Columns[0]
				f.a.Columns[0] = nil
				ok, e := m.BasisColumnsAreSetCorrectly()
				if e != nil || ok {
					t.Fatal("observed later invalid basis", ok, e)
				}
				f.a.Columns[0] = saved
			case "empty":
				f.basis = nil
				saveRows := f.a.Rows
				f.a.Rows = nil
				saveCols := f.a.Columns
				f.a.Columns = nil
				ok, e := m.BasisColumnsAreSetCorrectly()
				h.must(e)
				if !ok {
					t.Fatal("empty shape")
				}
				pivot, e := m.NeedToPivotToBasisTableau()
				h.must(e)
				if pivot {
					t.Fatal("empty pivot")
				}
				f.a.Rows = saveRows
				f.a.Columns = saveCols
			}
			f.basis = savedBasis
			f.a.Columns[0][0] = savedCell
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactInitTableauRowsFailurePrefixes(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpInitTableauFixture(t, h, false)
	h.must(m.LeftBasisTableau.Resize(7))
	h.must(m.LeftBasisTableau.Insert(6))
	// The source clear detects a member left outside a previously shrunken domain.
	h.must(m.LeftBasisTableau.Resize(2))
	m.BlandModeTableau = true
	m.LeftBasisRepeated = 23
	if e := m.InitTableauRows(); e != ErrBounds {
		t.Fatal(e)
	}
	if m.BlandModeTableau || m.LeftBasisRepeated != 23 || m.LeftBasisTableau.Size() != 1 {
		t.Fatal("clear prefix")
	}
	if e := m.InitTableauRows(); e != ErrExactVectorPending {
		t.Fatal(e)
	}
	h.must(m.LeftBasisTableau.Resize(7))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactInitTableauPreservesOlderComparison(t *testing.T) {
	for _, only := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpInitTableauFixture(t, h, false)
			m.LookForFeasibleSolutionOnly = only
			f.settings.BackupCosts = true
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
				err = m.InitRunTableau()
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
			t.Fatal("no actual older comparison", only)
		}
	}
}
