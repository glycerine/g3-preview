package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPExactBoundStepIndependentFlipOracle(t *testing.T) {
	h := newLPPairHarness(t)
	worlds := 0
	for _, typ := range []ColumnType{ColumnFree, ColumnFixed, ColumnLowBound, ColumnUpperBound, ColumnBoxed} {
		for _, direction := range []int32{-1, 0, 1} {
			for world := 0; world < 32; world++ {
				s, e := NewLARSolver(h.m)
				h.must(e)
				_, e = s.AddVar(31)
				h.must(e)
				m := s.Core.RPrimal
				m.SignOfEnteringDelta = direction
				h.must(s.Bounds.Types.Set(0, typ))
				lowX, lowY := big.NewRat(-3, 7), big.NewRat(1, 13)
				upX, upY := big.NewRat(5, 11), big.NewRat(-1, 17)
				h.must(s.Bounds.Low.Set(0, h.pair(lowX.RatString(), lowY.RatString())))
				h.must(s.Bounds.Upper.Set(0, h.pair(upX.RatString(), upY.RatString())))
				x, y := big.NewRat(int64(world%5-2), 13), big.NewRat(int64(world%7-3), 19)
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(x.RatString(), y.RatString())))
				thetaX, thetaY := big.NewRat(int64(world%9), 17), big.NewRat(int64(world%11-5), 23)
				theta := h.pair(thetaX.RatString(), thetaY.RatString())
				out := h.pair("37/41", "43/47")
				allowed := typ == ColumnBoxed || (typ == ColumnUpperBound && direction > 0) || (typ == ColumnLowBound && direction < 0)
				dx, dy := big.NewRat(37, 41), big.NewRat(43, 47)
				if allowed {
					if direction > 0 {
						dx.Sub(upX, x)
						dy.Sub(upY, y)
					} else {
						dx.Sub(x, lowX)
						dy.Sub(y, lowY)
					}
				}
				unlimited := world%2 == 0
				cmp := dx.Cmp(thetaX)
				if cmp == 0 {
					cmp = dy.Cmp(thetaY)
				}
				want := allowed && (unlimited || cmp <= 0)
				got, err := m.TryJumpToAnotherBoundOnEntering(0, theta, out, unlimited)
				h.must(err)
				if got != want {
					t.Fatal(typ, direction, world, got, want)
				}
				h.check(out, dx, dy)
				// The separate unlimited selector only accepts boxed columns.
				h.must(out.Copy(h.m, h.pair("37/41", "43/47")))
				got, err = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out)
				h.must(err)
				if got != (typ == ColumnBoxed) {
					t.Fatal("unlimited selector", typ, got)
				}
				if typ == ColumnBoxed {
					h.check(out, dx, dy)
				} else {
					h.check(out, big.NewRat(37, 41), big.NewRat(43, 47))
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
				worlds++
			}
		}
	}
	if worlds != 480 {
		t.Fatal(worlds)
	}
}

