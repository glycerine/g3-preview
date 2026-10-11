package lp

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPRationalCoreUpdateIndependentSystems(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		r, n := 1+world%4, 3+world%4
		f := lpNewRationalCoreFixture(t, h, uint32(r), uint32(n))
		f.settings.SimplexStrategy = StrategyLU
		f.settings.DensityThreshold = 1e100
		if world%2 == 0 {
			f.settings.DensityThreshold = 0
		}
		perm := make([]int, n)
		for j := range perm {
			perm[j] = (j + world%n) % n
		}
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		leaveRow := world % r
		en, lv := uint32(perm[r]), f.basis[leaveRow]
		b := make([][]*big.Rat, r)
		d, before := make([]*big.Rat, r), make([]*big.Rat, r)
		for i := 0; i < r; i++ {
			d[i] = big.NewRat(int64((world+i)%7-3), 7)
			if i == leaveRow {
				d[i].SetInt64(1)
			}
			num := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), uint(81+world)), big.NewInt(int64(i+1)))
			before[i] = new(big.Rat).SetFrac(num, big.NewInt(19))
			if world%5 == 0 && i == leaveRow {
				before[i].SetInt64(0)
			}
			h.must(f.x.copySlot(&f.x.Data[perm[i]], &h.pair(before[i].RatString(), "0").X))
			b[i] = make([]*big.Rat, r)
			for j := 0; j < r; j++ {
				v := new(big.Rat)
				if j < i {
					v.SetFrac64(int64((world+i+j)%9-4), 17)
				}
				if i == j {
					v.SetInt64(int64(13 + i))
				}
				b[i][j] = v
			}
		}
		for i := 0; i < r; i++ {
			incoming, rhs := new(big.Rat), new(big.Rat)
			for j := 0; j < r; j++ {
				h.must(f.a.Set(uint32(i), uint32(perm[j]), &h.pair(b[i][j].RatString(), "0").X))
				incoming.Add(incoming, new(big.Rat).Mul(b[i][j], d[j]))
				rhs.Add(rhs, new(big.Rat).Mul(b[i][j], before[j]))
			}
			h.must(f.a.Set(uint32(i), en, &h.pair(incoming.RatString(), "0").X))
			h.must(f.b.copySlot(&f.b.Data[i], &h.pair(rhs.RatString(), "0").X))
		}
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, n)
		m.RowsNZ = make([]uint32, r)
		h.must(m.SolveBd(en))
		h.must(m.CalculatePivotRowOfBInverse(uint32(leaveRow)))
		h.must(m.CalculatePivotRowWhenInverseReady(uint32(leaveRow)))
		lpSparseRationalCheck(t, h, &m.PivotRow.Data[en], big.NewRat(1, 1))
		forced := world%3 == 0
		old := m.Factorization
		if forced {
			old.RefactorCounter = 200
		}
		m.Status = StatusOptimal
		m.IterationsNoCostGrowth = 43
		m.LookForFeasibleSolutionOnly = world%2 == 0
		m.StartTracingBasisChanges()
		slot := -f.heading[en] - 1
		step := h.pair(before[leaveRow].RatString(), "0")
		ok, e := m.UpdateBasisAndX(int32(en), int32(lv), &step.X)
		h.must(e)
		if !ok || f.basis[leaveRow] != en || f.nb[slot] != lv || f.heading[en] != int32(leaveRow) || f.heading[lv] != -slot-1 || !reflect.DeepEqual(m.Trace, []uint32{en, lv}) {
			t.Fatal(world, "basis prefix", ok, f.basis, f.nb, f.heading, m.Trace)
		}
		if m.Status != StatusOptimal || m.IterationsNoCostGrowth != 43 {
			t.Fatal("unexpected status/counter")
		}
		for j := 0; j < n; j++ {
			want := new(big.Rat)
			if uint32(j) == en {
				want.Set(before[leaveRow])
			}
			for i, col := range perm[:r] {
				if col == j {
					want.Sub(before[i], new(big.Rat).Mul(d[i], before[leaveRow]))
				}
			}
			lpSparseRationalCheck(t, h, &f.x.Data[j], want)
		}
		if forced {
			if old == m.Factorization || !old.closed {
				t.Fatal("refactor lifecycle")
			}
		} else if old != m.Factorization || old.RefactorCounter != 1 {
			t.Fatal("replace lifecycle")
		}
		// Independently verify original rows at the new point, then use the
		// newly updated LU to reconstruct that point after perturbing basics.
		off, e := m.AMultXIsOff()
		h.must(e)
		if off {
			t.Fatal("pivot changed A*x")
		}
		for _, j := range f.basis {
			h.must(h.m.SetInt32(&f.x.Data[j], 91))
		}
		ok, e = m.FindXBySolving()
		h.must(e)
		if !ok {
			t.Fatal("updated LU solve")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCoreUpdateSingularRollback(t *testing.T) {
	for _, forced := range []bool{false, true} {
		for _, only := range []bool{false, true} {
			for _, nonzero := range []bool{false, true} {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 2, 3)
				f.settings.SimplexStrategy = StrategyLU
				one := h.pair("1", "0")
				h.must(f.a.Set(0, 0, &one.X))
				h.must(f.a.Set(1, 1, &one.X))
				h.must(f.a.Set(1, 2, &one.X))
				h.must(f.x.copySlot(&f.x.Data[0], &h.pair("3", "0").X))
				h.must(f.x.copySlot(&f.x.Data[1], &h.pair("7", "0").X))
				for i := range f.b.Data {
					h.must(f.b.copySlot(&f.b.Data[i], &f.x.Data[i]))
				}
				m, e := NewRationalCoreSolverBase(f.input())
				h.must(e)
				m.ColumnsNZ = make([]uint32, 3)
				m.RowsNZ = make([]uint32, 2)
				h.must(m.SolveBd(2))
				h.must(m.CalculatePivotRowOfBInverse(0))
				h.must(m.CalculatePivotRowWhenInverseReady(0))
				for i := range m.CopyXB.Data {
					h.must(m.CopyXB.copySlot(&m.CopyXB.Data[i], &f.x.Data[i]))
				}
				m.LookForFeasibleSolutionOnly = only
				m.IterationsNoCostGrowth = 43
				if forced {
					m.Factorization.RefactorCounter = 200
				}
				m.StartTracingBasisChanges()
				step := h.pair("0", "0")
				if nonzero {
					step = h.pair("2/7", "0")
				}
				ok, e := m.UpdateBasisAndX(2, 0, &step.X)
				h.must(e)
				if ok || m.Status != StatusUnstable || m.IterationsNoCostGrowth != 44 || !reflect.DeepEqual(f.basis, []uint32{0, 1}) || len(m.Trace) != 0 || m.Factorization.GetStatus() != LUOK {
					t.Fatal("precise rollback", forced, only, nonzero, ok, m.Status, f.basis, m.Trace)
				}
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(3, 1))
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(7, 1))
				lpSparseRationalCheck(t, h, &f.x.Data[2], new(big.Rat))
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPRationalCoreUpdatePhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, phase := range []string{"replace", "refactor", "rollback", "restore"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 1500 && !done; ordinal++ {
				h := newLPPairHarness(t)
				if direct {
					h = lpNewDirectSparseHarness(t)
				}
				f := lpNewRationalCoreFixture(t, h, 2, 3)
				f.settings.SimplexStrategy = StrategyLU
				huge := strings.Repeat("1234567890", 15)
				large := h.pair(huge+"7/13", "0")
				for i := uint32(0); i < 2; i++ {
					h.must(f.a.Set(i, i, &h.pair(fmt.Sprint(3+i), "0").X))
					if phase != "rollback" || i != 0 {
						h.must(f.a.Set(i, 2, &large.X))
					}
				}
				m, e := NewRationalCoreSolverBase(f.input())
				h.must(e)
				m.ColumnsNZ = make([]uint32, 3)
				m.RowsNZ = make([]uint32, 2)
				h.must(m.SolveBd(2))
				h.must(m.CalculatePivotRowOfBInverse(0))
				h.must(m.CalculatePivotRowWhenInverseReady(0))
				step := h.pair(huge+"1/17", "0")
				if phase == "refactor" || phase == "rollback" {
					m.Factorization.RefactorCounter = 200
				}
				if phase == "restore" {
					h.must(m.UpdateX(2, &step.X))
					h.must(m.ChangeBasis(2, 0))
				}
				m.IterationsNoCostGrowth = 43
				h.budget.FailAfter(ordinal)
				var err error
				var ok bool
				if phase == "restore" {
					err = m.RestoreXAndRefactor(2, 0, &step.X)
				} else {
					ok, err = m.UpdateBasisAndX(2, 0, &step.X)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
					if phase == "rollback" {
						if ok || m.Status != StatusUnstable || m.IterationsNoCostGrowth != 44 {
							t.Fatal("rollback outcome")
						}
					} else if phase != "restore" && !ok {
						t.Fatal("replace outcome")
					}
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(direct, phase, ordinal, err)
					}
					if _, e = m.UpdateBasisAndX(2, 0, &step.X); e != ErrExactVectorPending {
						t.Fatal("stopped retry", e)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
			if !done || direct && failures == 0 {
				t.Fatal("incomplete physical campaign", direct, phase, failures)
			}
		}
	}
}

func TestLPRationalCoreRestoreDiagnosticsAndGuards(t *testing.T) {
	for _, phase := range []string{"degenerate", "residual", "writer-degenerate", "writer-residual", "stored-backup", "nil-step", "nil-lu", "negative-entering", "negative-leaving", "short-pivot"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			one, two, zero := h.pair("1", "0"), h.pair("2", "0"), h.pair("0", "0")
			h.must(f.a.Set(0, 0, &one.X))
			h.must(f.a.Set(0, 1, &two.X))
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			m.Status = StatusOptimal
			m.ColumnsNZ = make([]uint32, 2)
			m.RowsNZ = make([]uint32, 1)
			h.must(m.SolveBd(1))
			h.must(m.CalculatePivotRowOfBInverse(0))
			h.must(m.CalculatePivotRowWhenInverseReady(0))
			var out bytes.Buffer
			m.Settings.DebugOut = &out
			if strings.HasPrefix(phase, "writer") {
				m.Settings.DebugOut = lpResidualWriterOnly{writer: &lpResidualFailWriter{remaining: 0}}
			}
			var err error
			guard := false
			switch phase {
			case "nil-step":
				_, err = m.UpdateBasisAndX(1, 0, nil)
				guard = true
			case "nil-lu":
				h.must(m.Factorization.Close())
				m.Factorization = nil
				_, err = m.UpdateBasisAndX(1, 0, &zero.X)
				guard = true
			case "negative-entering":
				_, err = m.UpdateBasisAndX(-1, 0, &zero.X)
				guard = true
			case "negative-leaving":
				_, err = m.UpdateBasisAndX(1, -1, &zero.X)
				guard = true
			case "short-pivot":
				owner := m.PivotRow.Data
				m.PivotRow.Data = nil
				_, err = m.UpdateBasisAndX(1, 0, &zero.X)
				m.PivotRow.Data = owner
				guard = true
			case "degenerate", "writer-degenerate":
				h.must(h.m.SetInt32(&f.a.Rows[0][0].Value, 0))
			case "residual", "writer-residual":
				h.must(f.b.copySlot(&f.b.Data[0], &one.X))
			case "stored-backup":
				h.must(f.x.copySlot(&f.x.Data[0], &h.pair("3", "0").X))
				h.must(f.b.copySlot(&f.b.Data[0], &f.x.Data[0]))
				h.must(m.UpdateX(1, &one.X))
				h.must(m.ChangeBasis(1, 0))
			}
			if guard {
				if err != ErrBounds {
					t.Fatal(phase, err)
				}
			} else {
				step := &zero.X
				if phase == "stored-backup" {
					step = &one.X
				}
				stdout := lpCaptureFeasibleStdout(t, func() { err = m.RestoreXAndRefactor(1, 0, step) })
				if strings.HasPrefix(phase, "writer") {
					if err == nil || m.Status != StatusOptimal {
						t.Fatal("writer publication", err, m.Status)
					}
				} else {
					h.must(err)
					if m.Status != StatusFloatingPointError {
						t.Fatal("restore outcome", m.Status)
					}
					want := "cannot restore solution\n"
					if phase == "degenerate" {
						want = "cannot refactor\n"
					}
					if out.String() != want {
						t.Fatal(out.String(), want)
					}
				}
				if phase == "residual" && stdout != "precise x is off (m_b[0] = 1 left side = 0 delta = 1 iters = 0)\n" {
					t.Fatal(stdout)
				}
				if phase == "stored-backup" {
					lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
					if stdout == "" {
						t.Fatal("precise backup was silently saved")
					}
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if _, e = m.UpdateBasisAndX(0, 1, &zero.X); e != ErrExactVectorPending {
				t.Fatal("closed update", e)
			}
			if e = m.RestoreXAndRefactor(0, 1, &zero.X); e != ErrExactVectorPending {
				t.Fatal("closed restore", e)
			}
			if e = m.InitLU(); e != ErrExactVectorPending {
				t.Fatal("closed refactor", e)
			}
		})
	}
}

func TestLPRationalCoreUpdateReachedGuards(t *testing.T) {
	for _, phase := range []string{"nil", "heading", "pivot", "missing-lu", "stopped-lu", "change-prefix", "forced-change-prefix", "restore-nil", "restore-heading", "stopped-restore", "stopped-basis", "restore-early"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			m.ColumnsNZ = make([]uint32, 2)
			m.RowsNZ = make([]uint32, 1)
			h.must(m.SolveBd(1))
			h.must(m.CalculatePivotRowOfBInverse(0))
			h.must(m.CalculatePivotRowWhenInverseReady(0))
			old := m.Factorization
			step := &h.pair("0", "0").X
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
				step = &h.pair("1", "0").X
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
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(1, 1))
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-2, 3))
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

func TestLPRationalCoreUpdateOlderComparison(t *testing.T) {
	for _, op := range []string{"update", "restore", "init-lu", "restore-basis"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.must(f.a.Set(0, 1, &v.X))
				m, e := NewRationalCoreSolverBase(f.input())
				h.must(e)
				m.ColumnsNZ = make([]uint32, 2)
				m.RowsNZ = make([]uint32, 1)
				h.must(m.SolveBd(1))
				h.must(m.CalculatePivotRowOfBInverse(0))
				h.must(m.CalculatePivotRowWhenInverseReady(0))
				step := &h.pair("0", "0").X
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

func TestLPRationalCoreUpdateMissingBackupAndLiveStep(t *testing.T) {
	for _, phase := range []string{"missing-backup", "entering-alias", "leaving-alias", "leaving-alias-forward"} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 3)
		f.settings.SimplexStrategy = StrategyLU
		one := h.pair("1", "0")
		h.must(f.a.Set(0, 0, &one.X))
		h.must(f.a.Set(1, 1, &one.X))
		if phase != "missing-backup" {
			h.must(f.a.Set(0, 2, &one.X))
		}
		h.must(f.a.Set(1, 2, &one.X))
		h.must(h.m.SetInt32(&f.x.Data[0], 3))
		h.must(h.m.SetInt32(&f.x.Data[1], 7))
		if phase == "entering-alias" {
			h.must(h.m.SetInt32(&f.x.Data[2], 1))
		}
		// Original b is independently xB + x_entering in each stored row.
		for i := range f.b.Data {
			v := int32(3 + 4*i)
			if phase == "entering-alias" {
				v++
			}
			h.must(h.m.SetInt32(&f.b.Data[i], v))
		}
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.ColumnsNZ = make([]uint32, 3)
		m.RowsNZ = make([]uint32, 2)
		h.must(m.SolveBd(2))
		h.must(m.CalculatePivotRowOfBInverse(0))
		h.must(m.CalculatePivotRowWhenInverseReady(0))
		step := &one.X
		if phase == "entering-alias" {
			step = &f.x.Data[2]
		}
		if strings.HasPrefix(phase, "leaving-alias") {
			step = &f.x.Data[0]
			// Both are valid index orders for this dense nonzero direction.
			m.Ed.Index = []uint32{1, 0}
			if phase == "leaving-alias-forward" {
				m.Ed.Index = []uint32{0, 1}
			}
		}
		var ok bool
		var err error
		output := lpCaptureFeasibleStdout(t, func() { ok, err = m.UpdateBasisAndX(2, 0, step) })
		h.must(err)
		switch phase {
		case "missing-backup":
			if ok || m.Status != StatusFloatingPointError || m.IterationsNoCostGrowth != 0 || !strings.Contains(output, "m_b[1] = 7 left side = 0 delta = 7") {
				t.Fatal("missing source backup", ok, m.Status, output)
			}
		case "entering-alias":
			// Entering first becomes2, then the live delta2 subtracts from
			// each basic value. Precise update does not repair the residual.
			if !ok || output != "" {
				t.Fatal(ok, output)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[2], big.NewRat(2, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(1, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(5, 1))
		case "leaving-alias", "leaving-alias-forward":
			// Row0 changes live delta3 to0. Row1 therefore receives either
			// the initial delta or zero, according to the actual index order.
			if !ok || output != "" {
				t.Fatal(ok, output)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[2], big.NewRat(3, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
			want := int64(4)
			if phase == "leaving-alias-forward" {
				want = 7
			}
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(want, 1))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
