package lp

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLPPairCoreUpdateIndependentSystems(t *testing.T) {
	rng := rand.New(rand.NewPCG(813991, 7193))
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		rows := 1 + world%4
		cols := rows + 2
		f := lpNewPairCoreFixture(t, h, uint32(rows), uint32(cols))
		f.settings.SimplexStrategy = StrategyLU
		f.settings.DensityThreshold = 1e100
		if world%2 == 0 {
			f.settings.DensityThreshold = 0
		}
		perm := rng.Perm(cols)
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		leaveRow := world % rows
		en, lv := uint32(perm[rows]), f.basis[leaveRow]
		b := make([][]*big.Rat, rows)
		d := make([]*big.Rat, rows)
		before := make([][2]*big.Rat, rows)
		for i := range b {
			d[i] = big.NewRat(int64(rng.IntN(7)-3), 5)
			if i == leaveRow {
				d[i].SetInt64(1)
			}
			before[i] = [2]*big.Rat{big.NewRat(int64(rng.IntN(13)-6), 7), big.NewRat(int64(rng.IntN(9)-4), 11)}
			if world%5 == 0 && i == leaveRow {
				before[i] = [2]*big.Rat{new(big.Rat), new(big.Rat)}
			}
			h.must(f.x.Data[perm[i]].Copy(h.m, h.pair(before[i][0].RatString(), before[i][1].RatString())))
			b[i] = make([]*big.Rat, rows)
			for j := range b[i] {
				b[i][j] = big.NewRat(int64(rng.IntN(5)-2), 11)
				if i == j {
					b[i][j].SetInt64(int64(13 + rows + i))
				}
			}
		}
		for i := range b {
			incoming, rx, ry := new(big.Rat), new(big.Rat), new(big.Rat)
			for j, v := range b[i] {
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(perm[j]), &h.pair(v.RatString(), "0").X))
				}
				incoming.Add(incoming, new(big.Rat).Mul(v, d[j]))
				rx.Add(rx, new(big.Rat).Mul(v, before[j][0]))
				ry.Add(ry, new(big.Rat).Mul(v, before[j][1]))
			}
			if incoming.Sign() != 0 {
				h.must(f.a.Set(uint32(i), en, &h.pair(incoming.RatString(), "0").X))
			}
			h.must(f.b.Data[i].Copy(h.m, h.pair(rx.RatString(), ry.RatString())))
		}
		m, e := NewPairCoreSolverBase(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, cols)
		m.RowsNZ = make([]uint32, len(f.basis))
		h.must(m.SolveBd(en))
		h.must(m.CalculatePivotRowOfBInverse(uint32(leaveRow)))
		h.must(m.CalculatePivotRowWhenInverseReady(uint32(leaveRow)))
		lpSparseRationalCheck(t, h, &m.PivotRow.Data[en], big.NewRat(1, 1))
		m.LookForFeasibleSolutionOnly = world/7%2 == 0
		m.IterationsNoCostGrowth = 43
		m.Status = StatusOptimal
		forced := world%3 == 0
		if forced {
			m.Factorization.RefactorCounter = 200
		}
		old := m.Factorization
		slot := -f.heading[en] - 1
		m.StartTracingBasisChanges()
		step := h.pair(before[leaveRow][0].RatString(), before[leaveRow][1].RatString())
		ok, err := m.UpdateBasisAndX(int32(en), int32(lv), step)
		h.must(err)
		if !ok || f.basis[leaveRow] != en || f.nb[slot] != lv || f.heading[lv] != -slot-1 || f.heading[en] != int32(leaveRow) || !reflect.DeepEqual(m.Trace, []uint32{en, lv}) {
			t.Fatal(world, "basis publication")
		}
		if m.Status != StatusOptimal || m.IterationsNoCostGrowth != 43 {
			t.Fatal("unexpected source status/counter update")
		}
		expected := make([][2]*big.Rat, cols)
		for j := range expected {
			expected[j] = [2]*big.Rat{new(big.Rat), new(big.Rat)}
		}
		expected[en] = [2]*big.Rat{new(big.Rat).Set(before[leaveRow][0]), new(big.Rat).Set(before[leaveRow][1])}
		for i, j := range perm[:rows] {
			for c := 0; c < 2; c++ {
				expected[j][c].Sub(before[i][c], new(big.Rat).Mul(d[i], before[leaveRow][c]))
			}
		}
		for j := range expected {
			h.check(&f.x.Data[j], expected[j][0], expected[j][1])
		}
		if forced {
			if m.Factorization == old || !old.closed {
				t.Fatal("old factorization not replaced")
			}
		} else if m.Factorization != old || old.RefactorCounter != 1 {
			t.Fatal("column replacement was not used")
		}
		rhs := NewIndexedPairVector(h.m, uint32(rows))
		for i := range rhs.Data {
			h.must(rhs.Data[i].Copy(h.m, &f.b.Data[i]))
		}
		h.must(m.Factorization.SolveBy(rhs.Data))
		for i, j := range f.basis {
			h.check(&rhs.Data[i], expected[j][0], expected[j][1])
		}
		h.must(rhs.Close())
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreUpdateSingularRollback(t *testing.T) {
	for _, only := range []bool{false, true} {
		for _, forced := range []bool{false, true} {
			for _, nonzero := range []bool{false, true} {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 2, 3)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
				h.must(f.a.Set(1, 1, &h.pair("1", "0").X))
				h.must(f.a.Set(1, 2, &h.pair("1", "0").X))
				h.must(f.x.Data[0].Copy(h.m, h.pair("3", "5")))
				h.must(f.x.Data[1].Copy(h.m, h.pair("7", "11")))
				for i := range f.b.Data {
					h.must(f.b.Data[i].Copy(h.m, &f.x.Data[i]))
				}
				m, e := NewPairCoreSolverBase(f.input())
				h.must(e)
				m.ColumnsNZ = make([]uint32, 3)
				m.RowsNZ = make([]uint32, len(f.basis))
				h.must(m.SolveBd(2))
				h.must(m.CalculatePivotRowOfBInverse(0))
				h.must(m.CalculatePivotRowWhenInverseReady(0))
				// Precise UpdateX never writes CopyXB. Supply the real caller backup.
				for i := range m.CopyXB.Data {
					h.must(m.CopyXB.Data[i].Copy(h.m, &f.x.Data[i]))
				}
				m.LookForFeasibleSolutionOnly = only
				m.IterationsNoCostGrowth = 43
				if forced {
					m.Factorization.RefactorCounter = 200
				}
				m.StartTracingBasisChanges()
				step := h.pair("0", "0")
				if nonzero {
					step = h.pair("2/7", "3/11")
				}
				var ok bool
				var err error
				stdout := lpCaptureFeasibleStdout(t, func() { ok, err = m.UpdateBasisAndX(2, 0, step) })
				h.must(err)
				if ok || m.Status != StatusUnstable || m.IterationsNoCostGrowth != 44 || stdout != "" {
					t.Fatal(only, forced, nonzero, ok, m.Status, m.IterationsNoCostGrowth, stdout)
				}
				if !reflect.DeepEqual(f.basis, []uint32{0, 1}) || !reflect.DeepEqual(f.nb, []uint32{2}) || len(m.Trace) != 0 || m.Factorization == nil || m.Factorization.GetStatus() != LUOK {
					t.Fatal("precise rollback did not restore basis/LU")
				}
				h.check(&f.x.Data[0], big.NewRat(3, 1), big.NewRat(5, 1))
				h.check(&f.x.Data[1], big.NewRat(7, 1), big.NewRat(11, 1))
				h.check(&f.x.Data[2], big.NewRat(0, 1), big.NewRat(0, 1))
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPPairCoreUpdatePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"replace", "refactor", "rollback", "restore"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1500; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 2, 3)
					f.settings.SimplexStrategy = StrategyLU
					huge := strings.Repeat("1234567890", 10)
					for i := uint32(0); i < 2; i++ {
						diag := fmt.Sprint(3 + i)
						if op == "restore" && i == 0 {
							diag = huge + "3/23"
						}
						h.must(f.a.Set(i, i, &h.pair(diag, "0").X))
						if op != "rollback" || i != 0 {
							h.must(f.a.Set(i, 2, &h.pair(huge+"7/13", "0").X))
						}
					}
					m, e := NewPairCoreSolverBase(f.input())
					h.must(e)
					m.ColumnsNZ = make([]uint32, 3)
					m.RowsNZ = make([]uint32, len(f.basis))
					h.must(m.SolveBd(2))
					h.must(m.CalculatePivotRowOfBInverse(0))
					h.must(m.CalculatePivotRowWhenInverseReady(0))
					step := h.pair(huge+"1/17", huge+"3/19")
					if op == "refactor" || op == "rollback" {
						m.Factorization.RefactorCounter = 200
					}
					if op == "restore" {
						h.must(m.UpdateX(2, step))
						h.must(m.ChangeBasis(2, 0))
					}
					m.IterationsNoCostGrowth = 43
					h.budget.FailAfter(ordinal)
					var err error
					var ok bool
					if op == "restore" {
						err = m.RestoreXAndRefactor(2, 0, step)
					} else {
						ok, err = m.UpdateBasisAndX(2, 0, step)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						if op == "rollback" {
							if ok || m.Status != StatusUnstable || m.IterationsNoCostGrowth != 44 {
								t.Fatal("rollback result")
							}
						} else if op != "restore" && !ok {
							t.Fatal("replacement failed")
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.Factorization != nil && m.Factorization.stopped && m.calls.pending() {
							t.Fatal("adopted LU arithmetic")
						}
						if e = m.InitLU(); e != ErrExactVectorPending {
							t.Fatal(e)
						}
						if _, e = m.UpdateBasisAndX(2, 0, step); e != ErrExactVectorPending {
							t.Fatal(e)
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
			if !success || (direct && failures == 0) {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPPairCoreRestoreSourceDiagnostics(t *testing.T) {
	for _, phase := range []string{"degenerate", "residual", "writer-degenerate", "writer-residual", "stored-backup"} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 2)
		f.settings.SimplexStrategy = StrategyLU
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
		m, e := NewPairCoreSolverBase(f.input())
		h.must(e)
		m.Status = StatusOptimal
		var out bytes.Buffer
		m.Settings.DebugOut = &out
		if strings.Contains(phase, "writer") {
			m.Settings.DebugOut = &lpResidualWriterOnly{writer: &lpResidualFailWriter{remaining: 0}}
		}
		switch phase {
		case "degenerate", "writer-degenerate":
			// Make the original basis singular after construction; rollback must
			// diagnose the actual refactor rather than fabricate a status.
			p, e := f.a.ColumnValue(f.a.Columns[0][0])
			h.must(e)
			h.must(h.m.SetInt32(p, 0))
		case "residual", "writer-residual":
			h.must(f.b.Data[0].Copy(h.m, h.pair("1", "0")))
		case "stored-backup":
			h.must(f.x.Data[0].Copy(h.m, h.pair("3", "5")))
			h.must(f.b.Data[0].Copy(h.m, &f.x.Data[0]))
			h.must(m.Ed.SetValue(&h.pair("2", "0").X, 0))
			h.must(m.UpdateX(1, h.pair("1", "0")))
			h.must(m.ChangeBasis(1, 0))
		}
		step := h.pair("0", "0")
		if phase == "stored-backup" {
			step = h.pair("1", "0")
		}
		var err error
		stdout := lpCaptureFeasibleStdout(t, func() { err = m.RestoreXAndRefactor(1, 0, step) })
		if strings.Contains(phase, "writer") {
			if err == nil || m.Status != StatusOptimal {
				t.Fatal("writer prefix", phase, err, m.Status)
			}
		} else {
			h.must(err)
			if m.Status != StatusFloatingPointError {
				t.Fatal(phase, "missing failure status")
			}
			expected := "cannot restore solution\n"
			if phase == "degenerate" {
				expected = "cannot refactor\n"
			}
			if out.String() != expected {
				t.Fatal(out.String())
			}
		}
		if strings.Contains(phase, "residual") && stdout != "precise x is off (m_b[0] = (1, 0) left side = (0, 0) delta = (1, 0) iters = 0)\n" {
			t.Fatal(stdout)
		}
		if phase == "stored-backup" {
			// Exact UpdateX didn't populate CopyXB. The source restores its stored
			// zero, leaving a residual rather than silently saving the old value.
			h.check(&f.x.Data[0], big.NewRat(0, 1), big.NewRat(0, 1))
			if stdout == "" {
				t.Fatal("missing source residual with empty backup")
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreUpdateRollbackStoredBackup(t *testing.T) {
	for _, only := range []bool{false, true} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 3)
		f.settings.SimplexStrategy = StrategyLU
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.a.Set(1, 1, &h.pair("1", "0").X))
		h.must(f.a.Set(1, 2, &h.pair("1", "0").X))
		h.must(f.x.Data[0].Copy(h.m, h.pair("3", "5")))
		h.must(f.x.Data[1].Copy(h.m, h.pair("7", "11")))
		for i := range f.b.Data {
			h.must(f.b.Data[i].Copy(h.m, &f.x.Data[i]))
		}
		m, e := NewPairCoreSolverBase(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, 3)
		m.RowsNZ = make([]uint32, 2)
		h.must(m.SolveBd(2))
		h.must(m.CalculatePivotRowOfBInverse(0))
		h.must(m.CalculatePivotRowWhenInverseReady(0))
		m.Factorization.RefactorCounter = 200
		m.LookForFeasibleSolutionOnly = only
		m.IterationsNoCostGrowth = 43
		var out bytes.Buffer
		m.Settings.DebugOut = &out
		var ok bool
		var err error
		stdout := lpCaptureFeasibleStdout(t, func() { ok, err = m.UpdateBasisAndX(2, 0, h.pair("1", "0")) })
		h.must(err)
		if ok || m.Status != StatusFloatingPointError || m.IterationsNoCostGrowth != 43 || !reflect.DeepEqual(f.basis, []uint32{0, 1}) || m.Factorization.GetStatus() != LUOK {
			t.Fatal("failed exact rollback prefix", only)
		}
		// Only row1 was indexed by the old entering direction. Row0
		// remains unchanged; row1 is restored from its stored zero backup.
		if out.String() != "cannot restore solution\n" || stdout != "precise x is off (m_b[1] = (7, 11) left side = (0, 0) delta = (7, 11) iters = 0)\n" {
			t.Fatal(out.String(), stdout)
		}
		h.check(&f.x.Data[0], big.NewRat(3, 1), big.NewRat(5, 1))
		h.check(&f.x.Data[1], big.NewRat(0, 1), big.NewRat(0, 1))
		h.check(&f.x.Data[2], big.NewRat(0, 1), big.NewRat(0, 1))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreUpdateReachedGuards(t *testing.T) {
	for _, phase := range []string{"nil", "heading", "pivot", "missing-lu", "stopped-lu", "change-prefix", "forced-change-prefix", "restore-nil", "restore-heading", "stopped-restore", "stopped-basis", "restore-early"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			m, e := NewPairCoreSolverBase(f.input())
			h.must(e)
			m.ColumnsNZ = make([]uint32, 2)
			m.RowsNZ = make([]uint32, 1)
			h.must(m.SolveBd(1))
			h.must(m.CalculatePivotRowOfBInverse(0))
			h.must(m.CalculatePivotRowWhenInverseReady(0))
			old := m.Factorization
			step := h.pair("0", "0")
			want := error(ErrBounds)
			switch phase {
			case "nil":
				step = nil
			case "heading", "restore-heading":
				f.heading = nil
			case "pivot":
				h.must(m.PivotRow.resizeData(0))
			case "missing-lu":
				h.must(m.Factorization.Close())
				m.Factorization = nil
				step = h.pair("1", "0")
			case "stopped-lu":
				if e = m.Factorization.SolveBy(nil); e != ErrBounds {
					t.Fatal(e)
				}
				want = ErrLUStopped
			case "change-prefix", "forced-change-prefix":
				f.nb = nil
				if phase == "forced-change-prefix" {
					old.RefactorCounter = 200
				}
			case "restore-nil":
				step = nil
			case "stopped-restore", "stopped-basis":
				if e = m.RestoreBasisChange(99, 0); e != ErrBounds {
					t.Fatal(e)
				}
				want = ErrExactVectorPending
			case "restore-early":
				want = nil
			}
			var err error
			switch phase {
			case "restore-nil", "restore-heading", "stopped-restore":
				err = m.RestoreXAndRefactor(1, 0, step)
			case "stopped-basis":
				err = m.RestoreBasisChange(1, 0)
			case "restore-early":
				err = m.RestoreBasisChange(1, 99)
			default:
				_, err = m.UpdateBasisAndX(1, 0, step)
			}
			if err != want {
				t.Fatal(err, want)
			}
			if phase == "missing-lu" {
				h.check(&f.x.Data[1], big.NewRat(1, 1), big.NewRat(0, 1))
				h.check(&f.x.Data[0], big.NewRat(-2, 3), big.NewRat(0, 1))
			}
			if phase == "change-prefix" || phase == "forced-change-prefix" {
				if !reflect.DeepEqual(f.basis, []uint32{1}) || !reflect.DeepEqual(f.heading, []int32{-1, 0}) {
					t.Fatal("lost change-basis partial prefix")
				}
				if phase == "change-prefix" && old.RefactorCounter != 1 {
					t.Fatal("no actual preceding replacement")
				}
				if phase == "forced-change-prefix" && (m.Factorization != old || old.closed) {
					t.Fatal("refactored before metadata failure")
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPPairCoreUpdateOlderComparison(t *testing.T) {
	for _, op := range []string{"update", "restore", "init-lu", "restore-basis"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.must(f.a.Set(0, 1, &v.X))
				m, e := NewPairCoreSolverBase(f.input())
				h.must(e)
				m.ColumnsNZ = make([]uint32, 2)
				m.RowsNZ = make([]uint32, 1)
				h.must(m.SolveBd(1))
				h.must(m.CalculatePivotRowOfBInverse(0))
				h.must(m.CalculatePivotRowWhenInverseReady(0))
				step := h.pair("0", "0")
				if op != "update" {
					h.must(m.ChangeBasis(1, 0))
				}
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
				h.budget.FailAfter(ordinal)
				_, err := h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				old := h.m.ComparisonInvocationState()
				if old.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					switch op {
					case "update":
						_, err = m.UpdateBasisAndX(1, 0, step)
					case "restore":
						err = m.RestoreXAndRefactor(1, 0, step)
					case "init-lu":
						err = m.InitLU()
					case "restore-basis":
						err = m.RestoreBasisChange(1, 0)
					}
					if m.calls.comparison != 0 || f.a.calls.comparison != 0 {
						t.Fatal("adopted caller comparison", err)
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(m.Close())
				f.close()
				if old.Active && old != h.m.ComparisonInvocationState() {
					t.Fatal("cleanup consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}