func TestLPExactBoundStepIndependentLimitAndAliases(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 160; world++ {
		s, e := NewLARSolver(h.m)
		h.must(e)
		m := s.Core.RPrimal
		ax, ay := big.NewRat(int64(world%17-8), 11), big.NewRat(int64(world%19-9), 13)
		bx, by := big.NewRat(int64(world%13-6), 17), big.NewRat(int64(world%23-11), 19)
		lim, theta := h.pair(ax.RatString(), ay.RatString()), h.pair(bx.RatString(), by.RatString())
		unlimited := world%3 == 0
		wantX, wantY := bx, by
		cmp := ax.Cmp(bx)
		if cmp == 0 {
			cmp = ay.Cmp(by)
		}
		if unlimited || cmp <= 0 {
			wantX, wantY = ax, ay
		}
		h.must(m.LimitTheta(lim, theta, &unlimited))
		if unlimited {
			t.Fatal("unlimited not cleared")
		}
		h.check(theta, wantX, wantY)
		h.must(m.LimitTheta(theta, theta, &unlimited))
		h.check(theta, wantX, wantY)
		_, e = s.AddVar(31)
		h.must(e)
		h.must(s.Bounds.Types.Set(0, ColumnBoxed))
		m.SignOfEnteringDelta = 1
		h.must(s.Bounds.Upper.Set(0, h.pair("3/7", "1/11")))
		h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("1/13", "-1/17")))
		// Assigning t before testing theta makes a shared output/limit compare equal.
		h.must(theta.Copy(h.m, h.pair("-100", "0")))
		jump, e := m.TryJumpToAnotherBoundOnEntering(0, theta, theta, false)
		h.must(e)
		if !jump {
			t.Fatal("source theta/output alias must accept equality")
		}
		h.check(theta, new(big.Rat).Sub(big.NewRat(3, 7), big.NewRat(1, 13)), new(big.Rat).Add(big.NewRat(1, 11), big.NewRat(1, 17)))
		// Output may be the actual source bound; the subtraction reads it first.
		jump, e = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, &s.Bounds.Upper.values[0])
		h.must(e)
		if !jump {
			t.Fatal("bound alias")
		}
		h.check(&s.Bounds.Upper.values[0], new(big.Rat).Sub(big.NewRat(3, 7), big.NewRat(1, 13)), new(big.Rat).Add(big.NewRat(1, 11), big.NewRat(1, 17)))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBoundStepLeavingRowsIndependentMembership(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 160; world++ {
		s := lpNewLARLUPreparedModel(t, h, lpExactIdentity(4))
		m := s.Core.RPrimal
		low, upper := h.pair("-3/7", "1/13"), h.pair("5/11", "-1/17")
		expected := make([]*RationalPair, 4)
		for j := uint32(0); j < 4; j++ {
			kind := []ColumnType{ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed}[(int(j)+world)%4]
			h.must(s.Bounds.Types.Set(j, kind))
			h.must(s.Bounds.Low.Set(j, low))
			h.must(s.Bounds.Upper.Set(j, upper))
			if kind == ColumnFixed {
				h.must(s.Bounds.Low.Set(j, upper))
			}
			if kind == ColumnLowBound || (kind == ColumnBoxed && world%2 == 0) {
				h.must(s.Core.RX.Data[j].Copy(h.m, h.pair("-3/7", "0")))
				expected[j] = &s.Bounds.Low.values[j]
			} else {
				h.must(s.Core.RX.Data[j].Copy(h.m, h.pair("5/11", "0")))
				expected[j] = &s.Bounds.Upper.values[j]
			}
		}
		order := []uint32{3, 1, 2, 0}
		if world%2 == 0 {
			order = []uint32{2, 0, 3, 1}
		}
		for _, j := range order {
			h.must(m.InfSet.Insert(j))
		}
		out := h.pair("97", "101")
		for want := int32(0); want < 4; want++ {
			before := append([]uint32(nil), m.InfSet.Indices()...)
			got, e := m.FindLeavingTableauRows(out)
			h.must(e)
			if got != want || !reflect.DeepEqual(before, m.InfSet.Indices()) {
				t.Fatal(world, got, want, m.InfSet.Indices())
			}
			equal, e := out.Equal(h.m, expected[got])
			h.must(e)
			if !equal {
				t.Fatal("leaving target", world, got)
			}
			m.InfSet.Erase(uint32(got))
		}
		h.must(out.Copy(h.m, h.pair("97", "101")))
		got, e := m.FindLeavingTableauRows(out)
		h.must(e)
		if got != -1 {
			t.Fatal("empty membership")
		}
		h.check(out, big.NewRat(97, 1), big.NewRat(101, 1))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBoundStepPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"jump", "unlimited", "limit", "limit-unlimited", "leaving", "leaving-boxed"} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s, e := NewLARSolver(h.m)
					h.must(e)
					_, e = s.AddVar(31)
					h.must(e)
					m := s.Core.RPrimal
					huge := strings.Repeat("1234567890", 100)
					v := h.pair(huge+"1/7", huge+"3/11")
					lim := h.pair(huge+"5/13", huge+"7/17")
					out := h.pair("0", "0")
					m.SignOfEnteringDelta = 1
					h.must(s.Bounds.Types.Set(0, ColumnBoxed))
					h.must(s.Bounds.Upper.Set(0, v))
					if op == "leaving" || op == "leaving-boxed" {
						kind := ColumnUpperBound
						if op == "leaving-boxed" {
							kind = ColumnBoxed
						}
						h.must(s.Bounds.Types.Set(0, kind))
						h.must(m.InfSet.Insert(0))
						h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(huge+"9/3", huge+"11/5")))
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "jump":
						_, err = m.TryJumpToAnotherBoundOnEntering(0, lim, out, false)
					case "unlimited":
						_, err = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out)
					case "limit":
						unlimited := false
						err = m.LimitTheta(lim, v, &unlimited)
					case "limit-unlimited":
						unlimited := true
						err = m.LimitTheta(lim, out, &unlimited)
						if err != nil && !unlimited {
							t.Fatal("unlimited cleared before completed assignment")
						}
					case "leaving", "leaving-boxed":
						_, err = m.FindLeavingTableauRows(out)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.PairCoreSolverBase.calls.pending() && m.calls.pending() {
							t.Fatal("adopted base comparison")
						}
						if _, e = m.FindLeavingTableauRows(out); e != ErrExactVectorPending {
							t.Fatal("lost stopped primal", e)
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
				t.Fatal("incomplete campaign", direct, op, success, failures)
			}
		}
	}
}

