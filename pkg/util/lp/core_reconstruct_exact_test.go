package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestLPPairCoreReconstructIndependentSystems(t *testing.T) {
	rng := rand.New(rand.NewPCG(9071, 719))
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		n := 1 + world%4
		f := lpNewPairCoreFixture(t, h, uint32(n), uint32(n+2))
		f.settings.SimplexStrategy = StrategyLU
		f.settings.DensityThreshold = 1e100
		if world%2 == 0 {
			f.settings.DensityThreshold = 0
		}
		perm := rng.Perm(n + 2)
		for i := range f.basis {
			f.basis[i] = uint32(perm[i])
		}
		x, y := make([]*big.Rat, n+2), make([]*big.Rat, n+2)
		for j := range x {
			x[j] = big.NewRat(int64(world%19+j-9), 7)
			y[j] = big.NewRat(int64(world%23-j-11), 13)
			if j == perm[n] || j == perm[n+1] {
				h.must(f.x.Data[j].Copy(h.m, h.pair(x[j].RatString(), y[j].RatString())))
			} else {
				h.must(f.x.Data[j].Copy(h.m, h.pair("97", "101")))
			}
		}
		expected := make([][2]*big.Rat, n)
		for i := 0; i < n; i++ {
			bx, by, rx, ry := new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)
			for k, j := range perm {
				a := big.NewRat(int64(rng.IntN(7)-3), 11)
				if k == i {
					a.SetInt64(int64(9 + n + i))
				}
				if a.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(a.RatString(), "0").X))
				}
				bx.Add(bx, new(big.Rat).Mul(a, x[j]))
				by.Add(by, new(big.Rat).Mul(a, y[j]))
				if k < n {
					rx.Add(rx, new(big.Rat).Mul(a, x[j]))
					ry.Add(ry, new(big.Rat).Mul(a, y[j]))
				}
			}
			expected[i] = [2]*big.Rat{rx, ry}
			h.must(f.b.Data[i].Copy(h.m, h.pair(bx.RatString(), by.RatString())))
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		rs := NewIndexedPairVector(h.m, uint32(n))
		h.must(m.RsMinusAnX(rs.Data))
		for i := range expected {
			h.check(&rs.Data[i], expected[i][0], expected[i][1])
		}
		if world%3 == 0 {
			h.must(m.CopyRsToXB(rs.Data))
			for i, j := range f.basis {
				h.check(&f.x.Data[j], expected[i][0], expected[i][1])
			}
		}
		if world%2 == 0 {
			var ok bool
			var err error
			out := lpCaptureFeasibleStdout(t, func() { ok, err = m.FindXBySolving() })
			h.must(err)
			if !ok || out != "" {
				t.Fatal(world, ok, out)
			}
		} else {
			h.must(m.SolveAxEqualsB())
			var off bool
			var err error
			out := lpCaptureFeasibleStdout(t, func() { off, err = m.AMultXIsOff() })
			h.must(err)
			if off || out != "" {
				t.Fatal(world, off, out)
			}
		}
		for j := range x {
			h.check(&f.x.Data[j], x[j], y[j])
		}
		if m.reconstructionWork != nil || m.residualWork != nil {
			t.Fatal("successful locals retained")
		}
		h.must(rs.Close())
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreReconstructLiveAliases(t *testing.T) {
	for _, op := range []string{"rs-prefix", "rs-suffix", "copy-swap"} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 3)
		for i := uint32(0); i < 2; i++ {
			h.must(f.a.Set(i, i, &h.pair("1", "0").X))
		}
		h.must(f.a.Set(0, 2, &h.pair("2", "0").X))
		h.must(f.a.Set(1, 2, &h.pair("5", "0").X))
		h.must(f.b.Data[0].Copy(h.m, h.pair("7", "0")))
		h.must(f.b.Data[1].Copy(h.m, h.pair("11", "0")))
		h.must(f.x.Data[0].Copy(h.m, h.pair("5", "0")))
		h.must(f.x.Data[1].Copy(h.m, h.pair("11", "0")))
		h.must(f.x.Data[2].Copy(h.m, h.pair("3", "0")))
		if op == "copy-swap" {
			f.basis = []uint32{1, 0}
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		switch op {
		case "rs-prefix":
			h.must(m.RsMinusAnX(f.x.Data))
			h.check(&f.x.Data[0], big.NewRat(1, 1), big.NewRat(0, 1))
			h.check(&f.x.Data[1], big.NewRat(-4, 1), big.NewRat(0, 1))
		case "rs-suffix":
			h.must(m.RsMinusAnX(f.x.Data[1:]))
			h.check(&f.x.Data[1], big.NewRat(95, 1), big.NewRat(0, 1))
			h.check(&f.x.Data[2], big.NewRat(-44, 1), big.NewRat(0, 1))
		case "copy-swap":
			h.must(m.CopyRsToXB(f.x.Data))
			h.check(&f.x.Data[0], big.NewRat(11, 1), big.NewRat(0, 1))
			h.check(&f.x.Data[1], big.NewRat(11, 1), big.NewRat(0, 1))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreReconstructStaleFactorizationReportsResidual(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 1, 1)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
	h.must(f.b.Data[0].Copy(h.m, h.pair("4", "0")))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	value, e := f.a.ColumnValue(f.a.Columns[0][0])
	h.must(e)
	h.must(h.m.SetInt32(value, 2))
	var ok bool
	var err error
	out := lpCaptureFeasibleStdout(t, func() { ok, err = m.FindXBySolving() })
	h.must(err)
	if ok || out != "precise x is off (m_b[0] = (4, 0) left side = (8, 0) delta = (-4, 0) iters = 0)\n" {
		t.Fatal(ok, out)
	}
	h.check(&f.x.Data[0], big.NewRat(4, 1), big.NewRat(0, 1))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairCoreReconstructGuardsAndPrefixes(t *testing.T) {
	for _, op := range []string{"rs-short", "rhs-short", "heading-short", "x-short", "copy-basis", "copy-short", "missing-lu"} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 1, 2)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
		h.must(f.b.Data[0].Copy(h.m, h.pair("7", "0")))
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		rs := NewIndexedPairVector(h.m, 1)
		var err error
		switch op {
		case "rs-short":
			err = m.RsMinusAnX(nil)
		case "rhs-short":
			h.must(f.b.resizeData(0))
			err = m.RsMinusAnX(rs.Data)
		case "heading-short":
			f.heading = nil
			err = m.RsMinusAnX(rs.Data)
			h.check(&rs.Data[0], big.NewRat(7, 1), big.NewRat(0, 1))
		case "x-short":
			h.must(f.x.resizeData(1))
			err = m.RsMinusAnX(rs.Data)
		case "copy-basis":
			f.basis = nil
			err = m.CopyRsToXB(rs.Data)
		case "copy-short":
			err = m.CopyRsToXB(nil)
		case "missing-lu":
			err = m.SolveAxEqualsB()
			if m.reconstructionWork == nil {
				t.Fatal("missing partial RHS owner")
			}
			h.check(&m.reconstructionWork.Data[0], big.NewRat(7, 1), big.NewRat(0, 1))
		}
		if err != ErrBounds {
			t.Fatal(op, err)
		}
		if e = m.RsMinusAnX(rs.Data); e != ErrExactVectorPending {
			t.Fatal("lost RHS stop", e)
		}
		if e = m.CopyRsToXB(rs.Data); e != ErrExactVectorPending {
			t.Fatal("lost copy stop", e)
		}
		if e = m.SolveAxEqualsB(); e != ErrExactVectorPending {
			t.Fatal("lost stopped reconstruct", e)
		}
		if _, e = m.FindXBySolving(); e != ErrExactVectorPending {
			t.Fatal(e)
		}
		h.must(rs.Close())
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPairCoreReconstructPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"rs", "solve", "find", "residual"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1600; ordinal++ {
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
					f.settings.DensityThreshold = 1e100
					huge := strings.Repeat("1234567890", 12)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &h.pair(fmt.Sprint(i+2)+"/7", "0").X))
						h.must(f.a.Set(i, 2, &h.pair(huge+fmt.Sprint(i+1)+"/11", "0").X))
						h.must(f.b.Data[i].Copy(h.m, h.pair(huge+fmt.Sprint(i+3)+"/13", huge+fmt.Sprint(i+5)+"/17")))
					}
					h.must(f.x.Data[2].Copy(h.m, h.pair(huge+"7/19", huge+"9/23")))
					m, e := NewPairPrimalCoreSolver(f.input())
					h.must(e)
					rs := NewIndexedPairVector(h.m, 2)
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "rs":
						err = m.RsMinusAnX(rs.Data)
					case "solve":
						err = m.SolveAxEqualsB()
					case "find":
						lpCaptureFeasibleStdout(t, func() { _, err = m.FindXBySolving() })
					case "residual":
						lpCaptureFeasibleStdout(t, func() { _, err = m.AMultXIsOff() })
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.PairCoreSolverBase.calls.pending() && f.a.calls.pending() {
							t.Fatal("adopted matrix invocation")
						}
						if m.Factorization.ready() == ErrLUStopped {
							if m.PairCoreSolverBase.calls.pending() {
								t.Fatal("adopted LU invocation")
							}
							for i := 0; i < 2; i++ {
								h.check(&f.x.Data[i], big.NewRat(0, 1), big.NewRat(0, 1))
							}
						}
						if e = m.SolveAxEqualsB(); e != ErrExactVectorPending {
							t.Fatal("lost stopped owner", e)
						}
					}
					h.must(m.Close())
					h.must(rs.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPPairCoreReconstructStoppedLUKeepsRHS(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewPairCoreFixture(t, h, 1, 1)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, &h.pair("2", "0").X))
	h.must(f.b.Data[0].Copy(h.m, h.pair("7", "3")))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	if e = m.Factorization.SolveBy(nil); e == nil {
		t.Fatal("short RHS accepted")
	}
	if e = m.SolveAxEqualsB(); e != ErrLUStopped {
		t.Fatal(e)
	}
	if m.reconstructionWork == nil || m.PairCoreSolverBase.calls.pending() {
		t.Fatal("lost partial RHS or adopted LU")
	}
	h.check(&m.reconstructionWork.Data[0], big.NewRat(7, 1), big.NewRat(3, 1))
	h.check(&f.x.Data[0], big.NewRat(0, 1), big.NewRat(0, 1))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPPairCoreReconstructPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"rs", "solve", "find", "residual"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewPairCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("2", "0").X))
				h.must(f.a.Set(0, 1, &h.pair("3", "0").X))
				h.must(f.b.Data[0].Copy(h.m, h.pair("7", "3")))
				m, e := NewPairPrimalCoreSolver(f.input())
				h.must(e)
				rs := NewIndexedPairVector(h.m, 1)
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
					switch op {
					case "rs":
						err = m.RsMinusAnX(rs.Data)
					case "solve":
						err = m.SolveAxEqualsB()
					case "find":
						lpCaptureFeasibleStdout(t, func() { _, err = m.FindXBySolving() })
					default:
						lpCaptureFeasibleStdout(t, func() { _, err = m.AMultXIsOff() })
					}
					h.must(err)
					if m.PairCoreSolverBase.calls.comparison != 0 || f.a.calls.comparison != 0 {
						t.Fatal("adopted caller")
					}
				}
				h.must(m.Close())
				h.must(rs.Close())
				f.close()
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("destroyed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal(op, "no actual older comparison")
		}
	}
}
