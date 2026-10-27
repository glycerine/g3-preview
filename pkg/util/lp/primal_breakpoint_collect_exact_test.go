package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactBreakpointCollectIndependentDeltaOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for world := 0; world < 600; world++ {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "2", "3", false)
		m := s.Core.RPrimal
		h.must(m.BreakpointQueue.Resize(1))
		m.SignOfEnteringDelta = int32(world%3 - 1)
		typ := BreakpointType((world / 3) % 3)
		xx, xy := big.NewRat(int64(world%17-8), 7), big.NewRat(int64(world%19-9), 11)
		bx, by := big.NewRat(int64(world%23-11), 13), big.NewRat(int64(world%29-14), 17)
		d := big.NewRat(int64(world%11+1), 19)
		if world%2 == 0 {
			d.Neg(d)
		}
		if world%7 == 0 {
			xx.Set(bx)
			xy.Set(by)
			if world%5 == 0 {
				d.SetInt64(0)
			}
		}
		if world%5 == 1 {
			xx.Set(bx)
		}
		x, bound, scalar := h.pair(xx.RatString(), xy.RatString()), h.pair(bx.RatString(), by.RatString()), h.pair(d.RatString(), "0")
		qx, qy := new(big.Rat).Sub(xx, bx), new(big.Rat).Sub(xy, by)
		add := false
		if qx.Sign() == 0 && qy.Sign() == 0 {
			same := (d.Sign() > 0 && m.SignOfEnteringDelta > 0) || (d.Sign() < 0 && m.SignOfEnteringDelta < 0)
			add = typ == BreakpointFixed || (typ == BreakpointLow && same) || (typ == BreakpointUpper && !same)
		} else {
			qx.Quo(qx, d)
			qy.Quo(qy, d)
			sign := qx.Sign()
			if sign == 0 {
				sign = qy.Sign()
			}
			add = (sign > 0 && m.SignOfEnteringDelta > 0) || (sign < 0 && m.SignOfEnteringDelta < 0)
		}
		h.must(m.TryAddBreakpoint(2, x, &scalar.X, typ, bound))
		wantCount := 0
		if add {
			wantCount = 1
		}
		if len(m.Breakpoints) != wantCount || int(m.BreakpointQueue.Size()) != wantCount {
			t.Fatal(world, add, len(m.Breakpoints), m.BreakpointQueue.Size())
		}
		if add {
			h.check(m.Breakpoints[0].Delta, qx, qy)
			if m.Breakpoints[0].Column != 2 || m.Breakpoints[0].Type != typ {
				t.Fatal("metadata")
			}
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBreakpointCollectIndependentModelOracle(t *testing.T) {
	for world := 0; world < 200; world++ {
		h := newLPPairHarness(t)
		coefficient := int64(world%7 - 3)
		s := lpNewRemovalModel(t, h, fmt.Sprint(coefficient), "3", true)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(2))
		h.must(m.BreakpointQueue.Resize(8))
		m.SignOfEnteringDelta = int32(world%3 - 1)
		// The actual tableau basis is identity; both entering coefficients are -a.
		d := big.NewRat(-coefficient, 1)
		indices := []uint32{0, 1}
		if world%2 == 0 {
			indices = []uint32{1, 0}
		}
		for _, i := range indices {
			v := h.pair(d.RatString(), "0")
			h.must(m.Ed.SetValue(&v.X, i))
		}
		type expected struct {
			j    uint32
			typ  BreakpointType
			x, y *big.Rat
		}
		var want []expected
		for _, i := range indices {
			j := (*m.Basis)[i]
			kind := ColumnType((world + int(i)*2) % 5)
			h.must(s.Bounds.Types.Set(j, kind))
			xx, xy := big.NewRat(int64((world+int(i))%9-4), 3), big.NewRat(int64(world%5-2), 7)
			lowx, lowy, upx, upy := big.NewRat(-2, 3), big.NewRat(-1, 7), big.NewRat(5, 3), big.NewRat(2, 7)
			if kind == ColumnFixed {
				upx.Set(lowx)
				upy.Set(lowy)
			}
			if world%5 == 0 {
				xx.Set(lowx)
				xy.Set(lowy)
			}
			h.must((*m.X)[j].Copy(h.m, h.pair(xx.RatString(), xy.RatString())))
			h.must((*m.LowBounds)[j].Copy(h.m, h.pair(lowx.RatString(), lowy.RatString())))
			h.must((*m.UpperBounds)[j].Copy(h.m, h.pair(upx.RatString(), upy.RatString())))
			add := func(bx, by *big.Rat, typ BreakpointType) {
				if d.Sign() == 0 {
					return
				}
				qx, qy := new(big.Rat).Sub(xx, bx), new(big.Rat).Sub(xy, by)
				include := false
				if qx.Sign() == 0 && qy.Sign() == 0 {
					same := (d.Sign() > 0 && m.SignOfEnteringDelta > 0) || (d.Sign() < 0 && m.SignOfEnteringDelta < 0)
					include = typ == BreakpointFixed || (typ == BreakpointLow && same) || (typ == BreakpointUpper && !same)
				} else {
					qx.Quo(qx, d)
					qy.Quo(qy, d)
					sign := qx.Sign()
					if sign == 0 {
						sign = qy.Sign()
					}
					include = (sign > 0 && m.SignOfEnteringDelta > 0) || (sign < 0 && m.SignOfEnteringDelta < 0)
				}
				if include {
					want = append(want, expected{j, typ, qx, qy})
				}
			}
			switch kind {
			case ColumnFixed:
				add(lowx, lowy, BreakpointFixed)
			case ColumnBoxed:
				add(lowx, lowy, BreakpointLow)
				add(upx, upy, BreakpointUpper)
			case ColumnLowBound:
				add(lowx, lowy, BreakpointLow)
			case ColumnUpperBound:
				add(upx, upy, BreakpointUpper)
			}
		}
		enteringKind := ColumnFree
		if world%2 == 0 {
			enteringKind = ColumnBoxed
			h.must((*m.LowBounds)[0].Copy(h.m, h.pair("-1/3", "-1/7")))
			h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("2/3", "3/7")))
			x, y := big.NewRat(1, 1), big.NewRat(4, 7)
			typ := BreakpointUpper
			if m.SignOfEnteringDelta < 0 {
				x.Neg(x)
				y.Neg(y)
				typ = BreakpointLow
			}
			want = append(want, expected{0, typ, x, y})
		}
		h.must(s.Bounds.Types.Set(0, enteringKind))
		// Existing records must be destroyed before collection starts.
		h.must(m.AddBreakpoint(1, h.pair("99", "101"), BreakpointFixed))
		h.must(m.FillBreakpointsArray(0))
		if len(m.Breakpoints) != len(want) || int(m.BreakpointQueue.Size()) != len(want) {
			t.Fatal(world, len(m.Breakpoints), len(want))
		}
		for k, w := range want {
			b := m.Breakpoints[k]
			if b.Column != w.j || b.Type != w.typ {
				t.Fatal("source order", world, k)
			}
			h.check(b.Delta, w.x, w.y)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBreakpointCollectPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"ratio", "row-boxed", "fill-positive", "fill-negative"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpNewRemovalModel(t, h, "-2/3", "3", false)
					m := s.Core.RPrimal
					h.must(m.Ed.Resize(1))
					h.must(m.BreakpointQueue.Resize(4))
					huge := strings.Repeat("1234567890", 12)
					x, bound, d := h.pair("-"+huge+"1/7", huge+"3/11"), h.pair(huge+"5/13", huge+"7/17"), h.pair("2/3", "0")
					m.SignOfEnteringDelta = -1
					if op == "row-boxed" {
						h.must(s.Bounds.Types.Set(2, ColumnBoxed))
						h.must((*m.X)[2].Copy(h.m, x))
						h.must((*m.LowBounds)[2].Copy(h.m, bound))
						h.must((*m.UpperBounds)[2].Copy(h.m, h.pair(huge+"9/19", huge+"1/23")))
						h.must(m.Ed.SetValue(&d.X, 0))
					}
					if strings.HasPrefix(op, "fill") {
						h.must(s.Bounds.Types.Set(0, ColumnBoxed))
						h.must((*m.LowBounds)[0].Copy(h.m, x))
						h.must((*m.UpperBounds)[0].Copy(h.m, bound))
						if op == "fill-positive" {
							m.SignOfEnteringDelta = 1
						}
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "ratio":
						err = m.TryAddBreakpoint(2, x, &d.X, BreakpointLow, bound)
					case "row-boxed":
						err = m.TryAddBreakpointInRow(0)
					default:
						err = m.FillBreakpointsArray(0)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.BreakpointQueue.calls.pending() && m.calls.pending() {
							t.Fatal("adopted queue invocation")
						}
						if op == "row-boxed" && m.BreakpointQueue.calls.comparison != 0 && len(m.Breakpoints) != 2 {
							t.Fatal("lost boxed upper append prefix")
						}
						if e := m.ClearBreakpoints(); e != ErrExactVectorPending {
							t.Fatal("lost stopped collector", e)
						}
					}
					h.must(s.Close())
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

func TestLPExactBreakpointCollectGuardsAndPrefixes(t *testing.T) {
	for _, phase := range []string{"nil", "row-index", "unprepared-ed", "unknown", "zero-row", "entering-index"} {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "-2/3", "3", false)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(1))
		h.must(m.BreakpointQueue.Resize(4))
		m.SignOfEnteringDelta = 1
		d := h.pair("2/3", "0")
		h.must(m.Ed.SetValue(&d.X, 0))
		h.must(s.Bounds.Types.Set(2, ColumnLowBound))
		h.must((*m.X)[2].Copy(h.m, h.pair("1", "0")))
		want := ErrBounds
		var err error
		switch phase {
		case "nil":
			err = m.TryAddBreakpoint(2, nil, &d.X, BreakpointLow, h.pair("0", "0"))
		case "row-index":
			if sourceDebugEnabled {
				want = ErrAssertion
			}
			err = m.TryAddBreakpointInRow(99)
		case "unprepared-ed":
			h.must(m.Ed.Resize(0))
			err = m.TryAddBreakpointInRow(0)
		case "unknown", "zero-row":
			h.must(s.Bounds.Types.Set(2, ColumnType(99)))
			want = nil
			if phase == "zero-row" {
				h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
			} else if sourceDebugEnabled {
				want = ErrAssertion
			}
			err = m.TryAddBreakpointInRow(0)
		case "entering-index":
			h.must(m.AddBreakpoint(0, h.pair("99", "101"), BreakpointFixed))
			err = m.FillBreakpointsArray(99)
			if len(m.Breakpoints) != 1 || m.Breakpoints[0].Column != 2 {
				t.Fatal("lost completed row prefix")
			}
		}
		if err != want {
			t.Fatal(phase, err, want)
		}
		if err != nil {
			if e := m.TryAddBreakpointInRow(0); e != ErrExactVectorPending {
				t.Fatal("lost row stop", e)
			}
			if e := m.TryAddBreakpoint(2, h.pair("0", "0"), &d.X, BreakpointLow, h.pair("0", "0")); e != ErrExactVectorPending {
				t.Fatal("lost delta stop", e)
			}
			if e := m.FillBreakpointsArray(0); e != ErrExactVectorPending {
				t.Fatal("lost fill stop", e)
			}
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBreakpointCollectPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"ratio", "row", "fill"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				s := lpNewRemovalModel(t, h, "-3/7", "3", false)
				var e error
				m := s.Core.RPrimal
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.must(m.Ed.Resize(1))
				h.must(h.m.SetString(&m.Ed.Data[0], "3/7"))
				m.SignOfEnteringDelta = -1
				delta := h.pair("5/13", "7/17")
				h.must(s.Bounds.Types.Set(2, ColumnBoxed))
				h.must((*m.X)[2].Copy(h.m, h.pair("-1", "1/11")))
				h.must((*m.UpperBounds)[2].Copy(h.m, h.pair("1", "0")))
				h.must(m.Ed.SetValue(&m.Ed.Data[0], 0))
				h.must(m.BreakpointQueue.Resize(4))
				h.must(m.AddBreakpoint(2, h.pair("1/3", "2/7"), BreakpointLow))
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
					case "ratio":
						err = m.TryAddBreakpoint(2, &(*m.X)[2], &m.Ed.Data[0], BreakpointLow, delta)
					case "row":
						err = m.TryAddBreakpointInRow(0)
					case "fill":
						err = m.FillBreakpointsArray(0)
					}
					if m.calls.comparison != 0 || m.BreakpointQueue.calls.comparison != 0 {
						t.Fatal("adopted older comparison", err)
					}
				}
				h.must(s.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
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
