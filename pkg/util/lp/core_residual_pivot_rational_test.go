package lp

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestLPRationalCorePivotIndependentInverseRows(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 100; world++ {
		h := newLPPairHarness(t)
		r, n := 1+world%4, 4+world%4
		f := lpNewRationalCoreFixture(t, h, uint32(r), uint32(n))
		f.settings.SimplexStrategy = StrategyLU
		perm := make([]int, n)
		for j := range perm {
			perm[j] = (j + world%n) % n
		}
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		a := make([][]*big.Rat, r)
		for i := 0; i < r; i++ {
			a[i] = make([]*big.Rat, n)
			for j := 0; j < n; j++ {
				v := new(big.Rat)
				if j == i {
					v.SetInt64(1)
				} else if j < i || j >= r {
					num := new(big.Int).Lsh(big.NewInt(int64(1+i+j)), uint(81+world))
					if (i+j+world)%2 == 0 {
						num.Neg(num)
					}
					v.SetFrac(num, big.NewInt(23))
				}
				a[i][perm[j]] = v
				h.must(f.a.Set(uint32(i), uint32(perm[j]), &h.pair(v.RatString(), "0").X))
			}
		}
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.RowsNZ = make([]uint32, r)
		for pivot := 0; pivot < r; pivot++ {
			p := make([]*big.Rat, r)
			for j := r - 1; j >= 0; j-- {
				p[j] = new(big.Rat)
				if j == pivot {
					p[j].SetInt64(1)
				}
				for i := j + 1; i < r; i++ {
					p[j].Sub(p[j], new(big.Rat).Mul(p[i], a[i][perm[j]]))
				}
			}
			h.must(m.CalculatePivotRowOfBInverse(uint32(pivot)))
			for i := range p {
				lpSparseRationalCheck(t, h, &m.PivotRowB1.Data[i], p[i])
			}
			h.must(m.CalculatePivotRowWhenInverseReady(uint32(pivot)))
			count := uint32(0)
			for j := 0; j < n; j++ {
				want := new(big.Rat)
				if f.heading[j] < 0 {
					for i := 0; i < r; i++ {
						want.Add(want, new(big.Rat).Mul(p[i], a[i][j]))
					}
				}
				lpSparseRationalCheck(t, h, &m.PivotRow.Data[j], want)
				if want.Sign() != 0 {
					count++
				}
			}
			if m.RowsNZ[pivot] != count || m.PivotRow.Size() != count {
				t.Fatal("source precise nonzero count", m.RowsNZ, count)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCoreResidualIndependentRowsAndFindX(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		r, n := 1+world%4, 3+world%4
		f := lpNewRationalCoreFixture(t, h, uint32(r), uint32(n))
		f.settings.SimplexStrategy = StrategyLU
		point, dots := make([]*big.Rat, n), make([]*big.Rat, r)
		for j := range point {
			point[j] = big.NewRat(int64(world-57+3*j), 19)
			h.must(f.x.copySlot(&f.x.Data[j], &h.pair(point[j].RatString(), "0").X))
		}
		for i := 0; i < r; i++ {
			dots[i] = new(big.Rat)
			for j := 0; j < n; j++ {
				v := new(big.Rat)
				if i == j {
					v.SetInt64(1)
				} else if j < i || j >= r {
					v.SetFrac64(int64((world+i+j)%11-5), 17)
				}
				h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
				dots[i].Add(dots[i], new(big.Rat).Mul(v, point[j]))
			}
			h.must(f.b.copySlot(&f.b.Data[i], &h.pair(dots[i].RatString(), "0").X))
		}
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.TotalIterations = uint32(world + 7)
		mismatch := -1
		delta := new(big.Rat)
		if world%3 != 0 {
			mismatch = world % r
			delta.SetFrac(big.NewInt(int64(1-2*(world%2))), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))
			rhs := new(big.Rat).Add(dots[mismatch], delta)
			h.must(f.b.copySlot(&f.b.Data[mismatch], &h.pair(rhs.RatString(), "0").X))
		}
		var off bool
		var err error
		output := lpCaptureFeasibleStdout(t, func() { off, err = m.AMultXIsOff() })
		h.must(err)
		want := ""
		if mismatch >= 0 {
			rhs := new(big.Rat).Add(dots[mismatch], delta)
			want = fmt.Sprintf("precise x is off (m_b[%d] = %s left side = %s delta = %s iters = %d)\n", mismatch, rhs.RatString(), dots[mismatch].RatString(), delta.RatString(), m.TotalIterations)
		}
		if off != (mismatch >= 0) || output != want {
			t.Fatal(world, off, output, want)
		}
		// Solve reconstructs all basics and then runs the actual residual query.
		ok, e := m.FindXBySolving()
		h.must(e)
		if !ok {
			t.Fatal("independent nonsingular system failed residual check")
		}
		// Original point remains a solution when RHS was not perturbed.
		if mismatch < 0 {
			for j := range point {
				lpSparseRationalCheck(t, h, &f.x.Data[j], point[j])
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

type lpRationalResidualMutatingWriter struct {
	bytes.Buffer
	f        *lpRationalCoreFixture
	flushes  int
	flushErr error
}

func (w *lpRationalResidualMutatingWriter) Write(p []byte) (int, error) {
	if string(p) == "left side = " {
		if e := w.f.h.m.SetInt32(&w.f.x.Data[0], 5); e != nil {
			return 0, e
		}
	}
	return w.Buffer.Write(p)
}
func (w *lpRationalResidualMutatingWriter) Flush() error { w.flushes++; return w.flushErr }

func TestLPRationalCoreResidualDiagnosticPrefixesAndRecompute(t *testing.T) {
	want := "precise x is off (m_b[0] = 7 left side = 3 delta = 4 iters = 9)\n"
	for limit := 0; limit <= len(want)+1; limit++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 1)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.b.copySlot(&f.b.Data[0], &h.pair("7", "0").X))
		h.must(f.x.copySlot(&f.x.Data[0], &h.pair("3", "0").X))
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.TotalIterations = 9
		m.residualWork = NewIndexedRationalVector(h.m, 3)
		h.must(f.a.DotRow(0, f.x.Data, &m.residualWork.Data[0]))
		h.must(m.buildRowResidual(0))
		if limit < len(want) {
			w := &lpResidualFailWriter{remaining: limit}
			e = m.printRowResidual(0, lpResidualWriterOnly{w})
			if !errors.Is(e, lpErrResidualWrite) || w.String() != want[:limit] {
				t.Fatal(limit, e, w.String())
			}
		} else {
			w := &lpRationalResidualMutatingWriter{f: f}
			if limit > len(want) {
				w.flushErr = lpErrResidualWrite
			}
			e = m.printRowResidual(0, w)
			if e != w.flushErr || w.flushes != 1 || !strings.Contains(w.String(), "left side = 5 delta = 4") {
				t.Fatal("recompute/flush", e, w.flushes, w.String())
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCoreResidualPivotPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		for _, phase := range []string{"residual", "find", "inverse", "pivot", "diagnostic"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 900 && !done; ordinal++ {
				h := newLPPairHarness(t)
				if direct {
					h = lpNewDirectSparseHarness(t)
				}
				f := lpNewRationalCoreFixture(t, h, 2, 3)
				f.settings.SimplexStrategy = StrategyLU
				large := h.pair(strings.Repeat("1234567890", 20)+"7/13", "0")
				one := h.pair("1", "0")
				h.must(f.a.Set(0, 0, &one.X))
				h.must(f.a.Set(1, 1, &one.X))
				h.must(f.a.Set(1, 0, &large.X))
				h.must(f.a.Set(0, 2, &large.X))
				h.must(f.a.Set(1, 2, &large.X))
				h.must(f.x.copySlot(&f.x.Data[2], &large.X))
				m, e := NewRationalCoreSolverBase(f.input())
				h.must(e)
				// A*x = b without using any newly ported residual helper.
				if phase == "residual" {
					v, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 20) + "7/13")
					v.Mul(v, v)
					for i := range f.b.Data {
						h.must(f.b.copySlot(&f.b.Data[i], &h.pair(v.RatString(), "0").X))
					}
				}
				m.RowsNZ = make([]uint32, 2)
				if phase == "pivot" {
					h.must(m.CalculatePivotRowOfBInverse(1))
				}
				if phase == "diagnostic" {
					m.residualWork = NewIndexedRationalVector(h.m, 3)
					h.must(f.a.DotRow(0, f.x.Data, &m.residualWork.Data[0]))
					h.must(m.buildRowResidual(0))
				}
				h.budget.FailAfter(ordinal)
				var err error
				switch phase {
				case "residual":
					_, err = m.AMultXIsOff()
				case "find":
					_, err = m.FindXBySolving()
				case "inverse":
					err = m.CalculatePivotRowOfBInverse(1)
				case "pivot":
					err = m.CalculatePivotRowWhenInverseReady(1)
				case "diagnostic":
					var w bytes.Buffer
					err = m.printRowResidual(0, &w)
				}
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(direct, phase, ordinal, err)
					}
					if phase != "diagnostic" {
						if _, e = m.FindXBySolving(); e != ErrExactVectorPending {
							t.Fatal("stopped retry", e)
						}
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
			if !done || direct && failures == 0 {
				t.Fatal("physical campaign incomplete", direct, phase, failures)
			}
		}
	}
}

func TestLPRationalCoreResidualPivotGuards(t *testing.T) {
	for _, phase := range []string{"rhs", "size", "matrix-stopped", "stdout-closed", "find-no-lu", "inverse-no-lu", "inverse-tableau", "inverse-settings", "inverse-row", "pivot-index", "pivot-zero-skip", "pivot-matrix-row", "pivot-heading", "pivot-count"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyLU
			one := h.pair("1", "0")
			h.must(f.a.Set(0, 0, &one.X))
			h.must(f.a.Set(0, 1, &one.X))
			h.must(f.b.copySlot(&f.b.Data[0], &one.X))
			m, e := NewRationalCoreSolverBase(f.input())
			h.must(e)
			m.RowsNZ = make([]uint32, 1)
			var err error
			want := ErrBounds
			switch phase {
			case "rhs":
				owner := f.b.Data
				f.b.Data = nil
				_, err = m.AMultXIsOff()
				f.b.Data = owner
			case "size":
				owner := f.x.Data
				f.x.Data = nil
				_, err = m.AMultXIsOff()
				f.x.Data = owner
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "matrix-stopped":
				e = f.a.DotRow(99, f.x.Data, &one.X)
				if e == nil {
					t.Fatal("bad row accepted")
				}
				_, err = m.AMultXIsOff()
				want = ErrExactVectorPending
			case "stdout-closed":
				closed, e := os.CreateTemp(t.TempDir(), "g3-rational-residual-")
				h.must(e)
				h.must(closed.Close())
				old := os.Stdout
				os.Stdout = closed
				_, err = m.AMultXIsOff()
				os.Stdout = old
				h.must(os.Remove(closed.Name()))
				if err == nil {
					t.Fatal("closed stdout accepted")
				}
				want = err
			case "find-no-lu":
				h.must(m.Factorization.Close())
				m.Factorization = nil
				_, err = m.FindXBySolving()
			case "inverse-no-lu":
				h.must(m.Factorization.Close())
				m.Factorization = nil
				err = m.CalculatePivotRowOfBInverse(0)
				if m.PivotRowB1.Size() != 1 {
					t.Fatal("lost unit prefix")
				}
			case "inverse-tableau":
				f.settings.SimplexStrategy = StrategyTableauRows
				err = m.CalculatePivotRowOfBInverse(0)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "inverse-row":
				err = m.CalculatePivotRowOfBInverse(99)
			case "inverse-settings":
				m.Settings = nil
				err = m.CalculatePivotRowOfBInverse(0)
				want = nil
				if sourceDebugEnabled {
					want = ErrBounds
				}
			case "pivot-index":
				m.PivotRowB1.Index = []uint32{99}
				err = m.CalculatePivotRowWhenInverseReady(0)
			case "pivot-zero-skip":
				m.PivotRowB1.Index = []uint32{0}
				f.a.Rows[0][0].Column = 99
				err = m.CalculatePivotRowWhenInverseReady(0)
				want = nil
			case "pivot-matrix-row":
				h.must(m.PivotRowB1.resizeData(2))
				h.must(m.PivotRowB1.SetValue(&one.X, 1))
				err = m.CalculatePivotRowWhenInverseReady(0)
			case "pivot-heading":
				h.must(m.PivotRowB1.SetValue(&one.X, 0))
				f.a.Rows[0][0].Column = 99
				err = m.CalculatePivotRowWhenInverseReady(0)
			case "pivot-count":
				h.must(m.PivotRowB1.SetValue(&one.X, 0))
				err = m.CalculatePivotRowWhenInverseReady(99)
				lpSparseRationalCheck(t, h, &m.PivotRow.Data[1], big.NewRat(1, 1))
			}
			if err != want {
				t.Fatal(phase, err, want)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if _, e = m.FindXBySolving(); e != ErrExactVectorPending {
				t.Fatal("closed find", e)
			}
			if _, e = m.AMultXIsOff(); e != ErrExactVectorPending {
				t.Fatal("closed residual", e)
			}
			if e = m.CalculatePivotRowOfBInverse(0); e != ErrExactVectorPending {
				t.Fatal("closed inverse", e)
			}
			if e = m.CalculatePivotRowWhenInverseReady(0); e != ErrExactVectorPending {
				t.Fatal("closed pivot", e)
			}
		})
	}
}

func TestLPRationalCorePivotCancellationAndStaleResidual(t *testing.T) {
	for _, phase := range []string{"cancel", "stale", "empty"} {
		h := newLPPairHarness(t)
		rows := uint32(1)
		if phase == "empty" {
			rows = 0
		}
		f := lpNewRationalCoreFixture(t, h, rows, 2)
		f.settings.SimplexStrategy = StrategyLU
		one, two := h.pair("1", "0"), h.pair("2", "0")
		if rows != 0 {
			h.must(f.a.Set(0, 0, &one.X))
			h.must(f.b.copySlot(&f.b.Data[0], &two.X))
			if phase == "cancel" {
				h.must(f.a.Set(0, 1, &two.X))
				h.must(f.a.Set(0, 1, &h.pair("-2", "0").X))
			}
		}
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.RowsNZ = make([]uint32, rows)
		switch phase {
		case "cancel":
			h.must(m.CalculatePivotRowOfBInverse(0))
			h.must(m.CalculatePivotRowWhenInverseReady(0))
			if m.PivotRow.Size() != 0 || m.RowsNZ[0] != 0 {
				t.Fatal("exact cancellation retained membership", m.PivotRow.Index)
			}
		case "stale":
			// Change the real coefficient after LU construction, without
			// replacing the cached factorization. The actual residual query
			// must reject its reconstructed x rather than report a solve.
			h.must(h.m.SetInt32(&f.a.Rows[0][0].Value, 2))
			var ok bool
			var err error
			out := lpCaptureFeasibleStdout(t, func() { ok, err = m.FindXBySolving() })
			h.must(err)
			if ok || out != "precise x is off (m_b[0] = 2 left side = 4 delta = -2 iters = 0)\n" {
				t.Fatal("stale LU residual", ok, out)
			}
		case "empty":
			off, e := m.AMultXIsOff()
			h.must(e)
			if off {
				t.Fatal("empty matrix residual")
			}
			ok, e := m.FindXBySolving()
			h.must(e)
			if !ok {
				t.Fatal("empty solve")
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalCoreResidualPivotPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 2)
		f.settings.SimplexStrategy = StrategyLU
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		m, e := NewRationalCoreSolverBase(f.input())
		h.must(e)
		m.RowsNZ = make([]uint32, 1)
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
			off, e := m.AMultXIsOff()
			h.must(e)
			if off {
				t.Fatal("zero residual")
			}
			h.must(m.CalculatePivotRowOfBInverse(0))
			h.must(m.CalculatePivotRowWhenInverseReady(0))
			if m.calls.comparison != 0 {
				t.Fatal("adopted older comparison")
			}
		}
		h.must(m.Close())
		if older.Active && h.m.ComparisonInvocationState() != older {
			t.Fatal("consumed older comparison")
		}
		h.must(caller.cleanupRaw(h.m))
		h.must(caller.cleanupRoots(h.m))
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}