func TestLPExactBoundStepSourcePrefixesAndGuards(t *testing.T) {
	for _, op := range []string{"nil-limit", "nil-jump", "column", "extent", "nil-theta", "negative", "negative-rejected", "unlimited-negative", "free-nil", "leaving-feasible", "leaving-kind", "leaving-nil", "leaving-boxed-low", "leaving-upper-extent", "leaving-low-extent", "leaving-boxed-X", "leaving-boxed-low-extent", "leaving-type", "jump-column"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s, e := NewLARSolver(h.m)
			h.must(e)
			_, e = s.AddVar(31)
			h.must(e)
			m := s.Core.RPrimal
			h.must(s.Bounds.Types.Set(0, ColumnBoxed))
			h.must(s.Bounds.Low.Set(0, h.pair("0", "0")))
			h.must(s.Bounds.Upper.Set(0, h.pair("2", "0")))
			m.SignOfEnteringDelta = 1
			theta, out := h.pair("1", "0"), h.pair("37", "41")
			var err error
			want := ErrBounds
			switch op {
			case "jump-column":
				_, err = m.TryJumpToAnotherBoundOnEntering(99, theta, out, false)
			case "leaving-upper-extent", "leaving-low-extent", "leaving-boxed-low-extent", "leaving-boxed-X", "leaving-type":
				h.must(m.InfSet.Insert(0))
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("-1", "0")))
				switch op {
				case "leaving-upper-extent":
					h.must(s.Bounds.Types.Set(0, ColumnUpperBound))
					saved := s.Bounds.Upper.values
					s.Bounds.Upper.values = nil
					_, err = m.FindLeavingTableauRows(out)
					s.Bounds.Upper.values = saved
				case "leaving-low-extent", "leaving-boxed-low-extent":
					if op == "leaving-low-extent" {
						h.must(s.Bounds.Types.Set(0, ColumnLowBound))
					}
					saved := s.Bounds.Low.values
					s.Bounds.Low.values = nil
					_, err = m.FindLeavingTableauRows(out)
					s.Bounds.Low.values = saved
				case "leaving-boxed-X":
					saved := s.Core.RX.Data
					s.Core.RX.Data = nil
					_, err = m.FindLeavingTableauRows(out)
					s.Core.RX.Data = saved
				case "leaving-type":
					saved := s.Bounds.Types.values
					s.Bounds.Types.values = nil
					_, err = m.FindLeavingTableauRows(out)
					s.Bounds.Types.values = saved
				}
			case "nil-limit":
				unlimited := true
				err = m.LimitTheta(nil, out, &unlimited)
			case "nil-jump":
				_, err = m.TryJumpToAnotherBoundOnEntering(0, theta, nil, true)
			case "column":
				_, err = m.TryJumpToAnotherBoundOnEnteringUnlimited(99, out)
			case "extent":
				saved := s.Bounds.Upper.values
				s.Bounds.Upper.values = nil
				_, err = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out)
				s.Bounds.Upper.values = saved
			case "nil-theta":
				_, err = m.TryJumpToAnotherBoundOnEntering(0, nil, out, false)
				h.check(out, big.NewRat(2, 1), big.NewRat(0, 1))
			case "negative", "negative-rejected", "unlimited-negative":
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("3", "0")))
				want = nil
				if op == "unlimited-negative" {
					jump, e := m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out)
					err = e
					if !jump {
						t.Fatal("literal unlimited negative step")
					}
				} else {
					if op == "negative-rejected" {
						h.must(theta.Copy(h.m, h.pair("-2", "0")))
					}
					jump, e := m.TryJumpToAnotherBoundOnEntering(0, theta, out, false)
					err = e
					if op == "negative-rejected" {
						if jump {
							t.Fatal("negative rejected step")
						}
					} else if sourceDebugEnabled {
						want = ErrAssertion
					} else if !jump {
						t.Fatal("native negative accepted step")
					}
				}
				h.check(out, big.NewRat(-1, 1), big.NewRat(0, 1))
			case "free-nil":
				h.must(s.Bounds.Types.Set(0, ColumnFree))
				jump, e := m.TryJumpToAnotherBoundOnEntering(0, nil, nil, false)
				err = e
				want = nil
				if jump {
					t.Fatal("free flip")
				}
			case "leaving-feasible":
				h.must(m.InfSet.Insert(0))
				_, err = m.FindLeavingTableauRows(out)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "leaving-kind":
				h.must(s.Bounds.Types.Set(0, ColumnType(99)))
				h.must(m.InfSet.Insert(0))
				_, err = m.FindLeavingTableauRows(out)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					h.check(out, big.NewRat(0, 1), big.NewRat(0, 1))
				}
			case "leaving-nil":
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("3", "0")))
				h.must(m.InfSet.Insert(0))
				_, err = m.FindLeavingTableauRows(nil)
			case "leaving-boxed-low":
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("-1", "0")))
				h.must(m.InfSet.Insert(0))
				j, e := m.FindLeavingTableauRows(&s.Bounds.Low.values[0])
				err = e
				want = nil
				if j != 0 {
					t.Fatal("low target alias")
				}
			}
			if err != want {
				t.Fatal(op, err, want)
			}
			if err != nil {
				unlimited := false
				for _, call := range []func() error{func() error { return m.LimitTheta(theta, out, &unlimited) }, func() error { _, e := m.TryJumpToAnotherBoundOnEntering(0, theta, out, false); return e }, func() error { _, e := m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out); return e }, func() error { _, e := m.FindLeavingTableauRows(out); return e }} {
					if e := call(); e != ErrExactVectorPending {
						t.Fatal("lost stopped primal", e)
					}
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactBoundStepPreservesOlderComparisonOwner(t *testing.T) {
	for _, op := range []string{"jump", "unlimited", "limit", "leaving"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, e := NewLARSolver(h.m)
				h.must(e)
				_, e = s.AddVar(31)
				h.must(e)
				m := s.Core.RPrimal
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				lim := h.pair("17", "19")
				out := h.pair("23", "29")
				h.must(s.Bounds.Types.Set(0, ColumnUpperBound))
				h.must(s.Bounds.Upper.Set(0, v))
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(strings.Repeat("9876543210", 100)+"/3", "0")))
				m.SignOfEnteringDelta = 1
				if op == "unlimited" {
					h.must(s.Bounds.Types.Set(0, ColumnBoxed))
				}
				if op == "leaving" {
					h.must(m.InfSet.Insert(0))
				}
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
					case "jump":
						_, err = m.TryJumpToAnotherBoundOnEntering(0, lim, out, false)
					case "unlimited":
						_, err = m.TryJumpToAnotherBoundOnEnteringUnlimited(0, out)
					case "limit":
						unlimited := false
						err = m.LimitTheta(v, lim, &unlimited)
					case "leaving":
						_, err = m.FindLeavingTableauRows(out)
					}
					if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
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
